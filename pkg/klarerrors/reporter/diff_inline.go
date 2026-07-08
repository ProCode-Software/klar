package reporter

import (
	"cmp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/char"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// printInlineDiffLine prints a line with inline diff underlines. Inline diffs are where diff
// ranges, both additions and deletions, are displayed alongside the original source.
func (r *Reporter) printInlineDiffLine(s *diffState, dl *diffLine) (lastLine uint32) {
	// Sort by column position, then deletions first
	slices.SortFunc(dl.ranges, sortDiffEdits)
	var (
		orig         = r.getTokensOnLine(s, dl.line)
		merged, last = r.makeMergedTokens(dl.line, orig, dl.ranges)
		firstOnLine  int // Index of the first token on the current line
	)
	// Print the merged line(s) with both additions and removals highlighted
	for line := dl.line; line <= last; line++ {
		r.printDiffLine(s, line, merged, &firstOnLine, false)
	}

	for s.lastReadTok < len(s.tokens) &&
		s.tokens[s.lastReadTok].End().Line <= last {
		s.lastReadTok++
	}
	return last
}

// sortDiffEdits sorts inline edits by their column position. If two edits
// begin at the same position, deletions are sorted before additions.
func sortDiffEdits(a, b klarerrs.DiffEdit) int {
	if colOrder := cmp.Compare(a.Start().Col, b.Start().Col); colOrder != 0 {
		return colOrder
	}
	if !a.Operation() && b.Operation() {
		return -1
	}
	if a.Operation() && !b.Operation() {
		return 1
	}
	return 0
}

// getTokensOnLine returns the tokens from the original source that intersect with line.
func (r *Reporter) getTokensOnLine(s *diffState, line uint32) []lexer.Token {
	for i := s.lastReadTok; i < len(s.tokens); i++ {
		tok := s.tokens[i]
		if tok.Position.Line > line {
			return s.tokens[s.lastReadTok:i]
		}
		if tok.End().Line < line {
			s.lastReadTok = i + 1
			continue
		}
	}
	return s.tokens[s.lastReadTok:]
}

// makeMergedTokens merges the original tokens on the current line with the diff
// edits to create a unified line that displays both additions and deletions.
// The end line of the last token that begins on this line (may not be the same
// as line) is returned.
func (r *Reporter) makeMergedTokens(
	line uint32, orig []lexer.Token, edits []klarerrs.DiffEdit,
) (merged []lexer.Token, lastLine uint32) {
	var (
		currEdit int
		lastCol  uint32 = 1
		vpos            = lexer.Position{line, 1} // New position after additions
	)
	merged = make([]lexer.Token, 0, len(orig)+len(edits))
	addToken := func(tok lexer.Token) {
		// TODO: Test this for multiline tokens
		_, n := extractLine(tok, line)
		// If the tokens starts after what we currently have, use the token's position
		if tok.Position.Col > vpos.Col {
			vpos = tok.Position
			lastCol = tok.Position.Col
		}
		tok.Position = vpos
		vpos.Col += n
		vpos.Line += tok.End().Line - vpos.Line
		merged = append(merged, tok)
	}
	addAddition := func(e klarerrs.DiffEdit) {
		switch e := e.(type) {
		case klarerrs.AddedString:
			addToken(addedStringToToken(e))
		case klarerrs.AddedTokens:
			for _, tok := range e.Tokens {
				tok.Kind = addedToken
				addToken(tok)
			}
		}
		currEdit++
	}

	for _, tok := range orig {
		// Check if this token was deleted
		isDeleted := slices.ContainsFunc(edits, func(edit klarerrs.DiffEdit) bool {
			del, ok := edit.(klarerrs.DeletedRange)
			return ok && del.Range.TokenIntersects(tok)
		})
		if isDeleted {
			tok.Kind = deletedToken
		}
		// Insert additions starting at or before the current column. If the token was
		// deleted, only additions BEFORE it are added.
		for currEdit < len(edits) && edits[currEdit].Start().Col <= tok.Position.Col {
			if isDeleted && edits[currEdit].Start().Col == tok.Position.Col {
				break
			}
			addAddition(edits[currEdit])
		}
		// Add the offset from the last actual token
		if lastCol < tok.Position.Col {
			vpos.Col += tok.Position.Col - lastCol
		}
		// Add the original token
		addToken(tok)
		lastCol = tok.Position.Col + tok.Len()
	}
	// There may be more additions after the last source token
	//
	//  func count(first, last: Int) = last - first + 1
	//                                              +++
	for currEdit < len(edits) {
		addAddition(edits[currEdit])
	}

	lastLine = line
	if len(merged) > 0 {
		// TODO: Should this be the end position of the token?
		// And is this needed, or lastLine always == line?
		lastLine = merged[len(merged)-1].Position.Line
	}
	return
}

func addedStringToToken(e klarerrs.AddedString) lexer.Token {
	return lexer.Token{
		Kind:       addedToken,
		Source:     e.String,
		Position:   e.Position,
		Attributes: map[string]any{"length": uint32(utf8.RuneCountInString(e.String))},
	}
}

// printDiffLine prints a single line of a diff with its syntax-highlighted tokens
// and diff underscores.
func (r *Reporter) printDiffLine(s *diffState, line uint32, tokens []lexer.Token,
	firstTokOnLine *int, op bool,
) {
	r.printDiffLineNumber(s, line, op, false)
	r.printSourceLine(&state{tokens: tokens}, line, firstTokOnLine, nil)
	r.newline()
	r.printDiffUnderlines(s, tokens, line)
}

// printDiffUnderlines adds the +/- underlines for each line.
func (r *Reporter) printDiffUnderlines(s *diffState, tokens []lexer.Token, line uint32) {
	// First check if any token on this line is a diff token
	if !slices.ContainsFunc(tokens, func(tok lexer.Token) bool {
		return ranges.FromToken(tok).LineIn(line) &&
			(tok.Kind == addedToken || tok.Kind == deletedToken)
	}) {
		return
	}

	// Line number prefix
	r.appendSpace(hintMargin + s.digitWidth + 1)
	r.appendf(r.ColorPalette.Box, "%c ", r.CharacterSet.HighlightLine)

	var lastCol uint32 = 1
	for _, tok := range tokens {
		if tok.Position.Line > line {
			break
		}
		end := tok.End()
		if end.Line < line {
			continue
		}
		if tok.Position.Line == line {
			// Padding between tokens
			r.padding(lastCol, tok.Position.Col)
			lastCol = tok.Position.Col
		}
		// Calculate length on this line
		ulLen := r.tokenLenOnLine(tok, line)
		switch tok.Kind {
		case deletedToken:
			r.appendf(r.ColorPalette.DiffDelete, "%s", char.Repeat('-', int(ulLen)))
		case addedToken:
			r.appendf(r.ColorPalette.DiffAdd, "%s", char.Repeat('+', int(ulLen)))
		default:
			r.appendSpace(int(ulLen))
		}
		lastCol += ulLen
		if end.Line > line {
			break
		}
	}
	r.newline()
}

// tokenLenOnLine returns the length of the part of tok that is on line.
func (r *Reporter) tokenLenOnLine(tok lexer.Token, line uint32) uint32 {
	rang := ranges.FromToken(tok)
	if rang.IsSingleLine() {
		return uint32(utf8.RuneCountInString(tok.Source))
	}
	// Multi-line token
	if line < tok.Position.Line {
		return 0
	}
	currLine := tok.Position.Line
	for srcLine := range strings.SplitSeq(tok.Source, "\n") {
		if currLine == line {
			return uint32(utf8.RuneCountInString(srcLine))
		}
		currLine++
	}
	return 0 // Token ends before `line`
}
