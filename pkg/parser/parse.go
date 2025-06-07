package parser

import (
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	parse "github.com/ProCode-Software/klar/internal/parser"
)

type (
	ParseError   = errors.ParseError
	ParseOptions = parse.ParseOptions
)

func NewParser(tokens []lexer.Token, options parse.ParseOptions) *parse.Parser {
	p := parse.New(tokens)
	p.Options = options
	return p
}
