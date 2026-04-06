package reporter

import (
	"github.com/ProCode-Software/klar/internal/char"
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
		file = r.getFile(fileName)
		s    = &state{
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
			s.hlColors[hl] = mainHlColor // Primary highlight
		} else {
			s.hlColors[hl] = colors[colorI%len(colors)]
			colorI++
		}
	}

	firstTokOnLine := file.getTokenIndexForLine(startLine)
	for line := startLine; line <= endLine; line++ {
		groups := groupHighlights(highlights, line)
		if len(groups.newSingleLine) == 0 && len(groups.newMultiline) == 0 {
			// No new highlights on this line: check if we can collapse lines
			if n := r.tryCollapseLines(s, line, endLine, &firstTokOnLine, groups.existing); n > 0 {
				line += n
				firstTokOnLine = file.getTokenIndexForLine(line + 1)
				continue
			}
		}
		// Print the line number
		r.printLineNumber(s, line)

		// Print a line of code
		lastCol := r.printSourceLine(s, line, &firstTokOnLine, groups.existing)
		r.newline()
		// TODO: change these params
		r.printHighlights(s, line, lastCol, groups)
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
// possible, tryCollapseLines prints the collapsed lines and returns the
// distance between line and the next line with highlights, otherwise it
// returns 0. tryCollapseLines only collapses lines if there are no new or
// ending highlights on the next line, assuming there are none on line.
func (r *Reporter) tryCollapseLines(s *state, line,
	endLine uint32, firstTokOnLine *int, activeHls []errors.Highlight,
) (n uint32) {
	// Find the next line with highlights starting or ending.
	for line := line + 1; line <= endLine; line++ {
		groups := groupHighlights(s.highlights, line)
		if len(groups.newSingleLine) > 0 || len(groups.newMultiline) > 0 ||
			r.highlightsEndOnLine(line, groups.existing) {
			break
		}
		n++
	}
	if n <= 1 { // If that line is the next one, just print it.
		return 0
	}
	// Print 1 line of context before collapsing
	r.printLineNumber(s, line)
	r.printSourceLine(s, line, firstTokOnLine, activeHls)
	r.newline()

	// Print the collapsed line number, highlight pipes, and ellipsis
	r.printCollapsedLineNumber(s, activeHls)
	return n
}

// highlightsEndOnLine reports whether any highlight ends on the given line.
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

// printCollapsedLineNumber prints the collapsed line number, the pipes of
// active multiline highlights, and ellipsis.
func (r *Reporter) printCollapsedLineNumber(s *state, highlights []errors.Highlight) {
	// Character for the border of the box.
	borderChar := r.CharacterSet.SkipLineL // Horizontal line on right side
	if len(highlights) > 0 {
		// No horizontal line
		borderChar = r.CharacterSet.BoxL
	}

	// Print the dashed line number
	r.appendSpace(s.margin)
	r.appendf(r.ColorPalette.Box, "%s %c ",
		// Dashes in line num position
		char.RepeatRune(r.CharacterSet.SkipLine, s.digitWidth),
		borderChar, // Box border
	)

	// Print the pipes of active multiline highlights
	if len(highlights) > 0 {
		// Exclude the last pipe. We want another style for that last one.
		r.printHighlightPipes(s, highlights[:len(highlights)-1])
		// Last pipe + ellipsis
		r.appendf(
			s.hlColors[highlights[len(highlights)-1]], "%c %[2]c%[2]c%[2]c\n",
			r.CharacterSet.HighlightMultilineCollapsed,
			r.CharacterSet.CollapsedEllipsis,
		)
		return
	}
	// No active multiline highlights: ellipsis only in the box color
	r.appendf(r.ColorPalette.Box, "%[1]c%[1]c%[1]c\n", r.CharacterSet.CollapsedEllipsis)
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
	singleLine := groups.newSingleLine
	pipeLen := len(groups.existing) * 2
	// Print the underlines
	if len(singleLine) > 0 {
		printLineStart()
		singleLine = r.printUnderlines(s, pipeLen, singleLine, func(rem []errors.Highlight) {
			// If underline line overflows, print the stems of the other highlights
			// on the next line.
			if len(rem) > 0 {
				r.printArrows(s, rem, printLineStart, pipeLen, true)
			} else {
				printLineStart()
			}
		})
		r.newline()
	}
	// The arrows and the messages
	r.printArrows(s, singleLine, printLineStart, pipeLen, false)

	// 2. Ending multiline highlights
	// ===================
	r.printEndingMultilineLabels(s, groups.existing, line)

	// 3. New multiline highlights
	// ===================
	r.printNewMultilineUnderlines(s, groups.newMultiline, lastCol, printLineStart)
}
