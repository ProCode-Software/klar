package reporter

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// printHint prints a hint message and an optional diff.
func (r *Reporter) printHint(file string, hint errors.Hint) {
	r.appendString("Hint", r.ColorPalette.HintColor)
	r.appendString(": ", ansi.CodeDim)

	cli.Wrap(hint.Message, r.buf, termWidth, termWidth-len("Hint: "), 2)
	r.newline()

	if hint.Diff != nil {
		// r.newline()
		r.printDiff(file, hint.Diff)
	}
}

// diffLine groups diff operations that apply to a single line.
type diffLine struct {
	lineNum    uint32
	fullAdd    *[]lexer.Token
	fullRemove *ranges.Range
	insertions []*[]lexer.Token
	removals   []ranges.Range
}

// printDiff formats and prints all line changes and highlights described in the diff.
func (r *Reporter) printDiff(file string, diff *errors.Diff) {
	lines, orderedLines := r.groupDiffLines(diff)

	var maxLine uint32
	for _, l := range orderedLines {
		if l > maxLine {
			maxLine = l
		}
	}
	digitWidth := max(digitLen(maxLine), 1)

	for _, n := range orderedLines {
		dl := lines[n]

		if dl.fullRemove != nil {
			r.printFullRemove(file, dl, digitWidth)
		}

		if dl.fullAdd != nil {
			r.printFullAdd(dl, digitWidth)
		}

		if len(dl.insertions) > 0 || len(dl.removals) > 0 {
			r.printInlineDiff(file, dl, digitWidth)
		}
	}
}

// groupDiffLines organizes disjoint diff ranges by line number so they can be accurately displayed together.
func (r *Reporter) groupDiffLines(diff *errors.Diff) (map[uint32]*diffLine, []uint32) {
	lines := make(map[uint32]*diffLine)
	var orderedLines []uint32

	getLine := func(n uint32) *diffLine {
		if l, ok := lines[n]; ok {
			return l
		}
		l := &diffLine{lineNum: n}
		lines[n] = l
		orderedLines = append(orderedLines, n)
		return l
	}

	for _, dr := range diff.Ranges {
		if dr.Line {
			if dr.Operation {
				// Added a brand new line. Store its tokens in fullAdd.
				if dr.Added != nil && len(*dr.Added) > 0 {
					l := getLine((*dr.Added)[0].Position.Line)
					l.fullAdd = dr.Added
				}
			} else {
				// Removed a line entirely.
				l := getLine(dr.Range.Start.Line)
				l.fullRemove = &dr.Range
			}
		} else {
			if dr.Operation {
				// Inline insertions on an existing line.
				if dr.Added != nil && len(*dr.Added) > 0 {
					l := getLine((*dr.Added)[0].Position.Line)
					l.insertions = append(l.insertions, dr.Added)
				}
			} else {
				// Inline removals on an existing line.
				l := getLine(dr.Range.Start.Line)
				l.removals = append(l.removals, dr.Range)
			}
		}
	}

	// Sort the registered lines so the diff streams linearly from top to bottom.
	slices.Sort(orderedLines)
	return lines, orderedLines
}

// printFullRemove highlights and renders a completely removed line using the '-' indicator.
func (r *Reporter) printFullRemove(file string, dl *diffLine, digitWidth int) {
	r.appendString(fmt.Sprintf("%*d", digitWidth, dl.lineNum), r.ColorPalette.BoxColor)
	r.appendSpace(1)
	r.appendRune('-', r.ColorPalette.DiffDeleteColor)
	r.appendSpace(1)

	fileData := r.checkForFile(file)
	var lastCol uint32 = 1
	startIdx := fileData.getTokenIndexForLine(dl.lineNum)
	for i := startIdx; i < len(fileData.tokens); i++ {
		tok := fileData.tokens[i]
		if tok.Position.Line <= dl.lineNum && ranges.FromToken(tok).End.Line >= dl.lineNum && tok.Source != "\n" {
			startCol := tok.Position.Col
			if dl.lineNum > tok.Position.Line {
				startCol = 1 // Clamped to beginning inside strings
			}

			text := r.extractLineFromToken(tok, dl.lineNum)
			if text != "" {
				padding := int(startCol) - int(lastCol)
				if padding > 0 {
					r.appendSpace(padding)
				}
				r.buf.WriteString(r.colorizeText(fileData.tokens, i, text))
				lastCol = startCol + uint32(len([]rune(text)))
			}
		} else if tok.Position.Line > dl.lineNum {
			break
		}
	}
	r.newline()
}

