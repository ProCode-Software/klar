package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
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
func (p *Parser) InsertEOS() (comments []*ast.Comment) {
	var (
		new      = make([]lexer.Token, 0, len(p.Tokens))
		brackets = make([]int, 0, len(p.Tokens)/8)
		assign   = make([]int, 0, len(p.Tokens)/12)
	)
	readComments := func(i int) (nextNonComment int) {
		i++
		for isComment(p.Tokens[i].Kind) {
			comments = append(comments, p.ParseComment(p.Tokens[i]))
			i++
		}
		return i
	}
	// Setup cached tokens
	p.assignmentTokens = make(map[int]struct{})
	p.lambdaTokens = make(map[int]struct{})
	p.listCastTokens = make(map[int]struct{})

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
		switch {
		// Comment
		case isComment(kind):
			comments = append(comments, p.ParseComment(tok))
			continue
		// TODO: cache assignment (not done)
		// edge case: #{ a, b, c }[a] = 2 doesn't work
		// keep a global bracket level count
		case isAssignment(kind), kind == lexer.Colon, kind == lexer.In:
			if len(assign) > 0 {
				p.assignmentTokens[assign[len(assign)-1]] = struct{}{}
			}
		// Add tokens that go before a destructure assignment here
		case prev == lexer.EndOfStatement, prev == lexer.For, len(new) == 0:
			assign = append(assign, len(new))
		}
		switch kind {
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
			var (
				newI         = readComments(i)
				lastBrackI   = len(brackets) - 1
				firstNewline int // Cannot be zero because preceded by bracket
			)
			// List cast: [Int](...)
			if kind == lexer.RightBracket && p.Tokens[newI].Kind == lexer.LeftParenthesis {
				p.listCastTokens[brackets[lastBrackI]] = struct{}{}
			}
			// Skip newlines
			if p.Tokens[newI].Kind == lexer.Newline {
				firstNewline = newI
				for p.Tokens[newI].Kind == lexer.Newline {
					newI = readComments(newI)
				}
			}
			new = append(new, tok)
			next := p.Tokens[newI]
			// Check for '->' (arrow function)
			if next.Kind == lexer.Arrow {
				p.lambdaTokens[brackets[lastBrackI]] = struct{}{} // Cache it
				// Remove the bracket from the array
				brackets = brackets[:lastBrackI]
				// Don't reparse the arrow
				new = append(new, next)
				i = newI
			} else if firstNewline > 0 && !canGoOnNewline(next.Kind) {
				// Still add the EOS
				newTok := p.Tokens[firstNewline]
				newTok.Kind = lexer.EndOfStatement
				new = append(new, newTok)
				i = newI - 1
			}
			continue // Already appended above
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
	p.Tokens = new[:len(new):len(new)]
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
