package reporter

import (
	"cmp"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// diffState tracks the state of the diff reporter, including the current token
// stream and the width of the line numbers.
type diffState struct {
	digitWidth  int
	tokens      []lexer.Token
	lastReadTok int
}

// diffLine groups diff operations that apply to a single line.
type diffLine struct {
	line        uint32
	addedLine   *errors.DiffEdit
	deletedLine *errors.DeletedRange
	ranges      []errors.DiffEdit
}

const (
	deletedToken lexer.TokenType = -69
	addedToken                   = deletedToken - 1
)

// printDiff formats and prints all line changes and highlights described in the diff.
func (r *Reporter) printDiff(diff *errors.Diff) {
	var (
		lines, end = r.groupDiffLines(diff)
		digitWidth = digitLen(end)
		state      = &diffState{
			digitWidth: digitWidth,
			tokens:     diff.Tokens,
		}
	)
	var lastLine uint32
	for _, lineNum := range slices.Sorted(maps.Keys(lines)) {
		if lineNum <= lastLine {
			continue
		}
		dl := lines[lineNum]
		// Order: 1. Full-line removals, 2. Full-line additions
		// Or: Inline
		if dl.deletedLine != nil {
			lastLine = r.printFullRemove(state, dl)
			continue
		}
		if dl.addedLine != nil {
			lastLine = r.printFullAdd(state, dl)
			continue
		}
		lastLine = r.printInline(state, dl)
	}
}

// groupDiffLines organizes disjoint diff ranges by line number so they can be accurately displayed together.
func (r *Reporter) groupDiffLines(diff *errors.Diff) (
	lines map[uint32]*diffLine, end uint32,
) {
	lines = make(map[uint32]*diffLine)
	for _, edit := range diff.Edits {
		lineNum := edit.Start().Line
		if _, ok := lines[lineNum]; !ok {
			lines[lineNum] = &diffLine{line: lineNum}
		}
		dl := lines[lineNum]
		end = max(end, edit.EndLine())
		if !edit.FullLine() {
			dl.ranges = append(dl.ranges, edit)
			continue
		}
		switch edit := edit.(type) {
		case errors.DeletedRange:
			dl.deletedLine = &edit
		case errors.AddedTokens, errors.AddedString:
			dl.addedLine = &edit
		}
	}
	return
}

// printFullRemove prints a block of source code representing a full-line deletion.
func (r *Reporter) printFullRemove(s *diffState, dl *diffLine) (lastLine uint32) {
	edit := *dl.deletedLine
	// Get intersecting tokens
	var firstTokI, maxTokI int = -1, len(s.tokens)
	for i, tok := range s.tokens[s.lastReadTok:] {
		if edit.Range.TokenIntersects(tok) {
			if firstTokI < 0 {
				firstTokI = i
			}
		} else {
			maxTokI = i
			lastLine = ranges.TokenEnd(s.tokens[i-1]).Line
			s.lastReadTok += i - 1
			break
		}
	}
	srcState := &state{tokens: s.tokens[firstTokI:maxTokI]}
	hlColor := r.ColorPalette.DiffDelete + r.ColorPalette.DiffDeleteBackground
	r.printDiffLineNumber(s, dl.line, false, true)
	// TODO: make printSourceLine highlight
	r.printSourceLine(srcState, dl.line, &s.lastReadTok, nil)
	_ = hlColor
	return
}

// printFullAdd prints a block of source code or strings representing a
// full-line addition.
func (r *Reporter) printFullAdd(s *diffState, dl *diffLine) (lastLine uint32) {
	hlColor := r.ColorPalette.DiffAdd + r.ColorPalette.DiffAddBackground
	switch edit := (*dl.addedLine).(type) {
	case errors.AddedString:
		// Multiline added string
		if edit.NumLines > 1 {
			for line := range strings.SplitSeq(edit.String, "\n") {
				r.printDiffLineNumber(s, dl.line, true, true)
				r.appendString(line, hlColor)
				r.newline()
			}
			return dl.line + edit.NumLines - 1
		}
		r.printDiffLineNumber(s, dl.line, true, true)
		r.appendString(edit.String, hlColor)
		r.newline()
		return dl.line
	case errors.AddedTokens:
		// TODO
		srcState := &state{tokens: edit.Tokens}
		r.printDiffLineNumber(s, dl.line, true, true)
		var firstTokOnLine int // edit.Position.Col
		r.printSourceLine(srcState, dl.line, &firstTokOnLine, nil)
		println("first on next line", firstTokOnLine) // TODO: print more lines
		r.newline()
	}
	return
}

// printInline prints a line with inline diff highlights.
// It displays both the "before" state (with deletions marked) and the "after"
// state (with additions inserted), using printDiffUnderlines to highlight the changes.
func (r *Reporter) printInline(s *diffState, dl *diffLine) (lastLine uint32) {
	r.sortDiffEdits(dl.ranges)
	var (
		orig         = r.getOriginalTokens(s, dl.line)
		merged, last = r.buildMergedTokens(dl.line, orig, dl.ranges)
		first        int
	)
	// Print the merged line(s) with both additions and removals highlighted
	for l := dl.line; l <= last; l++ {
		r.printDiffLine(s, l, merged, &first, false)
	}
	s.finishLine(dl.line)
	return last
}

// sortDiffEdits sorts inline edits by their column position, ensuring that
// deletions are processed before additions at the same column.
func (r *Reporter) sortDiffEdits(edits []errors.DiffEdit) {
	slices.SortFunc(edits, func(a, b errors.DiffEdit) int {
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
	})
}

// getOriginalTokens returns the tokens from the original source that intersect with line.
func (r *Reporter) getOriginalTokens(s *diffState, line uint32) (orig []lexer.Token) {
	for i := s.lastReadTok; i < len(s.tokens); i++ {
		tok := s.tokens[i]
		if tok.Position.Line > line {
			break
		}
		if ranges.TokenEnd(tok).Line < line {
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
	return merged, virtualPos.Line
}

// printDiffLine prints a single line of a diff with its syntax-highlighted tokens
// and diff underscores.
func (r *Reporter) printDiffLine(s *diffState, line uint32, tokens []lexer.Token, first *int, added bool) {
	r.printDiffLineNumber(s, line, added, false)
	r.printSourceLine(&state{tokens: tokens}, line, first, nil)
	r.newline()
	r.printDiffUnderlines(s, tokens, line)
}

// printDiffLineNumber prints the line number and diff indicator (+/-) for a line.
func (r *Reporter) printDiffLineNumber(s *diffState, line uint32, add, fullLine bool) {
	var char rune
	var color string
	switch {
	case !fullLine:
		char = r.CharacterSet.BoxL
		color = r.ColorPalette.Box
	case add:
		// TODO: should these have backgrounds?
		char = '+'
		color = r.ColorPalette.DiffAdd
	case !add:
		char = '-'
		color = r.ColorPalette.DiffDelete
	default:
		panic("unreachable")
	}
	r.appendSpace(hintMargin)
	r.appendf(color, "%*d %c ", s.digitWidth, line, char)
}

// printDiffUnderlines adds the +/- underlines for each line.
func (r *Reporter) printDiffUnderlines(s *diffState, tokens []lexer.Token, line uint32) {
	// First check if any token on this line is a diff token
	var hasChanges bool
	for _, tok := range tokens {
		if tok.Position.Line <= line && ranges.TokenEnd(tok).Line >= line {
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
		end := ranges.TokenEnd(tok)
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

// finishLine advances lastReadTok past all tokens that fully end on or before line.
func (s *diffState) finishLine(line uint32) {
	for s.lastReadTok < len(s.tokens) {
		if ranges.TokenEnd(s.tokens[s.lastReadTok]).Line > line {
			break
		}
		s.lastReadTok++
	}
}
