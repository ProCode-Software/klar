package reporter

import (
	"fmt"
	"strings"

	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// printSourceLine prints the syntax-highlighted tokens on line. "Pipes" of active multiline highlights are printed first. printSourceLine returns the end column of the last token on the line.
func (r *Reporter) printSourceLine(s *state, line uint32, firstTokOnLine *int,
	activeHls []errors.Highlight,
) (lastCol uint32) {
	// Print the "pipes" for active multiline highlights
	if len(activeHls) > 0 {
		r.printHighlightPipes(s, activeHls)
	}
	// Now, what you've been waiting for: print the actual tokens!
	lastCol = 1
	var i int
	for i = *firstTokOnLine; i < len(s.tokens) && s.tokens[i].Position.Line <= line; i++ {
		tok := s.tokens[i]
		if tok.Source == "\n" {
			continue // Don't print newlines
		}
		end := ranges.TokenEnd(tok)
		// Add a padding, unless the token started on a previous line
		if tok.Position.Line >= line {
			if padding := int(tok.Position.Col) - int(lastCol); padding > 0 {
				r.appendSpace(padding)
			} else if padding < 0 {
				// Error with input tokens
				panic(fmt.Sprintf("invalid token offsets: A(%s), B(%s)", s.tokens[i-1], tok))
			}
		}
		r.printLineFromToken(s.tokens, i, line) // Finally!
		lastCol = end.Col
		if end.Line > line {
			break
		}
	}
	*firstTokOnLine = i
	return
}

// printLineFromToken prints only the part of tok that is on a given line.
func (r *Reporter) printLineFromToken(tokens []lexer.Token, i int, line uint32) {
	tok := tokens[i]
	tokRange := ranges.FromToken(tok)
	errNoNewline := func() {
		panic(fmt.Sprintf("impossible: newline not found in %s token: %q",
			tok.Kind, tok.Source,
		))
	}
	switch {
	case tok.Kind == lexer.String: // Any string
		r.colorizeString(tok, line)
	case tokRange.IsSingleLine(): // Normal token
		r.colorize(tokens, i)
	case tokRange.Start.Line == line:
		// Fast path for taking the FIRST line
		nl := strings.IndexByte(tok.Source, '\n')
		if nl < 0 {
			errNoNewline()
		}
		// We can just use GetTokenColor because the token is guaranteed
		// to not be an identifier, which is single-line.
		r.appendString(tok.Source[:nl], r.ColorPalette.GetTokenColor(tok.Kind))
	case tokRange.End.Line == line:
		// Fast path for taking the LAST line of a token
		nl := strings.LastIndexByte(tok.Source, '\n') // Last newline
		// Things that should't happen
		if nl < 0 {
			errNoNewline()
		} else if nl == len(tok.Source)-1 {
			panic(fmt.Sprintf("impossible: newline is last character in %s token: %q",
				tok.Kind, tok.Source,
			))
		}
		r.appendString(tok.Source[nl+1:], r.ColorPalette.GetTokenColor(tok.Kind))
	default:
		// Slow path
		lines := strings.Split(tok.Source, "\n")
		r.appendString(lines[line-tok.Position.Line], r.ColorPalette.GetTokenColor(tok.Kind))
	}
}
