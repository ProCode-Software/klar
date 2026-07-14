package parser

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// InsertEOS performs statement termination inference by identifying which
// [lexer.Newline] tokens are used as End of Statement (EOS) markers.
//
// Klar does not use semicolons to terminate statements, and they are invalid.
// Newlines are required to terminate a statement, so InsertEOS will tell
// where a newline is being used to terminate a statement and replace it with
// end of statement. These are equivalent to semicolons in other languages.
//
// Klar's ASI isn't contextual, so the rules stay the same regardless of the expression.
// Because of this, there are some limitations such as EOS tokens always being
// added after '-' or '.' when those tokens are used to begin a statement:
//
//	print(x) // No EOS here
//	-x.toFixed(3) // Same as: print(x) - x.toFixed(3)
//
// Note that most of these kind of statements are invalid in Klar (untyped enum,
// invalid rest, or unused value).
// An EOS token is always added after a [lexer.RightCurlyBrace] '}' token.
func (p *Parser) InsertEOS() (comments []*ast.Comment) {
	new := make([]lexer.Token, 0, len(p.Tokens))
	brackets := make([]int, 0, len(p.Tokens)/8)
	p.listCastTokens = make(map[int]struct{}) // Keep track of list cast tokens

	// Parses all consecutive comments and returns the next non-comment token
	readComments := func(i int) (nextNonComment int) {
		i++
		for isComment(p.Tokens[i].Kind) {
			comments = append(comments, p.ParseComment(p.Tokens[i]))
			i++
		}
		return i
	}
outer:
	for i := 0; i < len(p.Tokens); i++ {
		var (
			tok  = p.Tokens[i]
			kind = tok.Kind
			prev lexer.TokenType
		)
		if len(new) > 0 {
			prev = new[len(new)-1].Kind
		}
		switch kind {
		case lexer.BlockComment, lexer.LineComment, lexer.Hashbang:
			i = readComments(i) - 1
			continue
		case lexer.LeftBracket:
			// Mark start position for brackets
			brackets = append(brackets, len(new))
		case lexer.Not:
			// Merge '!' + 'in' -> NotIn '!in'
			if i+1 >= len(p.Tokens) {
				break
			}
			next := p.Tokens[i+1]
			if next.Kind == lexer.In && next.Position.HasOffset(tok.Position, 0, 1) {
				new = append(new, lexer.Token{
					Kind:     lexer.NotIn,
					Source:   tok.Source + next.Source, // "!in"
					Position: tok.Position,
				})
				i++ // Skip 'in'
				continue
			}
		case lexer.EOF:
			// Add newline before EOF
			if i > 0 && prev != lexer.Newline && CanEndStatement(prev) {
				new = append(new, lexer.Token{
					Kind:     lexer.Newline,
					Position: tok.Position,
				})
			}
			new = append(new, tok)
			if i != len(p.Tokens)-1 {
				panic("EOF must be the last token")
			}
			break outer
		// Always add EOS after '}' unless empty in case it is on the same line
		// as an expression: { x + 3 }
		case lexer.RightCurlyBrace:
			if i > 0 && prev != lexer.LeftCurlyBrace &&
				prev != lexer.HashLeftCurlyBrace && CanEndStatement(prev) {
				new = append(new, lexer.Token{
					Kind:     lexer.Newline,
					Position: tok.Position,
				})
			}
			// TODO: Map casts
		case lexer.RightBracket:
			newI := readComments(i)
			lastBrackI := len(brackets) - 1
			if lastBrackI < 0 { // Unmatched bracket
				break
			}
			// List or map cast: [Int](...), #{String: Int}(...)
			if newI < len(p.Tokens) && p.Tokens[newI].Kind == lexer.LeftParenthesis {
				p.listCastTokens[brackets[lastBrackI]] = struct{}{}
				brackets = brackets[:lastBrackI] // Remove bracket
				new = append(new, tok, p.Tokens[newI])
				i = newI // Skip '('
				continue
			}
			brackets = brackets[:lastBrackI] // Remove bracket
		}
		if kind != lexer.Newline {
			new = append(new, tok)
			continue
		}
		// This is where the actual semicolon insertion process happens!
		var (
			// If a newline follows a keyword like 'func', don't add a newline
			continuesBefore = i == 0 || CanEndStatement(prev)
			nextTokI        = readComments(i)
			continuesStmt   = nextTokI < len(p.Tokens) &&
				ContinuesStatement(p.Tokens[nextTokI].Kind)
		)
		// The first consecutive newline is preserved
		// (ContinuesStatement(Newline) == true). If this is currently
		// the last one, ensure that newline is actually allowed after.
		// If not, remove that newline.
		if prev == lexer.Newline && continuesStmt {
			new = new[:len(new)-1]
		}
		// Add this EOS if the next token can't have a newline before
		// (for example, if the next token is '+', continuesStmt == true)
		if continuesBefore && !continuesStmt {
			new = append(new, tok)
		}
		i = nextTokI - 1 // Continuing the loop, skipping comments
	}
	// Add EOF if not present
	if len(new) > 0 && new[len(new)-1].Kind != lexer.EOF {
		panic("EOF not present in input p.Tokens")
	}
	p.Tokens = new
	brackets = nil
	return comments
}

