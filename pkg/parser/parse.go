package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/parser"
)

type (
	ParseError = errors.ParseError
	Options    = parser.ParseOptions
)

func Parse(tokens []lexer.Token, options *parser.ParseOptions) (
	program *ast.Program, errors []ParseError,
) {
	if options == nil {
		options = &parser.ParseOptions{}
	}
	p := parser.New(tokens, options)
	program = p.Parse()
	errors = p.Errors
	return
}

func ParseString(src string) (program *ast.Program, errors []ParseError, lexerErr error) {
	tokens, err := TokenizeString(src, true)
	if err != nil {
		return nil, nil, err
	}
	program, errors = Parse(tokens, nil)
	return
}
