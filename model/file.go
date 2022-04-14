package model

import (
	"fmt"
	"strings"
)

type File struct {
	BasePath string
	Name     string
	PkgName  string
	Structs  Structures
	Imports  Imports
}

func (f File) BaseDir() string {
	if strings.HasSuffix(f.BasePath, ".go") {
		return f.BasePath[0:strings.LastIndex(f.BasePath, "/")]
	}
	return f.BasePath
}

type Files []*File

func (fs Files) FindStruct(name string) *Structure {
	for _, f := range fs {
		if s := f.Structs.ByName(name); s != nil {
			return s
		}
	}

	return nil
}

// returns all fully-qualified pkg names of embedded imports
func (fs Files) EmbeddedImportPkgNames() []string {
	imps := make(map[string]bool)
	for _, f := range fs {
		for _, imp := range f.Structs.EmbeddedImportPkgNames() {
			if _, exists := imps[imp]; !exists {
				imps[imp] = true
			}
		}
	}
	ret := make([]string, 0)
	for imp := range imps {
		ret = append(ret, imp)
	}
	return ret
}

func (fs Files) JoinEmbeds() {
	for _, f := range fs {
		for _, s := range f.Structs {
			for _, impEmb := range s.ImportEmbeds {
				for tag, tagFields := range impEmb.Struct.FieldTagNames {
					for _, tagField := range tagFields {
						s.AddFieldTagName(tag, tagField.FieldName, tagField.TagName)
					}
				}
			}
			for _, embName := range s.EmbedNames {
				embStruct := fs.FindStruct(embName)
				if embStruct == nil {
					fmt.Printf("Could not find embed struct def for %s\n", embName)
					continue
				}
				for tag, tagFields := range embStruct.FieldTagNames {
					for _, tagField := range tagFields {
						s.AddFieldTagName(tag, tagField.FieldName, tagField.TagName)
					}
				}
			}
		}
	}
}
