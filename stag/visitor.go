package main

import (
	"fmt"
	"go/ast"
	"log"
	"strconv"
	"strings"

	"github.com/bradleygore/stag/model"
)

type visitor struct {
	depth int
	file  *model.File
}

func (v visitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}
	spacer := strings.Repeat("\t", int(v.depth))
	nodeInfo := fmt.Sprintf("%s%T", spacer, n)
	switch node := n.(type) {
	case *ast.File:
		nodeInfo += "(" + node.Name.String() + ")\n"
		if v.file == nil {
			v.file = &model.File{}
		}
		v.file.PkgName = node.Name.String()
		v.processFile(v.file, node)
	case *ast.ImportSpec:
		importName := node.Path.Value
		if node.Name != nil {
			importName = node.Name.Name
		}
		nodeInfo += "(" + node.Path.Value + "[" + node.Path.Kind.String() + "]" + " as " + importName + ")\n"
	case *ast.TypeSpec:
		nodeInfo += "(" + node.Name.Name + ")\n"
	case *ast.Field:
		nodeInfo += "(" + v.identNames(node.Names) + "[" + fmt.Sprintf("%v", node.Type) + "]"
		if node.Tag != nil {
			nodeInfo += " | " + node.Tag.Value
		}
		nodeInfo += ")\n"
	case *ast.Ident:
		nodeInfo += "(" + node.Name + ")\n"
	case *ast.SelectorExpr:
		nodeInfo += "(" + node.Sel.Name
		if node.Sel.Obj != nil {
			nodeInfo += "[" + node.Sel.Obj.Name + ":" + node.Sel.Obj.Kind.String() + "]"
		}
		nodeInfo += ")\n"
	default:
		nodeInfo += "\n"
	}
	if *verbose {
		fmt.Printf("visitor=%d%s", v.depth, nodeInfo)
	}
	v.depth += 1
	return v
}

func (v visitor) identNames(names []*ast.Ident) string {
	n := ""
	for idx := range names {
		if names[idx] != nil {
			if idx > 0 {
				n += ","
			}
			if names[idx].Obj != nil {
				n += "(" + names[idx].Obj.Name + ":" + names[idx].Obj.Kind.String() + ")"
			}
			n += names[idx].Name
		}
	}
	return n
}

func (v visitor) processFile(f *model.File, astf *ast.File) {
	for _, dec := range astf.Decls {
		switch decNode := dec.(type) {
		case *ast.GenDecl:
			for _, spec := range decNode.Specs {
				switch node := spec.(type) {
				case *ast.ImportSpec:
					importName := node.Path.Value
					if node.Name != nil {
						importName = node.Name.Name
					}
					if unq, err := strconv.Unquote(importName); err == nil {
						importName = unq
					}
					pkgPath := node.Path.Value
					if unq, err := strconv.Unquote(pkgPath); err == nil {
						pkgPath = unq
					}
					f.Imports = append(f.Imports, model.Import{
						PkgPath: pkgPath,
						Alias:   importName,
					})
				case *ast.TypeSpec:
					if struc, ok := node.Type.(*ast.StructType); ok {
						fStruct := &model.Structure{Name: node.Name.String()}
						for _, field := range struc.Fields.List {
							if v.identNames(field.Names) == "" {
								// dealing with embed
								if selectorExp, ok := field.Type.(*ast.SelectorExpr); ok {
									// embedding a type from imported pkg
									fStruct.ImportEmbeds = append(fStruct.ImportEmbeds, model.ImportEmbed{
										PkgName:    selectorExp.X.(*ast.Ident).Name,
										StructName: selectorExp.Sel.Name,
									})
								} else {
									// embedding a type local to the pakg
									fStruct.EmbedNames = append(fStruct.EmbedNames, fmt.Sprintf("%v", field.Type))
								}
								continue
							}
							if field.Tag == nil {
								continue
							}
							fieldName := field.Names[0].String()
							tags := v.parseFieldTag(field.Tag.Value, fieldName)
							if len(tags) > 0 {
								for tagName, tagFieldName := range tags {
									fStruct.AddFieldTagName(tagName, fieldName, tagFieldName)
								}
							}
						}
						f.Structs = append(f.Structs, fStruct)
					}
				}
			}
		}
	}
}

func (v visitor) parseFieldTag(tag, structFieldName string) map[string]string {
	tagNames := make(map[string]string)
	tags := strings.Split(strings.ReplaceAll(tag, "`", ""), " ")
	for _, t := range tags {
		colon := strings.IndexRune(t, ':')
		tagName, tagVal := t[:colon], t[colon+1:]
		tagFieldName := strings.SplitN(tagVal, ",", 1)[0]
		var err error
		if tagFieldName, err = strconv.Unquote(tagFieldName); err != nil {
			log.Fatalf("err unquoting tagFieldName %s: %v", tagFieldName, err)
		}
		// some tags use commas to separate values, by convention name comes first
		tagFieldName = strings.Split(tagFieldName, ",")[0]
		if len(tagFieldName) == 0 {
			tagFieldName = structFieldName
		}
		tagNames[tagName] = tagFieldName
	}

	return tagNames
}
