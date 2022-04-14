package main

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/bradleygore/stag/model"
	toolsimports "golang.org/x/tools/imports"
)

type generator struct {
	buf         bytes.Buffer
	indentLevel int
	file        *model.File
	tag         string
	dstFileName string // compiled during Generate
}

func (g *generator) indent() {
	g.indentLevel += 1
}
func (g *generator) outdent() {
	if g.indentLevel > 0 {
		g.indentLevel -= 1
	}
}

func (g *generator) spacer() string {
	return strings.Repeat("\t", g.indentLevel)
}

func (g *generator) fp(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, g.spacer()+format+"\n", args...)
}

func (g *generator) Generate() {
	if g.file == nil {
		log.Fatal("cannot Generate with no file")
	}
	strucs := g.file.Structs.HavingTags([]string{g.tag})
	if len(strucs) == 0 {
		return
	}
	tagUpper := strings.ToUpper(g.tag)
	g.fp("// Code generated by stag. DO NOT EDIT.")
	g.fp("// Source file: %s", g.file.BasePath)
	g.fp("")
	g.fp("package %s", g.file.PkgName)
	g.fp("")
	for _, s := range strucs {
		structName := fmt.Sprintf("%s_%s", s.Name, tagUpper)
		allTagFieldNamesProp := fmt.Sprintf("All%sFieldNames", tagUpper)
		g.fp("var %s = struct {", structName)
		g.indent()
		fields := s.FieldTagNames[g.tag]
		for _, field := range fields {
			if field.IsSkipped() {
				continue
			}
			g.fp("%s string", field.FieldName)
		}
		g.fp("%s []string", allTagFieldNamesProp)
		g.outdent()
		g.fp("}{")
		g.fp("")
		g.indent()
		for _, field := range fields {
			if field.IsSkipped() {
				continue
			}
			g.fp(`%s:"%s",`, field.FieldName, field.TagName)
		}
		allTagNames := ""
		for idx, tn := range fields.TagNames() {
			if idx > 0 {
				allTagNames += ","
			}
			allTagNames += fmt.Sprintf(`"%s"`, tn)
		}
		g.fp("%s:[]string{%s},", allTagFieldNamesProp, allTagNames)
		g.outdent()
		g.fp("}\n")
		g.fp("")
		g.fp("func IsValid%sField(f string) bool {", structName)
		g.indent()
		g.fp("for _, fn := range %s.%s {", structName, allTagFieldNamesProp)
		g.indent()
		g.fp("if fn == f {")
		g.indent()
		g.fp("return true")
		g.outdent()
		g.fp("}")
		g.outdent()
		g.fp("}")
		g.fp("return false")
		g.outdent()
		g.fp("}")
		g.fp("")
	}
	g.fp("")
}

// Output returns the generator's output, formatted in the standard Go style.
func (g *generator) Output() []byte {
	src, err := toolsimports.Process(g.dstFileName, g.buf.Bytes(), nil)
	if err != nil {
		log.Fatalf("Failed to format generated source code: %s\n%s", err, g.buf.String())
	}
	return src
}