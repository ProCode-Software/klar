package reporter

import (
	"slices"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/char"
	"github.com/ProCode-Software/klar/internal/klarerrs"
)

// printUnderlines prints the underlines for the single-line highlights, as well
// as the label for overflowing highlights and the rightmost highlight. pipeLen
// is the number of spaces to add in order to account for the pipes of the multiline highlights.
func (r *Reporter) printUnderlines(s *state, pipeLen int,
	highlights []klarerrs.Highlight, printLineStart func(rem []klarerrs.Highlight),
) (remHls []klarerrs.Highlight) {
	var lastCol uint32 = 1
	for i, hl := range highlights {
		rang := hl.Range
		if rang.Start.Col >= lastCol {
			r.padding(lastCol, rang.Start.Col)
		} else {
			// We need to underline on a new line
			r.newline()
			printLineStart(remHls)
			r.padding(1, rang.Start.Col)
		}

		shouldPrintLabel := i == len(highlights)-1 ||
			rang.End.Col > highlights[i+1].Range.Start.Col

		if ulLen := int(rang.LineLength()); ulLen <= 1 {
			// Use '^'
			r.appendRune(r.CharacterSet.HighlightSingle, s.hlColors[hl])
		} else {
			// Draw an underline with a stem
			stemOffset := getStemOffset(ulLen)
			stemChar := r.CharacterSet.ArrowT
			// Don't draw a stem for the rightmost highlight or if the next
			// highlight must start on a new line
			if shouldPrintLabel {
				stemChar = r.CharacterSet.HighlightMulti
			}
			r.appendf(s.hlColors[hl], "%s%c%s",
				char.RepeatRune(r.CharacterSet.HighlightMulti, stemOffset),
				stemChar,
				char.RepeatRune(r.CharacterSet.HighlightMulti, ulLen-stemOffset-1),
			)
		}

		// Print the label for the rightmost highlight or if the next
		// highlight can't fit on this line
		if shouldPrintLabel {
			// Number of spaces to add after the line number
			startCol := pipeLen + int(hl.Range.Start.Col-1)
			r.printLabel(hl.Message, s.hlColors[hl],
				startCol, int(hl.Range.LineLength()), func() { printLineStart(remHls) },
			)
		} else {
			remHls = append(remHls, hl)
		}
		lastCol = rang.End.Col
	}
	return remHls
}

// printArrows prints the arrows for the single-line highlights.
// If stemsOnly is true, it only prints the stems and not the labels.
func (r *Reporter) printArrows(s *state, remHls []klarerrs.Highlight,
	printLineStart func(), pipeLen int, stemsOnly bool,
) {
	for len(remHls) > 0 {
		printLineStart()
		var lastCol uint32 = 1
		// Print arrow lines
		for i, hl := range remHls {
			var (
				color      = s.hlColors[hl]
				rang       = hl.Range
				stemOffset = getStemOffset(int(rang.LineLength()))
			)
			r.padding(lastCol, rang.Start.Col)
			r.appendSpace(stemOffset)
			if i < len(remHls)-1 || stemsOnly {
				r.appendRune(r.CharacterSet.BoxL, color)
				lastCol = rang.End.Col
			} else {
				// Rightmost
				r.appendRune(r.CharacterSet.ArrowBL, color)
				r.appendRune(r.CharacterSet.BoxT, color)
			}
		}
		if stemsOnly {
			return
		}
		// Label and cut off the rightmost highlight
		hl := remHls[len(remHls)-1]
		startOffset := (int(hl.Range.Start.Col) - 1) + pipeLen
		r.printLabel(hl.Message, s.hlColors[hl], startOffset, -1, nil)
		r.newline()
		remHls = remHls[:len(remHls)-1]
	}
}

func (r *Reporter) printEndingMultilineLabels(s *state,
	highlights []klarerrs.Highlight, line uint32,
) {
	for i, hl := range slices.Backward(highlights) {
		if hl.Range.End.Line != line {
			continue
		}
		r.printEmptyLineNumber(s)
		if len(highlights[:i]) > 0 {
			r.printHighlightPipes(s, highlights[:i])
		}
		color := s.hlColors[hl]
		// Draw extra parts that aren't underlined to make up for other active pipes
		pipeLen := ((len(highlights[:i]) - i) * 2) - 1 // TODO: what is -1 for?
		r.appendf(color, "%c%s%s",
			r.CharacterSet.HighlightMultilineBL,
			char.RepeatRune(r.CharacterSet.BoxT, max(0, pipeLen)),
			char.RepeatRune(r.CharacterSet.HighlightMulti, int(hl.Range.End.Col)),
		)
		r.printLabel(hl.Message, color, -1 /* I don't care */, -1, nil) // TODO: care
		r.newline()
	}
}

// printNewMultilineUnderlines prints the underlines for multiline highlights
// that start on the current line. Each highlight is printed on its own line.
func (r *Reporter) printNewMultilineUnderlines(s *state, highlights []klarerrs.Highlight,
	lastCol uint32, printLineStart func(),
) {
	for i, hl := range highlights {
		var (
			color = s.hlColors[hl]
			pos   = hl.Range.Start
			// Reduce the offset (and maybe pipe length) to account for the pipe
			// lengths of previous printed pipes
			pipeLen = i * 2
			ulShift int
		)
		printLineStart()
		r.printHighlightPipes(s, highlights[:i])
		r.appendRune(r.CharacterSet.HighlightMultilineTL, color)
		// Dotted offset
		if pos.Col > 2 {
			offsetShift := int(pos.Col-2) - pipeLen
			if offsetShift < 0 {
				// Offset isn't long enough to reduce. Instead reduce the underline
				offsetShift, ulShift = 0, -offsetShift
			}
			r.appendf(color, "%s", char.RepeatRune(
				r.CharacterSet.HighlightMultilineOffset, offsetShift,
			))
		} else if pipeLen > 0 {
			ulShift = pipeLen
		}
		// Underline the contents of the first line
		r.appendf(color, "%s", char.RepeatRune(
			r.CharacterSet.HighlightMulti, max(0, int(lastCol-pos.Col-1)-ulShift),
		))
		r.newline()
	}
}

// printLabel prints a space and a label, wrapping the label if it doesn't fit in the
// terminal's width. If width > 0, printLabel may print an arrow on the next
// line with label under. offset is the number of spaces to add after the line
// number. ulWidth is the length of the underline in order to center the label.
func (r *Reporter) printLabel(label, color string,
	offset, ulWidth int, printLineStart func(),
) {
	if label == "" {
		return
	}
	labelLen := utf8.RuneCountInString(label)
	// If the label doesn't fit within the terminal's width, print it on the next line
	if ulWidth > 0 && Width > 0 && offset+labelLen > Width {
		r.newline()
		printLineStart() // Print pipes
		// Center the label
		ulCenter := offset + ulWidth/2
		textCenter := max(ulCenter-(labelLen/2), 0)
		r.appendSpace(textCenter)
	}
	r.appendSpace(1)
	r.appendString(label, color)
}

func getStemOffset(lineLen int) int {
	stemOffset := 2 // # of characters to draw before the stem
	if lineLen <= 3 {
		stemOffset = max(0, lineLen-2)
	}
	return stemOffset
}
