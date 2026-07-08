package reporter

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/klarerrs"
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
	addedLine   *klarerrs.DiffEdit
	deletedLine *klarerrs.DeletedLine
	ranges      []klarerrs.DiffEdit
}

const (
	deletedToken lexer.TokenType = -69
	addedToken                   = deletedToken + 1
)

// printDiff formats and prints all line changes and highlights described in the diff.
func (r *Reporter) printDiff(diff *klarerrs.Diff) {
	var (
		lines, end = r.groupDiffLines(diff)
		digitWidth = digitLen(end)
		file       = r.getFile(diff.File)
		state      = &diffState{
			digitWidth: digitWidth,
			tokens:     file.tokens,
		}
	)
	var lastLine uint32
	for _, lineNum := range slices.Sorted(maps.Keys(lines)) {
		if lastLine > 0 && lineNum <= lastLine {
			continue
		}
		if lastLine > 0 && lineNum > lastLine+1 {
			r.printSkippedDiffLineNumber(state)
		}
		dl := lines[lineNum]
		if dl.deletedLine != nil || dl.addedLine != nil {
			// Whole-line addition or removal. Removals, then additions
			// are printed in that order.
			if dl.deletedLine != nil {
				lastLine = r.printFullLineRemove(state, dl)
			}
			if dl.addedLine != nil {
				lastLine = r.printFullLineAdd(state, dl)
			}
		} else {
			// Inline diffs are where diff ranges, both additions and deletions, are
			// displayed alongside the original source.
			lastLine = r.printInlineDiffLine(state, dl)
		}
	}
}

func (r *Reporter) groupDiffLines(diff *klarerrs.Diff) (
	lines map[uint32]*diffLine, end uint32,
) {
	lines = make(map[uint32]*diffLine)
	for _, edit := range diff.Edits {
		start := edit.Start().Line
		endLine := edit.EndLine()
		for line := start; line <= endLine; line++ {
			if _, ok := lines[line]; !ok {
				lines[line] = &diffLine{line: line}
			}
			dl := lines[line]
			end = max(end, endLine) // Keep track of maximum line seen
			if !edit.FullLine() {
				dl.ranges = append(dl.ranges, edit)
				continue
			}
			switch edit := edit.(type) {
			case klarerrs.DeletedLine:
				dl.deletedLine = &edit
			case klarerrs.AddedTokens, klarerrs.AddedString:
				dl.addedLine = &edit
			}
		}
	}
	return
}

// printFullLineRemove prints a block of source code representing a full-line deletion.
func (r *Reporter) printFullLineRemove(s *diffState, dl *diffLine) (lastLine uint32) {
	edit := *dl.deletedLine
	// Get intersecting tokens
	firstTokI, maxTokI := -1, -1
	for i, tok := range s.tokens[s.lastReadTok:] {
		if ranges.FromToken(tok).LineIn(edit.Line) {
			if firstTokI < 0 {
				firstTokI = s.lastReadTok + i
			}
		} else if firstTokI >= 0 {
			maxTokI = s.lastReadTok + i
			break
		}
	}
	if maxTokI < 0 && firstTokI >= 0 {
		maxTokI = len(s.tokens)
	}
	if firstTokI < 0 {
		// The diff's token range doesn't include the line, so we can't print it
		panic(fmt.Sprintf("no tokens found for line %d", dl.line))
	}
	srcTokens := setTokenKind(s.tokens[firstTokI:maxTokI], deletedToken)
	srcState := &state{tokens: srcTokens}
	r.printDiffLineNumber(s, dl.line, false, true)
	r.printSourceLine(srcState, dl.line, new(int), nil)
	s.lastReadTok = maxTokI
	r.newline()
	return dl.line
}

// printFullLineAdd prints a block of source code or strings representing a
// full-line addition.
func (r *Reporter) printFullLineAdd(s *diffState, dl *diffLine) (lastLine uint32) {
	hlColor := r.ColorPalette.DiffAdd + r.ColorPalette.DiffAddBackground
	switch edit := (*dl.addedLine).(type) {
	case klarerrs.AddedString:
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
	case klarerrs.AddedTokens:
		var (
			srcTokens = setTokenKind(edit.Tokens, addedToken)
			srcState  = &state{tokens: srcTokens}
			end       = edit.EndLine()
		)
		for line := edit.Start().Line; line <= end; line++ {
			r.printDiffLineNumber(s, line, true, true)
			r.printSourceLine(srcState, line, new(int), nil)
			r.newline()
		}
		return end
	}
	return dl.line
}

func setTokenKind(tokens []lexer.Token, kind lexer.TokenType) []lexer.Token {
	tokens = slices.Clone(tokens)
	for i := range tokens {
		tokens[i].Kind = kind
	}
	return tokens
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
		panic(fmt.Sprintf("unhandled diff line type: add = %v, fullLine = %v", add, fullLine))
	}
	r.appendSpace(hintMargin)
	r.appendf(color, "%*d %c ", s.digitWidth, line, char)
}

func (r *Reporter) printSkippedDiffLineNumber(s *diffState) {
	r.appendSpace(hintMargin)
	r.appendf(
		r.ColorPalette.Box, "%*c %c %[4]c%[4]c%[4]c\n",
		s.digitWidth, r.CharacterSet.SkipLine,
		r.CharacterSet.SkipLineL,
		r.CharacterSet.CollapsedEllipsis,
	)
}
