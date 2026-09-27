// Klonparse parses a Klon file and converts it to JSON, or generates an
// AST represented as JSON.
//
// Exit codes:
//
//	0: Success
//	1: Parse error
//	3: Internal error
package main

import (
	"encoding/json/v2"
	"flag"
	"fmt"
	"os"
	"reflect"

	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonerrs"
	"github.com/ProCode-Software/klar/pkg/klon/klonflags"
)

const help = `Klonparse parses a Klon file and converts it to JSON, or generates an AST represented as JSON.

Exit codes returned by klonparse:
  0: Success
  1: Parse error
  3: Internal error`

func main() {
	flag.Usage = func() {
		fmt.Println(help)
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
	}
	flags, astMode := parseKlonFlags()
	doc, errs := klon.ParseRead(os.Stdin, flags)
	if len(errs) > 0 {
		writeErrors(errs)
		os.Exit(1)
	}
	if astMode {
		dumpAST(doc)
	} else {
		klonToJSON(doc, flags)
	}
	os.Exit(0)
}

func klonToJSON(doc *ast.Document, flags klonflags.Flags) {
	asAny, err := klon.UnmarshallDocumentAny(doc, flags)
	if err != nil {
		// Errors occur during preprocessing, so this is sure to be caused by
		// the input, such as unknown variable references.
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.MarshalWrite(os.Stdout, asAny); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to write JSON:", err)
		os.Exit(3)
	}
}

func dumpAST(doc *ast.Document) {
	err := json.MarshalWrite(
		os.Stdout, doc,
		// Add a Type field to every AST node
		json.WithMarshalers(json.MarshalFunc(marshalASTNode)),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to write AST:", err)
		os.Exit(3)
	}
}

// marshalASTNode marshals an AST node to JSON, adding a "Type" field to the result.
// The "Type" field contains the name of the node's type.
func marshalASTNode(node ast.Node) ([]byte, error) {
	// Not passing [json.WithMarshalers] here means nested nodes won't go
	// through this marshaller
	orig, err := json.Marshal(node)
	if err != nil {
		return nil, err
	}
	if orig[0] != '{' {
		return nil, fmt.Errorf("internal error: Klon ast.Node is not a struct")
	}
	rt := reflect.TypeOf(node)
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	res := fmt.Appendf(orig[:1], `"Type":"%s",%s`, rt.Name(), orig[1:])
	return res, nil
}

// Errors
// =========

func writeErrors(errs []*klon.Error) {
	converted := make([]klonError, len(errs))
	for i, e := range errs {
		converted[i] = klonError{
			Code:    e.Code,
			Range:   e.Range,
			Text:    e.Text,
			Token:   e.Token,
			Value:   e.Value,
			Warning: e.Warning,
		}
	}
	err := json.MarshalWrite(os.Stdout, struct{ Errors []klonError }{converted})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to write errors:", err)
		os.Exit(3)
	}
}

type klonError struct {
	Code    klonerrs.Code
	Range   ranges.Range
	Text    string
	Token   klon.Token `json:",omitzero"`
	Value   ast.Value  `json:",omitempty"`
	Warning bool       `json:",omitzero"`
}
