package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// tryStrconv returns res, panicking if err != nil.
func tryStrconv[T any](res T, err error) T {
	if err != nil {
		// All Klar number are valid Go numbers. strconv failing means
		// there is a bug in the lexer.
		panic("strconv.ParseInt/ParseFloat should not fail: " + err.Error())
	}
	return res
}

func (p *Parser) handleInvalidNumber(
	ne *lexer.NumberError, format lexer.IntFormat, tok lexer.Token,
) {
	var (
		err    *errors.ParseError
		src    = tok.Source
		tokPos = tok.Position
		errPos = ranges.Add(tokPos, 0, ne.Offset)
	)
	switch ne.Code {
	case lexer.ErrIntMisplacedSeparator:
		switch {
		case strings.Contains(src, "__"):
			// Consecutive separator
			err = errors.Position(errors.ErrConsecutiveSeparator, errPos)
		case src[len(src)-1] == '_':
			// Separator at end of number
			err = errors.Position(errors.ErrTrailingSeparator, errPos)
		default:
			// Somewhere else
			err = errors.Position(errors.ErrMisplacedSeparator, errPos)
		}
	case lexer.ErrIntIncompatibleDigit:
		err = errors.TokenPos(
			map[lexer.IntFormat]errors.ErrorCode{
				lexer.NumberFormatDecimal: errors.ErrExpectedDecimal,
				lexer.NumberFormatBinary:  errors.ErrExpectedBinary,
				lexer.NumberFormatOctal:   errors.ErrExpectedOctal,
				lexer.NumberFormatHex:     errors.ErrExpectedHex,
			}[format], errPos, tok,
		)
	case lexer.ErrInvalidDecimalPoint:
		err = errors.Position(errors.ErrInvalidDecimalPoint, errPos)
	default:
		panic(fmt.Sprintf("unhandled lexer.NumberErrorCode: %d", ne.Code))
	}
	p.Error(err)
}

func (p *Parser) ParseNumber() ast.Expression {
	var (
		token  = p.Advance()
		src    = token.Source
		a      = token.Attributes["params"].(lexer.NumberAttrs)
		format = a.Format
	)
	switch {
	case a.Error != nil:
		p.handleInvalidNumber(a.Error, format, token)
		// Set default value for ParseInt call
		src = "0"
	case (a.Flags & lexer.IsFloat) != 0:
		// Exponents are floats
		return &ast.FloatLiteral{
			Source:    src,
			Value:     tryStrconv(strconv.ParseFloat(src, 64)),
			Separator: (a.Flags & lexer.HasSeparator) != 0,
			Exponent:  (a.Flags & lexer.HasExponent) != 0,
		}
	// Go parses 0 prefix as octal
	// Also check if prefix is not 0o, 0b, or 0x
	case len(src) > 1 && (src[1] == '_' || lexer.IsDigit(rune(src[1]))):
		src = strings.TrimLeft(src, "0")
	}
	return &ast.IntegerLiteral{
		Format:    format,
		Source:    src,
		Value:     tryStrconv(strconv.ParseInt(src, 0, 0)),
		Separator: (a.Flags & lexer.HasSeparator) != 0,
	}
}

func (p *Parser) ParseSymbol() *ast.Symbol {
	return &ast.Symbol{Identifier: p.Advance().Source}
}

func (p *Parser) ParseString() *ast.StringLiteral {
	var (
		token   = p.Advance()
		a       = token.Attributes["params"].(lexer.StringAttrs)
		src     = token.Source
		escapes = p.parseStringEscapes(token)
		start   = 1 + a.QuoteCount
		strLen  = len(src) - 2
		full    string
	)
	if a.QuoteCount > 0 {
		strLen = len(src) - 2*a.QuoteCount - 1
	}
	if a.Unterminated {
		full = src[start:]
		if a.QuoteStyle != '`' && full[len(full)-1] == '\n' {
			// Use a better error for quoted strings with newline
			p.Error(errors.Position(errors.ErrMultilineQuotedString, token.Position))
		} else {
			p.Error(errors.Position(errors.ErrUnterminatedString, token.Position))
		}
	} else {
		full = src[start : start+strLen] // Remove quotes
	}
	return &ast.StringLiteral{
		QuoteStyle: a.QuoteStyle,
		Fragments:  escapes,
		QuoteCount: a.QuoteCount,
		Content:    full,
	}
}

func (p *Parser) ParseBoolean() *ast.BooleanLiteral {
	return &ast.BooleanLiteral{
		Value: p.Advance().Attributes["value"].(bool),
	}
}

func (p *Parser) ParseNil() *ast.NilLiteral {
	p.Advance()
	return &ast.NilLiteral{}
}

func (p *Parser) ParseRegexLiteral() *ast.RegexLiteral {
	re := p.Advance()
	params := re.Attributes["params"].(lexer.RegexAttrs)
	if params.Unterminated {
		p.Error(errors.Position(errors.ErrUnterminatedRegex, re.Position))
	}
	return &ast.RegexLiteral{
		Source:    params.Source,
		Flags:     params.Flags,
		Multiline: params.Multiline,
	}
}
