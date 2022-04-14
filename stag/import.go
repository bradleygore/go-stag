package main

import (
	"go/ast"
	"go/token"
	"log"

	"github.com/bradleygore/go-stag/model"
)

type pkgImport struct {
	path          string
	fs            *token.FileSet
	pkg           *ast.Package
	files         model.Files
	structsByName map[string]model.Structure
}

func (i *pkgImport) loadStruct(name string) *model.Structure {
	if i.structsByName == nil {
		i.structsByName = make(map[string]model.Structure)
	}

	if s, exists := i.structsByName[name]; exists {
		return &s
	}

	for fileName := range i.pkg.Files {
		f := i.pkg.Files[fileName]
		if i.fileContainsStruct(name, f) {
			v := visitor{file: &model.File{PkgName: i.pkg.Name}}
			v.processFile(v.file, f)
			i.files = append(i.files, v.file)
			if s := v.file.Structs.ByName(name); s != nil {
				i.structsByName[name] = *s
				return s
			}
			log.Fatalf("Unable to find struct %s in file %s from pkg %s after parsing", name, fileName, i.pkg.Name)
		}
	}

	return nil
}

func (i *pkgImport) fileContainsStruct(name string, f *ast.File) bool {
	for idx := range f.Decls {
		d := f.Decls[idx]
		switch decNode := d.(type) {
		case *ast.GenDecl:
			for _, spec := range decNode.Specs {
				switch node := spec.(type) {
				case *ast.TypeSpec:
					if _, ok := node.Type.(*ast.StructType); ok {
						if node.Name.String() == name {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

type pkgImports []pkgImport

func (pi pkgImports) ByPath(path string) *pkgImport {
	for _, i := range pi {
		if i.path == path {
			return &i
		}
	}
	return nil
}
