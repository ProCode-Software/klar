package reporter

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// Width is the width of the terminal. This may be replaced by [Reporter.Output]'s
// width if it is a [*os.File].
var Width int

// Report prints the given error.
func (r *Reporter) Report(e errors.CompileError) (n int64, err error) {
	r.init()
	r.buf = &bytes.Buffer{}

	// Highlights and ranges
	msgHighlight := errors.Highlight{e.GetRange(), e.GetLabel()}
	highlights := append([]errors.Highlight{msgHighlight}, e.GetHighlights()...)
	sortHighlights(highlights)
	startLine, endLine := r.getBoxRanges(highlights[0].Range,
		highlights[len(highlights)-1].Range,
	)

	// Highlight color
	hlc := r.ColorPalette.ErrorColor
	if _, ok := e.(*errors.Warning); ok {
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
	r.printHeader(e.GetFile(), e.GetRange(), digitWidth)

	// 3. Box (file content and highlights)
	r.printBox(e.GetFile(),
		highlights, &msgHighlight,
		startLine, endLine, digitWidth, 0, hlc,
	)

	// 4. Details
	for _, det := range e.GetDetails() {
		r.newline()
		r.printDetail(det, e.GetFile() == det.File)
	}

	// 5. Extended message
	// TODO: not implemented in [CompileError] yet

	// 6. Hints
	for _, hint := range e.GetHints() {
		r.newline()
		r.printHint(hint, e.GetFile())
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
func (r *Reporter) printMessage(e errors.CompileError, hlc string) {
	var b strings.Builder
	msgParts := strings.SplitAfterN(e.GetMessage(), ": ", 2)
	if r.UseColor {
		b.WriteString(ansi.Color(hlc, e.GetName()))
		b.WriteString(ansi.Color(ansi.CodeBoldDim, ": "))
		b.WriteString(ansi.Color(ansi.CodeBold, msgParts[0]))
	} else {
		fmt.Fprintf(&b, "%s: %s", e.GetName(), msgParts[0])
	}
	if len(msgParts) > 1 {
		b.WriteString(msgParts[1])
	}
	if e.GetCode() != 0 {
		code := " (" + e.GetCode().Format() + ")"
		if r.UseColor {
			b.WriteString(ansi.Dim(code))
		} else {
			b.WriteString(code)
		}
	}
	cli.Wrap(b.String(), r.buf, Width, 0, 2)
}

// printHeader prints the file name and position in the header.
func (r *Reporter) printHeader(file string, rang ranges.Range, digitWidth int) {
	r.appendSpace(digitWidth + 1)
	r.appendRune(r.CharacterSet.BoxTL, r.ColorPalette.Box)
	r.appendRune(r.CharacterSet.BoxT, r.ColorPalette.Box)
	r.appendSpace(1)
	rel := r.getFile(file).rel
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
func (r *Reporter) printDetail(det errors.Detail, sameFile bool) {
	const detailMargin = 2
	// Title
	r.appendSpace(detailMargin)
	var textColor string
	if r.UseColor {
		textColor = ansi.CodeBold
	}
	r.appendString(det.Message, textColor) // TODO: Should we use bold? Yellow?
	r.newline()

	startLine, endLine := r.getBoxRanges(det.Range, det.Range)
	digitWidth := digitLen(endLine)

	// Only print a header if the file is different from the main error
	if !sameFile {
		r.printHeader(det.File, ranges.Range{}, digitWidth)
	}
	// Box. Only the range is highlighted (no text)
	hl := errors.Highlight{Range: det.Range}
	r.printBox(det.File,
		[]errors.Highlight{hl}, &hl, startLine, endLine,
		digitWidth, detailMargin, r.ColorPalette.HintColor,
	)
}

const hintMargin = 4

// printHint prints a hint message and an optional diff.
func (r *Reporter) printHint(hint errors.Hint, file string) {
	r.appendString("Hint", r.ColorPalette.HintColor)
	if r.UseColor {
		r.appendString(": ", ansi.CodeDim)
	} else {
		r.buf.WriteString(": ")
	}

	cli.Wrap(hint.Message, r.buf, Width, Width-len("Hint: "), 2)
	r.newline()

	if hint.Diff != nil {
		r.newline() // TODO: do we need an extra newline?
		if hint.Diff.File == "" {
			hint.Diff.File = file
		}
		r.printDiff(hint.Diff)
	}
}
