package model

// Import is a singular import on a File
type Import struct {
	PkgPath string // full.domain/path/to/pkg
	Alias   string // imported as
}

type Imports []Import

func (i Imports) ByAlias(alias string) *Import {
	for _, imp := range i {
		if imp.Alias == alias {
			return &imp
		}
	}
	return nil
}

// ImportEmbed is an imported type that is embedded into a struct
type ImportEmbed struct {
	PkgName               string // short name used when referencing the imported pkg
	FullyQualifiedPkgName string // full pkg path
	StructName            string
	Struct                *Structure
}
