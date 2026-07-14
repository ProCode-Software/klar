package parser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func (p *Parser) handleInvalidNumber(
	nerr *lexer.NumberError, format lexer.IntFormat, tok lexer.Token,
) {
	var (
		err    *klarerrs.Error
		src    = tok.Source
		tokPos = tok.Position
		errPos = tokPos.Add(0, nerr.Offset)
	)
	switch nerr.Code {
	case lexer.ErrIntMisplacedSeparator:
		switch {
		case strings.Contains(src, "__"):
			// Consecutive separator
			err = klarerrs.Position(klarerrs.ErrConsecutiveSeparator, errPos)
		case src[len(src)-1] == '_':
			// Separator at end of number
			err = klarerrs.Position(klarerrs.ErrTrailingSeparator, errPos)
		default:
			// Somewhere else
			err = klarerrs.Position(klarerrs.ErrMisplacedSeparator, errPos)
		}
	case lexer.ErrIntIncompatibleDigit:
		err = klarerrs.TokenPos(
			map[lexer.IntFormat]klarerrs.Code{
				lexer.NumberFormatDecimal: klarerrs.ErrExpectedDecimal,
				lexer.NumberFormatBinary:  klarerrs.ErrExpectedBinary,
				lexer.NumberFormatHex:     klarerrs.ErrExpectedHex,
			}[format], errPos, tok,
		)
	case lexer.ErrInvalidDecimalPoint:
		err = klarerrs.Position(klarerrs.ErrInvalidDecimalPoint, errPos)
	default:
		panic(fmt.Sprintf("unhandled lexer.NumberErrorCode: %d", nerr.Code))
	}
	p.Error(err)
}

func (p *Parser) ParseNumber() ast.Expression {
	var (
		tok = p.Advance()
		src = tok.Source
		a   = tok.Attributes["params"].(lexer.NumberAttrs)
	)
	switch {
	case a.Error != nil:
		p.handleInvalidNumber(a.Error, a.Format, tok)
		src = "0" // Set default value for ParseInt call
	case (a.Flags & lexer.IsFloat) != 0:
		// Exponents are floats
		val, err := strconv.ParseFloat(src, 64)
		return &ast.FloatLiteral{
			Source: src,
			Value:  tryStrconv(p, tok, val, err),
			Flags:  a.Flags,
		}
	// Go parses 0 prefix as octal (including '0_386')
	// Also check if prefix is not 0o, 0b, or 0x
	case len(src) > 1 && (src[1] == '_' || lexer.IsDigit(rune(src[1]))):
		src = strings.TrimLeft(src, "0_")
	}
	val, err := strconv.ParseInt(src, 0, 0)
	return &ast.IntegerLiteral{
		Format: a.Format,
		Source: src,
		Value:  tryStrconv(p, tok, val, err),
		Flags:  a.Flags,
	}
}

// tryStrconv returns res after handling any error. If there is a value overflow,
// an error is reported to the parser. If any other error occurs, tryStrconv panics.
func tryStrconv[T int64 | float64](p *Parser, tok lexer.Token, res T, err error) T {
	if err == nil {
		return res
	}
	if errors.Is(err, strconv.ErrRange) {
		err := klarerrs.Token(klarerrs.ErrNumberTooBig, tok)
		err.Name = tok.Source
		p.ErrorLabelled(err, "Can you read this number?")
		return res
	}
	// All Klar number are valid Go numbers. Apart from an overflow, strconv
	// failing means there is a bug in the lexer.
	panic(fmt.Sprintf("strconv.ParseInt/Float failed while parsing numeric literal: %v", err))
}

func (p *Parser) ParseSymbol() *ast.Symbol {
	tok := p.Advance()
	sym := &ast.Symbol{Identifier: tok.Source}
	rangeFromToken(sym, tok)
	p.ValidateIdentName(sym.Identifier, sym)
	return sym
}

func (p *Parser) ParseString() *ast.StringLiteral {
	var (
		token   = p.Advance()
		a       = token.Attributes["params"].(lexer.StringAttrs)
		src     = token.Source
		escapes = p.parseStringEscapes(token)
		full    string
	)
	if a.Unterminated {
		full = src[1:]
		if a.QuoteStyle != '`' && src[len(src)-1] == '\n' {
			// Use a better error for quoted strings with newline
			p.Error(klarerrs.Position(
				klarerrs.ErrMultilineQuotedString,
				token.End(),
			))
		} else {
			// TODO: End() is wrong
			err := klarerrs.Position(klarerrs.ErrUnterminatedString, token.End() /* .Add(0, 1) */)
			err.Label = "Expected closing " + klarerrs.Quote(string(a.QuoteStyle))
			err.Highlights = append(err.Highlights, klarerrs.Highlight{
				Range:   ranges.Offset(token.Position, 0, 1),
				Message: "It was started here",
			})
			err.SetParam("start", token.Position)
			p.Error(err)
		}
	} else {
		full = src[1 : len(src)-1] // Remove quotes
	}
	return &ast.StringLiteral{
		QuoteStyle: a.QuoteStyle,
		Fragments:  escapes,
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
		// TODO: End() is wrong
		err := klarerrs.Position(klarerrs.ErrUnterminatedRegex, re.End())
		err.Highlights = append(err.Highlights, klarerrs.Highlight{
			Range:   ranges.SingleChar(re.Position),
			Message: "It was started here",
		})
		err.SetParam("start", re.Position)
		p.ErrorLabelled(err, "Expected closing '/'")
	}
	frags := make([]ast.StringFragment, len(params.Fragments))
	for i, frag := range params.Fragments {
		switch frag := frag.(type) {
		case lexer.TextFragment:
			frags[i] = frag
		case lexer.StringEscape:
			// The only escape in regex should be interpolation
			if frag.Type != lexer.EscInterpolation {
				panic(fmt.Sprintf(
					"invalid escape in regex: expected EscInterpolation, but got %d",
					frag.Type,
				))
			}
			frags[i] = ast.InterpolationFragment{
				Expression: p.parseStringInterpolation(*frag.Interpolated),
			}
		}
	}
	return &ast.RegexLiteral{
		Source:    params.Source,
		Flags:     params.Flags,
		Multiline: params.Multiline,
		Fragments: frags,
	}
}
