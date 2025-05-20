package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/parser"
)

func ParseTokens(tokens []lexer.Token, continueOnErr bool) (program ast.Program, errors []error) {
	return parser.Parse(tokens, continueOnErr)
}
