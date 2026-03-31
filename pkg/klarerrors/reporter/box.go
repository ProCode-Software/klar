package reporter

import (
	"github.com/ProCode-Software/klar/internal/char"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
)

type state struct {
	tokens     []lexer.Token
	digitWidth int
	margin     int
	highlights []errors.Highlight
	hlColors   map[errors.Highlight]string
}

// groupHighlights are the groups of highlights created by [groupHighlights].
type groupedHighlights struct {
	// Existing multiline highlights, and single and multi-line highlights
	// that start on a given line.
	existing, newSingleLine, newMultiline []errors.Highlight
}

// printBox prints the syntax-highlighted lines of the file
// and labels from the error. mainHl will be colored with mainHlColor.
// All other highlights will be colored with colors from r's ColorPalette.
// All lines, including the first line, contain the margin.
func (r *Reporter) printBox(fileName string,
	highlights []errors.Highlight, mainHl *errors.Highlight,
	startLine, endLine uint32, digitWidth, margin int, mainHlColor string,
) {
	var (
		file  = r.checkForFile(fileName)
		state = &state{
			tokens:     file.tokens,
			digitWidth: digitWidth,
			margin:     margin,
			highlights: highlights,
			hlColors:   make(map[errors.Highlight]string, len(highlights)),
		}
		colors = []string{
			// Repeated if there are more than 3 highlights
			r.ColorPalette.Highlight1, r.ColorPalette.Highlight2, r.ColorPalette.Highlight3,
		}
	)
	// Give each highlight a color
	var colorI int
	for _, hl := range highlights {
		if hl == *mainHl {
			state.hlColors[hl] = mainHlColor + ansi.CodeBold // Primary highlight
		} else {
			state.hlColors[hl] = colors[colorI%len(colors)]
			colorI++
		}
	}

	firstTokOnLine := file.getTokenIndexForLine(startLine)
	for line := startLine; line <= endLine; line++ {
		groups := groupHighlights(highlights, line)
		if len(groups.existing) > 0 &&
			len(groups.newSingleLine) == 0 && len(groups.newMultiline) == 0 {
			// No new highlights on this line: check if we can collapse lines
			if nextLine := r.tryCollapseLines(state, line, endLine, groups.existing); nextLine > line {
				line = nextLine - 1 // Because of line++ in the loop
				continue
			}
		}
		// Print the line number
		r.printLineNumber(state, line)

		// Print a line of code
		lastCol := r.printSourceLine(state, line, &firstTokOnLine, groups.existing)
		r.newline()
		// TODO: change these params
		r.printHighlights(state, line, lastCol, groups)
	}
}

// groupHighlights groups the highlights on line into active multiline highlights,
// and single and multiline highlights that start on line.
func groupHighlights(highlights []errors.Highlight, line uint32) *groupedHighlights {
	var existing, newSingleLine, newMultiline []errors.Highlight
	for _, hl := range highlights {
		switch {
		case !hl.Range.LineIn(line):
			continue // Not within this line
		case hl.Range.IsSingleLine():
			newSingleLine = append(newSingleLine, hl)
		case hl.Range.Start.Line == line:
			newMultiline = append(newMultiline, hl)
		default:
			existing = append(existing, hl)
		}
	}
	return &groupedHighlights{existing, newSingleLine, newMultiline}
}

// tryCollapseLines checks if lines starting from line can be collapsed. If
// possible, tryCollapseLines prints the collapsed lines and returns the next,
// line that is not collapsed, otherwise it returns the current line.
// tryCollapseLines only collapses lines if there are no new or ending
// highlights on the next line, assuming there are none on line.
func (r *Reporter) tryCollapseLines(s *state, line, endLine uint32, activeHls []errors.Highlight) (nextLine uint32) {
	var dist uint32 // Distance between 'line' and next line with highlights
	for line := line + 1; line <= endLine; line++ {
		groups := groupHighlights(s.highlights, line)
		if len(groups.newSingleLine) > 0 || len(groups.newMultiline) > 0 ||
			!r.highlightsEndOnLine(line, activeHls) {
			break
		}
		dist++
	}
	if dist <= 1 {
		return line
	}
	// Print the dashed line number
	r.appendSpace(s.margin)
	r.appendf(r.ColorPalette.Box, "%s%c ",
		char.RepeatRune(r.CharacterSet.SkipLine, s.digitWidth+1), // Dashes in line num pos
		r.CharacterSet.SkipLineL,                                 // Box divider
	)
	// Print pipes, except the last one. We want another style for that last one.
	r.printHighlightPipes(s, activeHls[:len(activeHls)-1])
	// Last pipe + ellipsis
	r.appendf(
		s.hlColors[activeHls[len(activeHls)-1]],
		"%c %[2]c%[2]c%[2]c\n", r.CharacterSet.HighlightMultilineSkip,
		r.CharacterSet.HighlightMultilineSkipEllipsis,
	)
	return line + dist
}

func (r *Reporter) highlightsEndOnLine(line uint32, highlights []errors.Highlight) bool {
	for _, hl := range highlights {
		if hl.Range.End.Line == line {
			return true
		}
	}
	return false
}

func (r *Reporter) printLineNumber(s *state, line uint32) {
	r.appendSpace(s.margin)
	r.appendf(r.ColorPalette.Box, "%*d %c ", s.digitWidth, line, r.CharacterSet.BoxL)
}

func (r *Reporter) printEmptyLineNumber(s *state) {
	r.appendSpace(s.margin + s.digitWidth + 1)
	r.appendf(r.ColorPalette.Box, "%c ", r.CharacterSet.HighlightLine)
}

func (r *Reporter) printHighlightPipes(s *state, activeHls []errors.Highlight) {
	for _, hl := range activeHls {
		r.appendf(s.hlColors[hl], "%c ", r.CharacterSet.BoxL)
	}
}

// printHighlights prints a line with the highlights that start or end on line.
func (r *Reporter) printHighlights(s *state, line, lastCol uint32,
	groups *groupedHighlights,
) {
	printLineStart := func() {
		r.printEmptyLineNumber(s)
		r.printHighlightPipes(s, groups.existing)
	}

	// 1. Single-line highlights
	// ==========
	remHls := groups.newSingleLine
	pipeLen := len(groups.existing) * 2
	// Print the underlines
	if len(remHls) > 0 {
		printLineStart()
		r.printUnderlines(s, pipeLen, remHls, func() {
			// If underline line overflows, print the stems of the other highlights
			// on the next line.
			r.printArrows(s, remHls[:len(remHls)-1], printLineStart, pipeLen, true)
		})
		r.newline()
		// Cut off the last highlight, which has been labelled
		remHls = remHls[:len(remHls)-1]
	}
	// The arrows and the messages
	r.printArrows(s, remHls, printLineStart, pipeLen, false)

	// 2. Ending multiline highlights
	// ===================
	r.printEndingMultilineLabels(s, groups.existing, line)

	// 3. New multiline highlights
	// ===================
	r.printNewMultilineUnderlines(s, groups.newMultiline, lastCol, printLineStart)
}
