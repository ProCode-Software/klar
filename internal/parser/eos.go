package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/lexer"
)

// InsertEOS remove newlines from the parser's Tokens and replaces them with
// end of statement instead.
//
// Klar does not use semicolons to terminate statements, and they are invalid.
// Newlines are required to terminate a statement, so InsertEOS will tell
// where a newline is being used to terminate a statement and replace it with
// end of statement. These are equivalent to semicolons in other languages.
func (p *Parser) InsertEOS() {
	if len(p.Tokens) <= 1 {
		return
	}
	for i := 0; i < len(p.Tokens); i++ {
		var (
			tok       = p.Tokens[i]
			insertEOS = true
		)
		switch {
		// Add before EOF
		case tok.Kind == lexer.EOF &&
			p.Tokens[i-1].Kind != lexer.EndOfStatement:
			fallthrough
		// Block with curly brace on same line:
		// 	func fn(x: Int) { return x * 2 }
		// but not {}
		case tok.Kind == lexer.RightCurlyBrace &&
			p.Tokens[i-1].Kind != lexer.EndOfStatement &&
			p.Tokens[i-1].Kind != lexer.LeftCurlyBrace:

			p.Tokens = slices.Insert(p.Tokens, i, lexer.Token{
				Kind:     lexer.EndOfStatement,
				Position: tok.Position,
			})
			if tok.Kind == lexer.EOF {
				break
			}
			i++
			continue
		case tok.Kind != lexer.Newline:
			continue
		case i > 0:
			switch p.Tokens[i-1].Kind {
			case
				// Punctuation
				lexer.Comma, lexer.LeftBracket, lexer.LeftCurlyBrace,
				lexer.LeftParenthesis, lexer.Colon, lexer.EndOfStatement,
				lexer.HashLeftCurlyBrace, lexer.Newline,
				// Keywords
				lexer.Import, lexer.Func, lexer.For, lexer.When, lexer.Type, lexer.Public:
				insertEOS = false
			case lexer.RightParenthesis, lexer.RightBracket:
			default:
				insertEOS = !canGoOnNewline(p.Tokens[i-1].Kind)
			}
		}
		// Should add EOS before next token?
		if insertEOS && len(p.Tokens) > i+1 && canGoOnNewline(p.Tokens[i+1].Kind) {
			insertEOS = false
		}
		if insertEOS {
			p.Tokens[i].Kind = lexer.EndOfStatement
		} else {
			p.Tokens = slices.Delete(p.Tokens, i, i+1)
			i--
		}
	}
}

// Never add EOS before these tokens, even if on newline. Essentially
// remove the newline.
// Example:
//
//	[1, 2, 3]
//		.sort()
//
// If a newline before is a bad practice (such as equal sign or parenthesis), then it will not be here.
// Tokens that begin statements (such as keywords) aren't here either.
func canGoOnNewline(t lexer.TokenType) bool {
	switch t {
	case
		// Assignment
		lexer.Equal, lexer.ColonEqual, lexer.PlusEqual, lexer.MinusEqual,
		// Arithmetic
		lexer.Plus, lexer.Minus, lexer.Asterisk, lexer.Slash, lexer.Caret,
		lexer.Percent,
		// Punctuation
		lexer.Dot, lexer.RightBracket, lexer.RightParenthesis, lexer.LeftCurlyBrace,
		// Operators
		lexer.Stroke, lexer.Pipeline, lexer.Arrow,
		// Comparison
		lexer.GreaterThan, lexer.LessThan, lexer.EqualEqual, lexer.GreaterEqualTo,
		lexer.LessEqualTo, lexer.NotEqual, lexer.Not, lexer.AndAnd,
		lexer.OrOr,
		// Whitespace
		lexer.Newline:
		return true
	default:
		return false
	}
}
