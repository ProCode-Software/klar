package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
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
		err    *klarerrs.Error
		src    = tok.Source
		tokPos = tok.Position
		errPos = tokPos.Add(0, ne.Offset)
	)
	switch ne.Code {
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
		full    string
	)
	if a.Unterminated {
		full = src[1:]
		if a.QuoteStyle != '`' && src[len(src)-1] == '\n' {
			// Use a better error for quoted strings with newline
			p.Error(klarerrs.Position(klarerrs.ErrMultilineQuotedString,
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
