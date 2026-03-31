package reporter

import (
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// GetTokenColor returns the color for token type k, or an empty
// string if no color is defined for k.
func (p *ColorPalette) GetTokenColor(k lexer.TokenType) string {
	if p.TokenColors != nil {
		return p.TokenColors[k]
	}
	return ""
}

// colorize colors the token at index i in the given slice of tokens,
// using r's ColorPalette's TokenColors for the token's kind and the
// token's context.
func (r *Reporter) colorize(tokens []lexer.Token, i int) {
	tok := tokens[i]
	color := r.ColorPalette.GetTokenColor(tok.Kind)
	// Type/function colors may still be defined
	if color == "" && tok.Kind != lexer.Identifier {
		r.appendString(tok.Source, "")
		return
	}
	// Previous and next token types (for colorizing identifiers)
	var prev, next lexer.TokenType
	if i > 0 {
		prev = tokens[i-1].Kind
	}
	if i < len(tokens)-1 {
		next = tokens[i+1].Kind
	}

	switch {
	case tok.Kind != lexer.Identifier:
		break // Use default color
	case isPrimitive(tok.Source), // Builtin type: Int
		prev == lexer.Arrow && next == lexer.LeftCurlyBrace, // Return type: -> Type {
		prev == lexer.Type,     // Type declaration: type Type
		next == lexer.Stroke,   // Union: Type1 | Type2
		next == lexer.Question: // Optional: Type?
		color = r.ColorPalette.Type
	case next == lexer.LeftParenthesis:
		// Function call: x(...
		if _, ok := builtinFuncs[tok.Source]; ok && r.ColorPalette.BuiltinFunc != "" {
			// Built-in function: print(...
			color = r.ColorPalette.BuiltinFunc
			break
		}
		fallthrough
	case prev == lexer.Func:
		// Function declaration: func x
		color = r.ColorPalette.Function
	}
	r.appendString(tok.Source, color)
}

func isPrimitive(name string) bool {
	_, ok := ast.PrimitiveTypeMap[name]
	return ok
}

// TODO: builtins should be defined somewhere else
var builtinFuncs = map[string]struct{}{
	"print": {}, "crashout": {}, "assert": {}, "TODO": {}, "clone": {},
}

// colorizeString colorizes a string token. It only prints the contents
// of the string that are on the given line number.
func (r *Reporter) colorizeString(t lexer.Token, targetLine uint32) {
	var (
		attrs       = t.Attributes["params"].(lexer.StringAttrs)
		stringColor = r.ColorPalette.GetTokenColor(lexer.String)
		escColor    = r.ColorPalette.StringEscape
		currLine    = t.Position.Line
	)
	if escColor == "" {
		escColor = stringColor
	}
	if targetLine == t.Position.Line {
		r.appendRune(attrs.QuoteStyle, stringColor)
	}
loop:
	for _, frag := range attrs.Fragments {
		switch frag := frag.(type) {
		case lexer.TextFragment:
			// Text fragments end in a newline
			content, hasNewline := strings.CutSuffix(frag.Source, "\n")
			if currLine == targetLine {
				r.appendString(content, stringColor)
			}
			if hasNewline {
				currLine++
				break
			}
		case lexer.StringEscape:
			// String interpolations may be multiline
			for content := range strings.SplitSeq(frag.Value, "\n") {
				if currLine > targetLine {
					// We didn't stop below because there may be only 1
					// iteration if the escape doesn't contain the newline.
					// If this loop runs again, it will reach here and terminate.
					break loop
				}
				if currLine == targetLine {
					r.appendString(content, escColor)
				}
				currLine++
			}
		}
	}
	if !attrs.Unterminated && targetLine == ranges.TokenEnd(t).Line {
		r.appendRune(attrs.QuoteStyle, stringColor)
	}
}
