// Copyright 2010 Adabra Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Stag generates static structs from struct tags.
//
// Given a struct like:
//
//    type Foo struct {
//       Name string `json:"theName" db:"the_name"`
//       Flavor string `json:"yummyFlavor" db:"mmm_flavor"`
//    }
//
// A cmd of
//   stag -source=path/to/file.go -tags=json,db
// would produce two files:
//   - path/to/file.stag_json.go
//   - path/to/file.stag_db.go
//
// With the output being:
//    //file.stag_json.go
//    var Foo_JSON = struct{
//        Name string
//        Flavor string
//    }{
//        Name: "theName",
//        Flavor: "yummyFlavor",
//    }
//
//    //file.stag_db.go
//    var Foo_DB = struct{
//        Name string
//        Flavor string
//    }{
//        Name: "the_name",
//        Flavor: "mmm_flavor",
//    }
package main

// Order of features to tackle:
//TODO(BDG): Support single file target
//TODO(BDG): Support embedded struct fields
//TODO(BDG): Support pkg or single file target, specifying types (inferring files if in pkg mode)
//TODO(BDG): Copyright file

import (
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/bradleygore/stag/model"
)

// cmd flags
var (
	source      = flag.String("source", "", "singular source file or directory to process")
	tagsArg     = flag.String("tags", "", "comma-separated set of tags to acquire static naming for")
	outType     = flag.String("out", "file", "output type; file | stdout; defaults to file (remains as file for pkg-wide processing)")
	showVersion = flag.Bool("version", false, "Print version.")
	showHelp    = flag.Bool("help", false, "show help")
	verbose     = flag.Bool("v", false, "verbose output")
)

// regex
var (
	rxIsGoFile   = regexp.MustCompile(".go$")
	rxIsStagFile = regexp.MustCompile(".stag-.*.go$")
)

func main() {
	flag.Usage = func() {
		printUsage(true)
	}
	flag.Parse()

	if *showVersion {
		printVersion()
		return
	}

	if *showHelp {
		printUsage(false)
		return
	}

	if len(*tagsArg) == 0 {
		printUsage(true)
		log.Fatal("tags is required")
	}
	tags := strings.Split(*tagsArg, ",")

	if *source == "" {
		printUsage(true)
		log.Fatal("source is required")
	}

	files := model.Files{}
	fs := token.NewFileSet()

	if rxIsGoFile.MatchString(*source) {
		if rxIsStagFile.MatchString(*source) {
			log.Fatal("cannot process a stag-generated file")
		}

		f, err := parser.ParseFile(fs, *source, nil, parser.AllErrors)

		if err != nil {
			log.Fatal(err)
		}

		vis := visitor{}
		ast.Walk(vis, f)
		if vis.file != nil {
			vis.file.BasePath = *source
			files = append(files, vis.file)
		}
	} else {
		pkgs, err := parser.ParseDir(fs, *source, func(fi os.FileInfo) bool {
			return !rxIsStagFile.MatchString(fi.Name())
		}, parser.AllErrors)

		if err != nil {
			log.Fatal(err)
		}

		for pkgName, pkg := range pkgs {
			fmt.Println("pkg: ", pkgName)
			for filePath, file := range pkg.Files {
				fmt.Println("\t-" + filePath)
				vis := visitor{file: &model.File{BasePath: filePath}}
				ast.Walk(vis, file)
				files = append(files, vis.file)
			}
		}
	}

	if len(files) == 0 {
		fmt.Print("no files needed processing")
		return
	}

	// grab all imports as *ast.Pkg
	imports := pkgImports{}

	// update file structs imports with fully qualified paths
	for _, f := range files {
		for _, s := range f.Structs {
			for idx := range s.ImportEmbeds {
				ie := &s.ImportEmbeds[idx]
				if imp := f.Imports.ByAlias(ie.PkgName); imp != nil {
					ie.FullyQualifiedPkgName = imp.PkgPath
				}
			}
		}
	}

	embedPkgs := files.EmbeddedImportPkgNames() // these are unique already
	for pidx := range embedPkgs {
		imp := pkgImport{
			path: embedPkgs[pidx],
			fs:   token.NewFileSet(),
		}
		fmt.Println("Processing imported pkg: ", imp.path)
		var pkgs map[string]*ast.Package
		if buildPkg, err := build.Import(imp.path, files[0].BaseDir(), build.FindOnly); err != nil {
			log.Fatalf("Error finding pkg dir for %s: %s", imp.path, err.Error())
		} else if pkgs, err = parser.ParseDir(imp.fs, buildPkg.Dir, func(fi os.FileInfo) bool {
			return !rxIsStagFile.MatchString(fi.Name())
		}, parser.AllErrors); err != nil {
			log.Fatalf("Error parsing pkg dir of %s for %s: %s", buildPkg.Dir, imp.path, err.Error())
		}

		for pkgName, pkg := range pkgs {
			if strings.HasSuffix(imp.path, pkgName) {
				imp.pkg = pkg
				imports = append(imports, imp)
				break
			}
		}

		if imp.pkg == nil {
			log.Fatalf("Could not find package with path %s", imp.path)
		}
	}

	for _, f := range files {
		for _, s := range f.Structs {
			for idx := range s.ImportEmbeds {
				ie := &s.ImportEmbeds[idx]
				if imp := imports.ByPath(ie.FullyQualifiedPkgName); imp != nil {
					if s := imp.loadStruct(ie.StructName); s != nil {
						ie.Struct = s
					} else {
						log.Fatalf("Could not find struct by name %s in pkg %s", ie.StructName, ie.FullyQualifiedPkgName)
					}
				} else {
					log.Fatalf("Unable to find import for %s", ie.FullyQualifiedPkgName)
				}
			}
		}
	}

	files.JoinEmbeds()

	tagGenerators := map[string][]*generator{}
	for _, t := range tags {
		tagGenerators[t] = []*generator{}
	}

	for _, f := range files {
		for _, tag := range tags {
			if _, exists := tagGenerators[tag]; !exists {
				tagGenerators[tag] = []*generator{}
			}
			tagGenerators[tag] = append(tagGenerators[tag], &generator{file: f, tag: tag})
		}
	}

	for _, tag := range tags {
		fmt.Printf("Processing for tag %s...\n", tag)
		for _, g := range tagGenerators[tag] {
			g.Generate()
			// not every file will have things we need to generate for
			if len(g.buf.Bytes()) == 0 {
				fmt.Printf("skipping file %s\n", g.file.BasePath)
				continue
			}
			dst := os.Stdout
			if *outType != "stdout" {
				g.dstFileName = strings.Replace(g.file.BasePath, ".go", fmt.Sprintf(".stag-%s.go", g.tag), 1)
				outFile, err := os.Create(g.dstFileName)
				if err != nil {
					log.Fatalf("Failed opening destination file %s: %v", g.dstFileName, err)
				}
				defer outFile.Close()
				dst = outFile
			}
			byts := g.Output()
			if _, err := dst.Write(byts); err != nil {
				log.Fatalf("Failed writing to destination: %v", err)
			}
		}
	}
}
