package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/parser"
)

type (
	ParseError = errors.ParseError
	Options    = parser.Options
)

func Parse(tokens []lexer.Token, options *parser.Options) (
	program *ast.Program, errors []*ParseError,
) {
	p := parser.New(tokens, options)
	program = p.Parse()
	errors = p.Errors
	return
}

func ParseString(src string) (program *ast.Program, parseErrs []*ParseError, readErr error) {
	tokens, err := TokenizeString(src, lexer.IncludeComments)
	if err != nil {
		return nil, nil, err
	}
	program, parseErrs = Parse(tokens, nil)
	return
}
