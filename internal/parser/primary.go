package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// Unwraps the error tuple and panics if err != nil
func unwrap[T any](res T, err error) T {
	if err != nil {
		panic(err)
	}
	return res
}

func (p *Parser) ParsePrimaryExpression() ast.ASTItem {
	token := p.Advance()
	src := token.Source
	switch token.Kind {
	case lexer.Identifier:
		return ast.Symbol{Identifier: src}
	case lexer.Numeric:
		if strings.Contains(src, ".") {
			return ast.FloatLiteral{
				Value: unwrap(strconv.ParseFloat(src, 64)),
			}
		}
		return ast.IntegerLiteral{
			Format: token.Attributes["format"].(int),
			Value:  int(unwrap(strconv.ParseInt(src, 0, 0))),
		}
	case lexer.String:
		if token.Attributes["err"] == lexer.ErrStrUnterminated {
			panic(errors.UnterminatedStringError(token.Position))
		}
		escapes := parseStringEscapes(token)
		return ast.StringLiteral{
			QuoteStyle: token.Attributes["quoteStyle"].(rune),
			Content:    token.Source[1 : len(token.Source)-1], // Remove quotes
			Escapes:    escapes,
		}
	case lexer.Boolean:
		return ast.BooleanLiteral{
			Value: unwrap(strconv.ParseBool(src)),
		}
	case lexer.Nil:
		return ast.NilLiteral{}
	default:
		panic(fmt.Sprintf(
			"Expected primary expression, got %s",
			token.Kind.String(),
		))
	}
}
