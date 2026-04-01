package reporter

import (
	"cmp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// printInline prints a line with inline diff highlights.
// It displays both the "before" state (with deletions marked) and the "after"
// state (with additions inserted), using printDiffUnderlines to highlight the changes.
func (r *Reporter) printInline(s *diffState, dl *diffLine) (lastLine uint32) {
	// Sort by column position, then deletions first
	slices.SortFunc(dl.ranges, sortDiffEdits)
	var (
		orig         = r.getOriginalTokens(s, dl.line)
		merged, last = r.buildMergedTokens(dl.line, orig, dl.ranges)
		first        int
	)
	// Print the merged line(s) with both additions and removals highlighted
	for line := dl.line; line <= last; line++ {
		r.printDiffLine(s, line, merged, &first, false)
	}

	for s.lastReadTok < len(s.tokens) &&
		s.tokens[s.lastReadTok].End().Line <= last {
		s.lastReadTok++
	}
	return last
}

// sortDiffEdits sorts inline edits by their column position, ensuring that
// deletions are processed before additions at the same column.
func sortDiffEdits(a, b errors.DiffEdit) int {
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

// getOriginalTokens returns the tokens from the original source that intersect with line.
func (r *Reporter) getOriginalTokens(s *diffState, line uint32) (orig []lexer.Token) {
	for i := s.lastReadTok; i < len(s.tokens); i++ {
		tok := s.tokens[i]
		if tok.Position.Line > line {
			break
		}
		if tok.End().Line < line {
			s.lastReadTok = i + 1
			continue
		}
		orig = append(orig, tok)
	}
	return
}

// buildMergedTokens creates a set of virtual tokens representing the merged "before" and "after"
// states of the line, adjusting positions and handling multi-line additions.
func (r *Reporter) buildMergedTokens(line uint32, orig []lexer.Token, edits []errors.DiffEdit) (
	merged []lexer.Token, lastLine uint32,
) {
	var (
		editI              int
		currentOriginalCol uint32 = 1
		virtualPos                = lexer.Position{Line: line, Col: 1}
	)
	addToken := func(tok lexer.Token) {
		tok.Position = virtualPos
		merged = append(merged, tok)
		if strings.Contains(tok.Source, "\n") {
			parts := strings.Split(tok.Source, "\n")
			virtualPos.Line += uint32(len(parts) - 1)
			virtualPos.Col = uint32(utf8.RuneCountInString(parts[len(parts)-1])) + 1
		} else {
			virtualPos.Col += uint32(utf8.RuneCountInString(tok.Source))
		}
	}
	for _, tok := range orig {
		// Insert additions that start at or before this token's column
		for editI < len(edits) && edits[editI].Start().Col <= tok.Position.Col {
			edit := edits[editI]
			if edit.Operation() {
				switch e := edit.(type) {
				case errors.AddedString:
					addToken(lexer.Token{Kind: addedToken, Source: e.String})
				case errors.AddedTokens:
					for _, t := range e.Tokens {
						t.Kind = addedToken
						addToken(t)
					}
				}
			}
			editI++
		}
		// Check if this token was deleted
		isDeleted := false
		for _, edit := range edits {
			if dr, ok := edit.(errors.DeletedRange); ok && dr.Range.TokenIntersects(tok) {
				isDeleted = true
				break
			}
		}
		if isDeleted {
			tok.Kind = deletedToken
			addToken(tok)
		} else {
			if tok.Position.Col > currentOriginalCol {
				virtualPos.Col += (tok.Position.Col - currentOriginalCol)
			}
			addToken(tok)
		}
		currentOriginalCol = tok.Position.Col + uint32(utf8.RuneCountInString(tok.Source))
	}
	// Append remaining additions at the end of the line
	for editI < len(edits) {
		edit := edits[editI]
		if edit.Operation() {
			switch e := edit.(type) {
			case errors.AddedString:
				addToken(lexer.Token{Kind: addedToken, Source: e.String})
			case errors.AddedTokens:
				for _, t := range e.Tokens {
					t.Kind = addedToken
					addToken(t)
				}
			}
		}
		editI++
	}
	var maxLine uint32 = line
	for _, t := range merged {
		maxLine = max(maxLine, t.Position.Line)
	}
	return merged, maxLine
}

// printDiffLine prints a single line of a diff with its syntax-highlighted tokens
// and diff underscores.
func (r *Reporter) printDiffLine(s *diffState, line uint32, tokens []lexer.Token, first *int, added bool) {
	r.printDiffLineNumber(s, line, added, false)
	r.printSourceLine(&state{tokens: tokens}, line, first, nil)
	r.newline()
	r.printDiffUnderlines(s, tokens, line)
}

// printDiffUnderlines adds the +/- underlines for each line.
func (r *Reporter) printDiffUnderlines(s *diffState, tokens []lexer.Token, line uint32) {
	// First check if any token on this line is a diff token
	var hasChanges bool
	for _, tok := range tokens {
		if tok.Position.Line <= line && tok.End().Line >= line {
			if tok.Kind == addedToken || tok.Kind == deletedToken {
				hasChanges = true
				break
			}
		}
	}
	if !hasChanges {
		return
	}

	// Line number prefix (similar to printEmptyLineNumber)
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
			if pad := int(tok.Position.Col) - int(lastCol); pad > 0 {
				r.appendSpace(pad)
				lastCol = tok.Position.Col
			}
		}

		// Calculate length on this line
		partLen := r.tokenLenOnLine(tok, line)
		switch tok.Kind {
		case deletedToken:
			r.appendString(strings.Repeat("-", int(partLen)), r.ColorPalette.DiffDelete)
		case addedToken:
			r.appendString(strings.Repeat("+", int(partLen)), r.ColorPalette.DiffAdd)
		default:
			r.appendSpace(int(partLen))
		}
		lastCol += partLen

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
	// For multiline tokens, find the relevant line
	lines := strings.Split(tok.Source, "\n")
	idx := int(line - tok.Position.Line)
	if idx < 0 || idx >= len(lines) {
		return 0
	}
	return uint32(utf8.RuneCountInString(lines[idx]))
}
