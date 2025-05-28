package parser

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// Unwraps the error tuple and panics if err != nil
func unwrap[T any](res T, err error) T {
	if err != nil {
		panic(err)
	}
	return res
}

func handleInvalidNumber(code, format int, tok lexer.Token) {
	var (
		err    errors.ParseError
		src    = tok.Source
		tokPos = tok.Position
		errPos = ranges.Add(tokPos, 0, tok.Attributes["errorPos"].(int))
	)
	switch code {
	case lexer.ErrIntMisplacedSeparator:
		// Check for consecutive separator
		consec := strings.Index(src, "__")
		if consec > -1 {
			err = errors.NewPositionError(errors.ErrConsecutiveSep, errPos)
		}
		// Otherwise it's at the beginning or end
		err = errors.NewPositionError(errors.ErrMisplacedSep, errPos)

	case lexer.ErrIntIncompatibleDigit:
		err = errors.NewTokenPosError(
			map[int]errors.ErrorCode{
				lexer.NumberFormatDecimal: errors.ErrExpectedDecimal,
				lexer.NumberFormatBinary:  errors.ErrExpectedHex,
				lexer.NumberFormatOctal:   errors.ErrExpectedOctal,
				lexer.NumberFormatHex:     errors.ErrExpectedHex,
			}[format], errPos, tok,
		)
	case lexer.ErrIntIllegalExponent:

	}
	panic(err)
}

func (p *Parser) ParsePrimaryExpression() ast.Node {
	var (
		token = p.Advance()
		src   = token.Source
	)
	switch token.Kind {
	case lexer.Identifier:
		return ast.Symbol{Identifier: src}
	case lexer.Numeric:
		format := token.Attributes["format"].(int)
		switch {
		case token.Attributes["invalid"] == true:
			handleInvalidNumber(token.Attributes["error"].(int), format, token)

		case strings.Contains(src, "."),
			format != lexer.NumberFormatHex && strings.ContainsAny(src, "eE"):
			// Exponents are floats
			return ast.FloatLiteral{
				Value: unwrap(strconv.ParseFloat(src, 64)),
			}
		// Go parses 0 prefix as octal
		case len(src) > 1 && (src[1] == '_' || unicode.IsDigit(rune(src[1]))):
			src = strings.TrimLeft(src, "0")
		}
		return ast.IntegerLiteral{
			Format: format,
			Value:  int(unwrap(strconv.ParseInt(src, 0, 0))),
		}
	case lexer.String:
		if token.Attributes["unterminated"] == true {
			panic(errors.NewPositionError(errors.ErrUnterminatedString, token.Position))
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
	case lexer.HashLeftCurlyBrace:
		return p.ParseMap()
	case lexer.Nil:
		return ast.NilLiteral{}
	default:
		panic(fmt.Sprintf(
			"Expected primary expression, got %s",
			token.Kind.String(),
		))
	}
}
