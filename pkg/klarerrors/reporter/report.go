package reporter

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/util"
)

// Width is the width of the terminal. This may be replaced by [Reporter.Output]'s
// width if it is a [*os.File].
var Width int

// Error is an interface that represents an error to report.
type Error interface {
	Title() string     // The title of the error, such as "Error".
	Message() string   // The error message.
	ErrorCode() string // The error code, displayed after the message.
	IsWarning() bool   // Whether the error is a warning.

	FilePath() string       // The full path of the file where the error occurred.
	Location() ranges.Range // The start and end positions of the error in the file.
	MainHighlight() string  // The text to display after the main underline.

	// Additional file ranges to display after the error.
	ErrorDetails() []klarerrs.Detail
	// Additional underline locations in the file.
	ErrorHighlights() []klarerrs.Highlight
	// Hints to display after the error. A hint may display a diff.
	ErrorHints() []klarerrs.Hint
}

// Report prints the given error.
func (r *Reporter) Report(e Error) (n int64, err error) {
	r.init()
	r.buf = &bytes.Buffer{}

	// Highlights and ranges
	msgHighlight := klarerrs.Highlight{e.Location(), e.MainHighlight()}
	highlights := append([]klarerrs.Highlight{msgHighlight}, e.ErrorHighlights()...)
	sortHighlights(highlights)
	startLine, endLine := r.getBoxRanges(
		highlights[0].Range,
		highlights[len(highlights)-1].Range,
	)

	// Highlight color
	hlc := r.ColorPalette.ErrorColor
	if e.IsWarning() {
		hlc = r.ColorPalette.WarningColor
	}

	// Digit width
	digitWidth := digitLen(endLine)

	// Now we start printing!
	// ==========

	// 0. Divider if needed
	if r.alreadyPrinted {
		// TODO: should we add blank lines around?
		r.printDivider()
	}

	// 1. Message
	r.printMessage(e, hlc)
	r.blankLine()

	// 2. Header (-- file.klar:1:2)
	r.printHeader(e.FilePath(), e.Location(), 0, digitWidth, r.ColorPalette.Box)

	// 3. Box (file content and highlights)
	r.printBox(
		e.FilePath(), startLine, endLine,
		highlights, &msgHighlight, hlc,
		boxOptions{digitWidth: digitWidth, margin: r.Margin, color: r.ColorPalette.Box},
	)

	// 4. Details
	for _, det := range e.ErrorDetails() {
		r.newline()
		r.printDetail(det, e.FilePath())
	}

	// 5. Extended message
	// TODO: not implemented in [*Error] yet

	// 6. Hints
	for _, hint := range e.ErrorHints() {
		r.newline()
		r.printHint(hint, e.FilePath())
	}

	r.alreadyPrinted = true
	return r.buf.WriteTo(r.Output)
}

func (r *Reporter) getBoxRanges(r1, r2 ranges.Range) (startLine, endLine uint32) {
	startLine = uint32(max(1, int(r1.Start.Line)-r.MaxLines+1))
	endLine = r2.End.Line
	// If the ranges are far apart, render less lines before the first
	// range to stay closer to MaxLines.
	if 1 < startLine && startLine < r1.Start.Line &&
		endLine-startLine > uint32(r.MaxLines) {
		startLine += 1
	}
	return
}

// printMessage prints the error message and error code.
func (r *Reporter) printMessage(e Error, hlc string) {
	var b strings.Builder
	msgParts := strings.SplitAfterN(e.Message(), ": ", 2)
	if r.UseColor {
		b.WriteString(ansi.Color(hlc, e.Title()))
		b.WriteString(ansi.Color(ansi.CodeBoldDim, ": "))
		b.WriteString(ansi.Color(ansi.CodeBold, msgParts[0]))
	} else {
		fmt.Fprintf(&b, "%s: %s", e.Title(), msgParts[0])
	}
	if len(msgParts) > 1 {
		b.WriteString(msgParts[1])
	}
	if e.ErrorCode() != "" {
		code := " (" + e.ErrorCode() + ")"
		if r.UseColor {
			b.WriteString(ansi.Dim(code))
		} else {
			b.WriteString(code)
		}
	}
	util.Wrap(b.String(), r.buf, Width, 0, 2)
}

// printHeader prints the file name and position in the header.
func (r *Reporter) printHeader(file string, rang ranges.Range,
	margin, digitWidth int, boxColor string,
) {
	r.appendSpace(margin + digitWidth + 1)
	r.appendRune(r.CharacterSet.BoxTL, boxColor)
	r.appendRune(r.CharacterSet.BoxT, boxColor)
	r.appendSpace(1)
	rel := r.getFile(file).shortPath
	if rel == "" {
		rel = file
	}
	var dim string
	if r.UseColor {
		dim = ansi.CodeDim
	}
	r.appendString(rel, r.ColorPalette.FileName)
	if pos := rang.Start; pos.Line > 0 {
		r.appendRune(':', dim)
		r.appendNumber(pos.Line, r.ColorPalette.FilePos)
		r.appendRune(':', dim)
		r.appendNumber(pos.Col, r.ColorPalette.FilePos)
	}
	r.newline()
}

// printDetail prints a detail message and the corresponding code snippet.
func (r *Reporter) printDetail(det klarerrs.Detail, errFile string) {
	const detailMargin = 2
	// Title
	r.appendSpace(detailMargin)
	r.appendf(r.ColorPalette.DetailColor, "%c %s:", r.CharacterSet.DetailIcon, det.Message)
	r.blankLine()

	startLine, endLine := r.getBoxRanges(det.Range, det.Range)
	digitWidth := digitLen(endLine)

	if det.File == "" {
		det.File = errFile // Maybe we should mutate the error
	}
	// Only print a header if the file is different from the main error
	if errFile != det.File {
		r.printHeader(
			det.File, ranges.Range{},
			detailMargin, digitWidth, r.ColorPalette.DetailBox,
		)
	}
	// Box. Only the range is highlighted (no text)
	// TODO: should we repeat the detail text in the undeline? Or even show
	// an underline only?
	hl := klarerrs.Highlight{Range: det.Range}
	r.printBox(
		det.File, startLine, endLine,
		[]klarerrs.Highlight{hl}, &hl, r.ColorPalette.HintColor,
		boxOptions{
			margin:     r.Margin + detailMargin,
			digitWidth: digitWidth,
			color:      r.ColorPalette.DetailBox,
		},
	)
}

const hintMargin = 4

// printHint prints a hint message and an optional diff.
func (r *Reporter) printHint(hint klarerrs.Hint, file string) {
	r.appendString("Hint", r.ColorPalette.HintColor)
	if r.UseColor {
		r.appendString(": ", ansi.CodeDim)
	} else {
		r.buf.WriteString(": ")
	}

	util.Wrap(hint.Message, r.buf, Width, Width-len("Hint: "), 2)
	r.newline()

	if hint.Diff != nil {
		r.newline() // TODO: do we need an extra newline?
		if hint.Diff.File == "" {
			hint.Diff.File = file
		}
		r.printDiff(hint.Diff)
	}
}
