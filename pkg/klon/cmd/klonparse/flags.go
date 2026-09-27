package main

import (
	"flag"

	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

func parseKlonFlags() (flags klonflags.Flags, ast bool) {
	flag.BoolVar(&ast, "ast", false, "Return an AST representation of the document")
	boolIsString := flag.Bool(
		"bool-is-string", false, "'true' and 'false' literals are strings",
	)
	numberIsString := flag.Bool("num-is-string", false, "Numeric literals are strings")
	emptyValueIsString := flag.Bool(
		"empty-val-is-str", false, "Empty markup values are decoded as empty strings",
	)
	omitNullFields := flag.Bool(
		"omit-null-fields", false, "Don't include fields set to 'none'",
	)
	noVars := flag.Bool("no-vars", false, "Don't allow variables in the input")
	flag.Parse()
	if *boolIsString {
		flags |= klonflags.BoolIsString
	}
	if *numberIsString {
		flags |= klonflags.NumberIsString
	}
	if *emptyValueIsString {
		flags |= klonflags.EmptyValueIsString
	}
	if *omitNullFields {
		flags |= klonflags.OmitNullFields
	}
	if *noVars {
		flags |= klonflags.NoVariables
	}
	return
}
