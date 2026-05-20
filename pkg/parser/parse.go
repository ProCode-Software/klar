package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/parser"
)

type (
	Error   = klarerrs.Error
	Options = parser.Options
)

func Parse(tokens []lexer.Token, options *parser.Options) (
	program *ast.Program, errors []*Error,
) {
	p := parser.New(tokens, options)
	program = p.Parse()
	errors = p.Errors
	return
}

func ParseString(src string) (program *ast.Program, errs []*Error) {
	return Parse(TokenizeString(src), nil)
}