// Never add EOS after these tokens. An EOS is added if [ContinuesStatement](t)
// returns false.
func CanEndStatement(t lexer.TokenType) bool {
	// All handled tokens are NUDs
	switch t {
	case
		// Punctuation
		lexer.LeftBracket, lexer.LeftCurlyBrace,
		lexer.LeftParenthesis, lexer.Colon, lexer.Newline,
		lexer.HashLeftCurlyBrace,
		// Keywords
		lexer.Func, lexer.For, lexer.When, lexer.Type,
		lexer.Go, lexer.Await, lexer.While, lexer.Not,
		lexer.Try, lexer.Readonly, lexer.Public:
		return false
	case lexer.RightParenthesis, lexer.RightBracket,
		lexer.NotNot, lexer.GreaterThan:
		return true
	default:
		return !ContinuesStatement(t)
	}
}

// Never add EOS before (or after) these tokens, even if on newline. Essentially
// remove the newline.
// Example:
//
//	[1, 2, 3]
//		.sort()
//
// If a newline before is a bad practice (such as parenthesis), then it will not
// be here. Tokens that begin statements (such as keywords) aren't here either.
// [lexer.Newline] is included and returns true so that extra newlines are removed.
func ContinuesStatement(t lexer.TokenType) bool {
	switch t {
	case
		// Assignment
		lexer.Equal, lexer.ColonEqual, lexer.PlusEqual, lexer.MinusEqual,
		lexer.AsteriskEqual, lexer.SlashEqual, lexer.PercentEqual, lexer.CaretEqual,
		// Arithmetic
		lexer.Plus, lexer.Minus, lexer.Asterisk, lexer.Slash, lexer.Caret,
		lexer.Percent,
		// Distributive
		lexer.And, lexer.Or,
		// Punctuation
		lexer.Dot, lexer.RightBracket, lexer.RightParenthesis, lexer.Comma,
		// Operators
		lexer.Stroke, lexer.Pipeline, lexer.Arrow, lexer.StrokeDot, lexer.Ellipsis,
		lexer.DotDotLessThan, lexer.NotNot,
		// Comparison
		lexer.GreaterThan, lexer.LessThan, lexer.EqualEqual, lexer.GreaterEqualTo,
		lexer.LessEqualTo, lexer.NotEqual, lexer.AndAnd,
		lexer.OrOr, lexer.In, lexer.NotIn, lexer.If:
		// Whitespace
		// lexer.Newline:
		return true
	default:
		return false
	}
}
