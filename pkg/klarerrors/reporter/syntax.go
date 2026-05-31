package reporter

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/analysis"
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
		next == lexer.Question, // Optional: Type?
		// Initializer: Capitalized identifier followed by '('
		next == lexer.LeftParenthesis && unicode.IsUpper(firstChar(tok.Source)):
		color = r.ColorPalette.Type
	case next == lexer.LeftParenthesis:
		// Function call: x(...
		if slices.Contains(builtinFuncs, tok.Source) &&
			r.ColorPalette.BuiltinFunc != "" {
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

func firstChar(s string) rune {
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

func isPrimitive(name string) bool {
	_, ok := ast.PrimitiveTypeMap[name]
	return ok
}

// TODO: builtins should be defined somewhere else
var builtinFuncs = analysis.BuiltinFuncs

// colorizeString colorizes a string token. It only prints the contents
// of the string that are on the given line number. It returns the length
// of the line.
func (r *Reporter) colorizeString(t lexer.Token, targetLine uint32) (n uint32) {
	var (
		attrs       = t.Attributes["params"].(lexer.StringAttrs)
		stringColor = r.ColorPalette.GetTokenColor(lexer.String)
		escColor    = r.ColorPalette.StringEscape
		currLine    = t.Position.Line
		firstOnLine = true
	)
	if escColor == "" {
		escColor = stringColor
	}

	// Fast path for single-line strings
	if ranges.FromToken(t).IsSingleLine() {
		r.appendString(t.Source, stringColor)
		return uint32(utf8.RuneCountInString(t.Source))
	}

	if targetLine == t.Position.Line {
		r.appendRune(attrs.QuoteStyle, stringColor)
		n++
	}
loop:
	for _, frag := range attrs.Fragments {
		switch frag := frag.(type) {
		case lexer.TextFragment:
			// Newline would always be the last character of a text fragment
			line, hasNewline := strings.CutSuffix(frag.Source, "\n")
			if currLine == targetLine {
				if frag.LineOffset > 0 {
					// First fragment on line. Add leading spaces because
					// they are not in string fragments.
					r.appendSpace(int(frag.LineOffset))
					n += frag.LineOffset
				}
				r.appendString(line, stringColor)
				n += uint32(utf8.RuneCountInString(line))
				firstOnLine = false
			}
			if hasNewline {
				if currLine++; currLine > targetLine {
					break loop
				}
				firstOnLine = true
			}
		case lexer.StringEscape:
			// String interpolations may be multiline
			var ranOnce bool
			for line := range strings.SplitSeq(frag.Value, "\n") {
				if ranOnce {
					// There may be only 1 iteration if the escape doesn't
					// contain the newline. If this loop runs again, it will
					// reach here and terminate.
					currLine++
				}
				if currLine > targetLine {
					break loop
				}
				if currLine == targetLine {
					// First fragment on line. Add leading spaces for same reason as above
					if firstOnLine {
						r.padding(1, frag.Pos.Col)
						n += frag.Pos.Col - 1
					}
					r.appendString(line, escColor)
					n += uint32(utf8.RuneCountInString(line))
					firstOnLine = false
				}
				ranOnce = true
			}
		}
	}
	if !attrs.Unterminated && targetLine == t.End().Line {
		r.appendRune(attrs.QuoteStyle, stringColor)
		n++
	}
	return n
}
