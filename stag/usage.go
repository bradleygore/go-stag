package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

var usageText = `stag generates static structs based on tag names for source structs

Example:
	stag -source=path/to/foo.go -tags=json,db

Given that source contains a struct like:
type Foo struct {
	Name string ` + usageBacktick(`json:"theName" db:"the_name"`) + `
	Flavor string ` + usageBacktick(`json:"yummyFlavor" db:"mmm_flavor"`) + `
}

stag will generate two files, having these contents:

//path/to/foo.stag_json.go
var Foo_JSON = struct{
	Name string
	Flavor string
}{
	Name: "theName",
	Flavor: "yummyFlavor",
}

//path/to/foo.stag_db.go
var Foo_DB = struct{
	Name string
	Flavor string
}{
	Name: "the_name",
	Flavor: "mmm_flavor",
}
`

func printUsage(printDefaultFlags bool) {
	_, _ = io.WriteString(os.Stderr, usageText)
	if printDefaultFlags {
		_, _ = io.WriteString(os.Stderr, "\n\n")
		flag.PrintDefaults()
	}
}

func usageBacktick(s string) string {
	return fmt.Sprintf("`%s`", s)
}
