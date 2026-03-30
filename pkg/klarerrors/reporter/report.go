package reporter

import (
	"bytes"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/ranges"
)

var termWidth int

// Report prints the given error.
func (r *Reporter) Report(e errors.CompileError) (n int64, err error) {
	r.init()
	r.buf = &bytes.Buffer{}

	// Highlights and ranges
	msgHighlight := errors.Highlight{e.GetRange(), e.GetLabel()}
	highlights := append([]errors.Highlight{msgHighlight}, e.GetHighlights()...)
	sortHighlights(highlights)

	startLine := uint32(max(1, int(highlights[0].Range.Start.Line)-r.MaxLines+1))
	endLine := highlights[len(highlights)-1].Range.End.Line

	// Highlight color
	hlc := r.ColorPalette.HighlightColor
	if _, ok := e.(*errors.Warning); ok {
		hlc = r.ColorPalette.WarningHighlightColor
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
		&msgHighlight, highlights,
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
		r.printHint(e.GetFile(), hint)
	}

	r.alreadyPrinted = true
	return r.buf.WriteTo(r.Output)
}

// printMessage prints the error message and error code.
func (r *Reporter) printMessage(e errors.CompileError, hlc string) {
	msgParts := strings.SplitAfterN(e.GetMessage(), ": ", 2)
	var code string
	if e.GetCode() != 0 {
		code = ansi.Dim(" (" + e.GetCode().Format() + ")")
	}
	r.appendString(e.GetName(), ansi.CodeBold+hlc)
	r.appendString(": ", ansi.CodeBoldDim)
	r.appendString(msgParts[0], ansi.CodeBold)
	if len(msgParts) > 1 {
		r.appendString(msgParts[1], "")
	}
	r.appendString(code, "")
}

// printHeader prints the file name and position in the header.
func (r *Reporter) printHeader(file string, rang ranges.Range, digitWidth int) {
	r.appendSpace(digitWidth + 1)
	r.appendRune(r.CharacterSet.BoxTL, r.ColorPalette.BoxColor)
	r.appendRune(r.CharacterSet.BoxT, r.ColorPalette.BoxColor)
	r.appendSpace(1)
	rel := r.files[file].rel
	if rel == "" {
		rel = file
	}
	r.appendString(rel, r.ColorPalette.FileNameColor)
	if pos := rang.Start; pos.Line > 0 {
		r.appendRune(':', ansi.CodeDim)
		r.appendNumber(pos.Line, r.ColorPalette.FilePosColor)
		r.appendRune(':', ansi.CodeDim)
		r.appendNumber(pos.Col, r.ColorPalette.FilePosColor)
	}
	r.newline()
}

// printDetail prints a detail message and the corresponding code snippet.
func (r *Reporter) printDetail(det errors.Detail, sameFile bool) {
	const detailMargin = 2
	// Title
	r.appendSpace(detailMargin)
	r.appendString(det.Message, ansi.CodeBold) // Should we use bold? Yellow?
	r.newline()

	startLine := max(1, det.Range.Start.Line-uint32(r.MaxLines)+1)
	endLine := det.Range.End.Line
	digitWidth := digitLen(det.Range.End.Line)

	// Only print a header if the file is different from the main error
	if !sameFile {
		r.printHeader(det.File, ranges.Range{}, digitWidth)
	}
	// Box. Only the range is highlighted (no text)
	hl := det.Highlight
	hl.Message = ""
	r.printBox(det.File,
		&hl, []errors.Highlight{hl}, startLine, endLine,
		digitWidth, detailMargin, r.ColorPalette.HintColor,
	)
}
