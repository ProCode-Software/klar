package reporter

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// printSourceLine prints the syntax-highlighted tokens on line. "Pipes" of
// active multiline highlights are printed first. printSourceLine returns
// the end column of the last token on the line.
func (r *Reporter) printSourceLine(s *state, line uint32, firstTokOnLine *int,
	activeHls []klarerrs.Highlight,
) (lastCol uint32) {
	// Print the "pipes" for active multiline highlights
	if len(activeHls) > 0 {
		r.printHighlightPipes(s, activeHls)
	}
	// Now, what you've been waiting for: print the actual tokens!
	lastCol = 1
	var i int
	for i = *firstTokOnLine; i < len(s.tokens); i++ {
		tok := s.tokens[i]
		if tok.Kind == lexer.EOF || tok.Position.Line > line {
			break
		}
		if tok.Source == "\n" && tok.Position.Line == line {
			i++
			break
		}
		end := tok.End()
		if end.Line < line {
			continue
		}
		if tok.Position.Line == line {
			// Add a padding between this and the last token
			if pad := int(tok.Position.Col) - int(lastCol); pad > 0 {
				r.appendSpace(pad)
				lastCol = tok.Position.Col
			} else if pad < 0 {
				// Error with input tokens
				panic(fmt.Sprintf(
					"invalid token offsets: A(%s), B(%s)",
					s.tokens[i-1], tok,
				))
			}
		}
		// Finally!
		lastCol += r.printLineFromToken(s.tokens, i, line)
		if end.Line > line {
			break
		}
	}
	*firstTokOnLine = i
	return
}

// printLineFromToken prints only the part of tok that is on a given line.
// It returns the length of the printed part in runes.
func (r *Reporter) printLineFromToken(tokens []lexer.Token, i int, line uint32) uint32 {
	tok := tokens[i]
	rang := ranges.FromToken(tok)
	errNoNewline := func() {
		panic(fmt.Sprintf(
			"impossible: newline not found in %s token_: %q",
			tok.Kind, tok.Source,
		))
	}
	switch {
	case tok.Kind == lexer.String: // Any string
		return r.colorizeString(tok, line)
	case rang.IsSingleLine(): // Normal token
		r.colorize(tokens, i)
		return tok.Len()
	case rang.Start.Line == line:
		// Fast path for taking the FIRST line
		nl := strings.IndexByte(tok.Source, '\n')
		if nl < 0 {
			errNoNewline()
		}
		// We can just use GetTokenColor because the token is guaranteed
		// to not be an identifier, which is single-line.
		src := tok.Source[:nl]
		r.appendString(src, r.ColorPalette.GetTokenColor(tok.Kind))
		return uint32(utf8.RuneCountInString(src))
	case rang.End.Line == line:
		var src string
		// Fast path for taking the LAST line of a token
		nl := strings.LastIndexByte(tok.Source, '\n') // Last newline
		// Things that should't happen
		if nl < 0 {
			errNoNewline()
		} else if nl < len(tok.Source)-1 {
			// If a newline was the last character, src will be empty
			src = tok.Source[nl+1:]
		}
		r.appendString(src, r.ColorPalette.GetTokenColor(tok.Kind))
		return uint32(utf8.RuneCountInString(src))
	default:
		// Slow path
		lines := strings.Split(tok.Source, "\n")
		src := lines[line-tok.Position.Line]
		r.appendString(src, r.ColorPalette.GetTokenColor(tok.Kind))
		// Line length for normal tokens includes the newline
		// TODO: is this actually the case?
		return uint32(utf8.RuneCountInString(src)) + 1
	}
}
