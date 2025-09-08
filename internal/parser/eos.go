package parser

import (
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// InsertEOS performs automatic semicolon insertion by removing [lexer.Newline]
// tokens from the parser's Tokens and replacing them with [lexer.EndOfStatement] (EOS).
//
// Klar does not use semicolons to terminate statements, and they are invalid.
// Newlines are required to terminate a statement, so InsertEOS will tell
// where a newline is being used to terminate a statement and replace it with
// end of statement. These are equivalent to semicolons in other languages.
//
// Klar's ASI isn't contextual, so the rules stay the same regardless of the expression.
// Because of this, there are some limitations such as EOS tokens always being
// added after '-', '+', '...', or '.' when those tokens are used to begin a statement:
//
//	print(x) // No EOS here
//	-x.toFixed(3) // Same as: print(x) - x.toFixed(3)
//
// Note that most of these kind of statements are invalid in Klar (untyped enum,
// invalid rest, or unused value).
// An EOS token is always added after a [lexer.RightCurlyBrace] '}' token.
func (p *Parser) InsertEOS() {
	if len(p.Tokens) <= 1 {
		return
	}
	for i := 0; i < len(p.Tokens); i++ {
		var (
			prev      lexer.Token
			tok       = p.Tokens[i]
			insertEOS = true
		)
		if i > 0 {
			prev = p.Tokens[i-1]
		}
		switch {
		// Add before EOF
		case tok.Kind == lexer.EOF &&
			prev.Kind != lexer.EndOfStatement:
			fallthrough
		// Block with curly brace on same line:
		// 	func fn(x: Int) { return x * 2 }
		// but not {}
		case tok.Kind == lexer.RightCurlyBrace &&
			prev.Kind != lexer.LeftCurlyBrace &&
			prev.Kind != lexer.HashLeftCurlyBrace &&
			canAddEOSAfter(prev.Kind):
			p.Tokens = slices.Insert(p.Tokens, i, lexer.Token{
				Kind:     lexer.EndOfStatement,
				Position: tok.Position,
			})
			i++
			continue
		case tok.Kind != lexer.Newline:
			continue
		case i > 0:
			insertEOS = canAddEOSAfter(prev.Kind)
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

func (p *Parser) InsertEOSNew() (comments []*ast.Comment) {
	new := make([]lexer.Token, 0, len(p.Tokens))
	var brackets []int
	readComments := func(i int) (nextNonComment int) {
		i++
		for isComment(p.Tokens[i].Kind) {
			comments = append(comments, p.ParseComment(p.Tokens[i]))
			i++
		}
		return i
	}
	for i := 0; i < len(p.Tokens); i++ {
		var (
			tok       = p.Tokens[i]
			kind      = tok.Kind
			prev      lexer.TokenType
			insertEOS = true
		)
		if len(new) > 0 {
			prev = new[len(new)-1].Kind
		}
		switch kind {
		// Comment
		case lexer.BlockComment, lexer.LineComment, lexer.Hashbang:
			comments = append(comments, p.ParseComment(tok))
			continue
		// Mark start position for brackets
		case lexer.LeftBracket, lexer.LeftCurlyBrace, lexer.HashLeftCurlyBrace,
			lexer.LeftParenthesis:
			brackets = append(brackets, len(new))
		case lexer.EOF:
			// Add before EOF
			if i > 0 && prev != lexer.EndOfStatement && canAddEOSAfter(prev) {
				new = append(new, lexer.Token{
					Kind:     lexer.EndOfStatement,
					Position: tok.Position,
				})
			}
		// Always add EOS after '}' unless empty in case it is on the same line
		// as an expression: { x + 3 }
		case lexer.RightCurlyBrace:
			if i > 0 && prev != lexer.LeftCurlyBrace &&
				prev != lexer.HashLeftCurlyBrace && canAddEOSAfter(prev) {
				new = append(new, lexer.Token{
					Kind:     lexer.EndOfStatement,
					Position: tok.Position,
				})
			}
			fallthrough
		case lexer.RightBracket, lexer.RightParenthesis:
			i = readComments(i)
			// Skip newlines
			for p.Tokens[i].Kind == lexer.Newline {
				i++
				i = readComments(i)
			}
			if arr := p.Tokens[i]; arr.Kind == lexer.Arrow {
				lastBrackI := len(brackets) - 1
				p.lambdaTokens[brackets[lastBrackI]] = struct{}{}
				// Remove the bracket from the array
				brackets = brackets[:lastBrackI]
				// Don't reparse the arrow
				new = append(new, arr)
				continue
			}
			i-- // Continuing the loop
		}
		if kind != lexer.Newline {
			new = append(new, tok)
			continue
		}
		if i > 0 {
			insertEOS = canAddEOSAfter(prev)
		}
		nextTokI := readComments(i)
		// Should add EOS before next token?
		if insertEOS && canGoOnNewline(p.Tokens[nextTokI].Kind) {
			insertEOS = false
		}
		if insertEOS {
			tok.Kind = lexer.EndOfStatement
			new = append(new, tok)
		}
		i = nextTokI - 1 // Continuing the loop
	}
	p.Tokens = new
	return
}

// Never add EOS after these tokens. All of the handled tokens are NUDs, otherwise
// an EOS is added if canGoOnNewline(t) returns false.
func canAddEOSAfter(t lexer.TokenType) bool {
	switch t {
	case
		// Punctuation
		lexer.LeftBracket, lexer.LeftCurlyBrace,
		lexer.LeftParenthesis, lexer.Colon, lexer.EndOfStatement,
		lexer.HashLeftCurlyBrace, lexer.Newline,
		// Keywords
		lexer.Import, lexer.Func, lexer.For, lexer.When, lexer.Type,
		lexer.Go, lexer.While, lexer.Can, lexer.NotCan:
		return false
	case lexer.RightParenthesis, lexer.RightBracket:
		return true
	default:
		return !canGoOnNewline(t)
	}
}

// Never add EOS before (or after) these tokens, even if on newline. Essentially
// remove the newline.
// Example:
//
//	[1, 2, 3]
//		.sort()
//
// If a newline before is a bad practice (such as parenthesis), then it will not be here.
// Tokens that begin statements (such as keywords) aren't here either. [lexer.Newline]
// is included and return true so that extra newlines are removed. Apart from that,
// all of the handled tokens are LEDs.
func canGoOnNewline(t lexer.TokenType) bool {
	switch t {
	case
		// Assignment
		lexer.Equal, lexer.ColonEqual, lexer.PlusEqual, lexer.MinusEqual,
		// Arithmetic
		lexer.Plus, lexer.Minus, lexer.Asterisk, lexer.Slash, lexer.Caret,
		lexer.Percent,
		// Distributive
		lexer.And, lexer.Or,
		// Punctuation
		lexer.Dot, lexer.RightBracket, lexer.RightParenthesis, lexer.LeftCurlyBrace,
		lexer.Comma,
		// Operators
		lexer.Stroke, lexer.Pipeline, lexer.Arrow, lexer.StrokeDot, lexer.Ellipsis,
		lexer.DotDotLessThan,
		// Comparison
		lexer.GreaterThan, lexer.LessThan, lexer.EqualEqual, lexer.GreaterEqualTo,
		lexer.LessEqualTo, lexer.NotEqual, lexer.Not, lexer.AndAnd,
		lexer.OrOr, lexer.In, lexer.NotIn,
		// Whitespace
		lexer.Newline:
		return true
	default:
		return false
	}
}

func isComment(t lexer.TokenType) bool {
	switch t {
	case lexer.BlockComment, lexer.LineComment, lexer.Hashbang:
		return true
	}
	return false
}

func (p *Parser) ParseComment(tok lexer.Token) *ast.Comment {
	switch {
	case tok.Kind == lexer.Hashbang:
		if tok.Position != (lexer.Position{1, 1}) {
			p.Error(errors.Token(errors.ErrMisplacedShebang, tok))
		}
	case tok.Attributes["unterm"] == true:
		p.Error(errors.ParseError{
			ErrorCode: errors.ErrUnterminatedComment,
			Token:     tok,
			Position:  tok.Position,
		})
	}
	return &ast.Comment{
		Value:    tok.Source,
		Type:     tok.Kind,
		BaseNode: ast.BaseNode{ranges.FromToken(tok)},
	}
}