// printFullAdd highlights and renders a completely new line using the '+' indicator.
func (r *Reporter) printFullAdd(dl *diffLine, digitWidth int) {
	r.appendString(fmt.Sprintf("%*d", digitWidth, dl.lineNum), r.ColorPalette.BoxColor)
	r.appendSpace(1)
	r.appendRune('+', r.ColorPalette.DiffAddColor)
	r.appendSpace(1)

	var lastCol uint32 = 1
	for i, tok := range *dl.fullAdd {
		tokRange := ranges.FromToken(tok)
		if tokRange.Start.Col > lastCol {
			r.appendSpace(int(tokRange.Start.Col - lastCol))
		}
		r.buf.WriteString(r.colorize(*dl.fullAdd, i))
		lastCol = tokRange.End.Col
	}
	r.newline()
}

// renderBlock holds text configured for console printing alongside positional metadata.
type renderBlock struct {
	coloredText string
	col         uint32
	endCol      uint32
}

// diffUnderline dictates what character gets printed to highlight added/removed ranges
// and the span length underneath a code file line.
type diffUnderline struct {
	col   uint32
	width uint32
	char  rune
	color string
}

// printInlineDiff merges line original tokens with line additions/removals
// and constructs overlapping sequences showing the differences visually inline.
func (r *Reporter) printInlineDiff(file string, dl *diffLine, digitWidth int) {
	slices.SortFunc(dl.insertions, func(a, b *[]lexer.Token) int {
		return cmp.Compare((*a)[0].Position.Col, (*b)[0].Position.Col)
	})

	origBlocks := r.getOriginalBlocks(file, dl.lineNum)
	blocks, underlines := r.buildInlineDiffBlocks(dl, origBlocks)

	r.renderDiffBlocks(dl.lineNum, digitWidth, blocks)
	r.renderUnderlines(digitWidth, underlines)
}

// getOriginalBlocks translates the original file token sequence of the specified line into sequential render blocks.
func (r *Reporter) getOriginalBlocks(file string, lineNum uint32) []renderBlock {
	var origBlocks []renderBlock
	fileData := r.checkForFile(file)
	startIdx := fileData.getTokenIndexForLine(lineNum)
	for i := startIdx; i < len(fileData.tokens); i++ {
		tok := fileData.tokens[i]
		if tok.Position.Line <= lineNum && ranges.FromToken(tok).End.Line >= lineNum && tok.Source != "\n" {
			startCol := tok.Position.Col
			if lineNum > tok.Position.Line {
				startCol = 1
			}

			text := r.extractLineFromToken(tok, lineNum)
			if text != "" {
				origBlocks = append(origBlocks, renderBlock{
					coloredText: r.colorizeText(fileData.tokens, i, text),
					col:         startCol,
					endCol:      startCol + uint32(len([]rune(text))),
				})
			}
		} else if tok.Position.Line > lineNum {
			break
		}
	}
	return origBlocks
}

