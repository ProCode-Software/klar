package parser

import (
	"strconv"
	"strings"

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

func (p *Parser) handleInvalidNumber(code int, format lexer.IntegerFormat, tok lexer.Token) {
	var (
		err    errors.ParseError
		src    = tok.Source
		tokPos = tok.Position
		errPos = ranges.Add(tokPos, 0, uint32(tok.Attributes["errorPos"].(int)))
	)
	switch code {
	case lexer.ErrIntMisplacedSeparator:
		switch {
		case strings.Contains(src, "__"):
			// Consecutive separator
			err = errors.Position(errors.ErrConsecutiveSep, errPos)
		case src[len(src)-1] == '_':
			// Separator at end of number
			err = errors.Position(errors.ErrTrailingSep, errPos)
		default:
			// Somewhere else
			err = errors.Position(errors.ErrMisplacedSep, errPos)
		}
	case lexer.ErrIntIncompatibleDigit:
		err = errors.TokenPos(
			map[lexer.IntegerFormat]errors.ErrorCode{
				lexer.NumberFormatDecimal: errors.ErrExpectedDecimal,
				lexer.NumberFormatBinary:  errors.ErrExpectedBinary,
				lexer.NumberFormatOctal:   errors.ErrExpectedOctal,
				lexer.NumberFormatHex:     errors.ErrExpectedHex,
			}[format], errPos, tok,
		)
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
	case a.Invalid:
		p.handleInvalidNumber(a.Error, format, token)
		// Set default value for ParseInt call
		src = "0"
	case a.Float:
		// Exponents are floats
		return &ast.FloatLiteral{
			Source:    src,
			Value:     unwrap(strconv.ParseFloat(src, 64)),
			Separator: a.HasSeparator,
			Exponent:  a.HasExponent,
		}
	// Go parses 0 prefix as octal
	// Also check if prefix is not 0o, 0b, or 0x
	case len(src) > 1 && (src[1] == '_' || lexer.IsDigit(rune(src[1]))):
		src = strings.TrimLeft(src, "0")
	}
	return &ast.IntegerLiteral{
		Format:    format,
		Source:    src,
		Value:     unwrap(strconv.ParseInt(src, 0, 0)),
		Separator: a.HasSeparator,
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
	)
	if a.QuoteCount > 0 {
		strLen = len(src) - 2*a.QuoteCount - 1
	}
	full := src[start : start+strLen] // Remove quotes
	if a.Unterminated {
		p.Error(errors.Position(errors.ErrUnterminatedString, token.Position))
		full = src[start:]
	}
	return &ast.StringLiteral{
		QuoteStyle: a.QuoteStyle,
		Segments:   a.Segments,
		QuoteCount: a.QuoteCount,
		Content:    full,
		Escapes:    escapes,
	}
}

func (p *Parser) ParseBoolean() *ast.BooleanLiteral {
	switch src := p.Advance().Source; src {
	case "true":
		return &ast.BooleanLiteral{Value: true}
	case "false":
		return &ast.BooleanLiteral{Value: false}
	default:
		panic("invalid boolean literal: '" + src + "'")
	}
}

func (p *Parser) ParseNil() *ast.NilLiteral {
	p.Advance()
	return &ast.NilLiteral{}
}

func (p *Parser) ParseRegexToken() *ast.RegexLiteral {
	re := p.Advance()
	params := re.Attributes["params"].(lexer.RegexAttrs)
	if params.Unterminated {
		p.Error(errors.Position(errors.ErrUnterminatedRegex, re.Position))
	}
	return &ast.RegexLiteral{
		Source:     params.Source,
		Flags:      params.Flags,
		QuoteCount: params.SlashCount,
		Multiline:  params.Multiline,
	}
}