// buildInlineDiffBlocks sequences the insertions into standard tokens pushing unaffected characters to the right
// and logs underlines where added inputs or removed intervals belong.
func (r *Reporter) buildInlineDiffBlocks(dl *diffLine, origBlocks []renderBlock) ([]renderBlock, []diffUnderline) {
	var blocks []renderBlock
	var underlines []diffUnderline
	var shift uint32 // Tracks how many characters standard tokens must be shifted rightward to accommodate new inserts

	insIdx := 0
	// Loop until all original blocks and active insertions are processed
	for len(origBlocks) > 0 || insIdx < len(dl.insertions) {
		// If we still have insertions, and the next insertion column comes before the next original block (or origBlocks is empty)
		if insIdx < len(dl.insertions) && (len(origBlocks) == 0 || (*dl.insertions[insIdx])[0].Position.Col <= origBlocks[0].col) {
			ins := dl.insertions[insIdx]
			insIdx++

			startCol := (*ins)[0].Position.Col
			endCol := ranges.FromToken((*ins)[len(*ins)-1]).End.Col
			width := endCol - startCol

			// Map every newly inserted token into a renderBlock, shifted right
			for i, tok := range *ins {
				tokCol := tok.Position.Col + shift
				tEC := ranges.FromToken(tok).End.Col + shift
				blocks = append(blocks, renderBlock{
					coloredText: r.colorize(*ins, i),
					col:         tokCol,
					endCol:      tEC,
				})
			}

			// Add a green '+' underline below the newly inserted text
			underlines = append(underlines, diffUnderline{
				col:   startCol + shift,
				width: width,
				char:  '+',
				color: r.ColorPalette.DiffAddColor,
			})

			shift += width
		} else {
			// Otherwise process the original text line block, just pushed to the right by whatever the current shift is
			blk := origBlocks[0]
			origBlocks = origBlocks[1:]

			blk.col += shift
			blk.endCol += shift
			blocks = append(blocks, blk)
		}
	}

	// Calculate and draw red '-' underlines for the specific columns that were removed
	for _, rem := range dl.removals {
		var s uint32 // Accumulative shift size specifically representing all insertions placed BEFORE this removal
		for _, ins := range dl.insertions {
			if (*ins)[0].Position.Col <= rem.Start.Col {
				start := (*ins)[0].Position.Col
				end := ranges.FromToken((*ins)[len(*ins)-1]).End.Col
				s += end - start
			}
		}

		underlines = append(underlines, diffUnderline{
			col:   rem.Start.Col + s,
			width: rem.End.Col - rem.Start.Col,
			char:  '-',
			color: r.ColorPalette.DiffDeleteColor,
		})
	}

	return blocks, underlines
}

// renderDiffBlocks commits the merged blocks layout text buffer accurately spacing standard tokens and shifts.
func (r *Reporter) renderDiffBlocks(lineNum uint32, digitWidth int, blocks []renderBlock) {
	// Setup left padding context and box border margin.
	r.appendString(fmt.Sprintf("%*d", digitWidth, lineNum), r.ColorPalette.BoxColor)
	r.appendSpace(1)
	r.appendRune(r.CharacterSet.BoxL, r.ColorPalette.BoxColor)
	r.appendSpace(1)

	var lastCol uint32 = 1
	for _, blk := range blocks {
		// Fill whitespace between tokens if the column delta leaves empty spacing.
		if blk.col > lastCol {
			r.appendSpace(int(blk.col - lastCol))
		}
		// Write ANSI-styled block string sequentially.
		r.buf.WriteString(blk.coloredText)
		lastCol = blk.endCol
	}
	r.newline()
}

// renderUnderlines stacks trailing highlights immediately under rendered file text.
func (r *Reporter) renderUnderlines(digitWidth int, underlines []diffUnderline) {
	if len(underlines) == 0 {
		return
	}

	slices.SortFunc(underlines, func(a, b diffUnderline) int {
		return cmp.Compare(a.col, b.col)
	})

	r.appendString(fmt.Sprintf("%*s", digitWidth, ""), "")
	r.appendSpace(1)
	r.appendRune(r.CharacterSet.HighlightLine, r.ColorPalette.BoxColor)
	r.appendSpace(1)

	var lastUCol uint32 = 1
	for _, u := range underlines {
		if u.width > 0 {
			if u.col > lastUCol {
				r.appendSpace(int(u.col - lastUCol))
			}
			r.appendString(strings.Repeat(string(u.char), int(u.width)), u.color)
			lastUCol = u.col + u.width
		}
	}
	r.newline()
}
