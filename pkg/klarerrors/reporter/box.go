package reporter

import (
	"fmt"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/char"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type state struct {
	tokens     []lexer.Token
	digitWidth int
	margin     int
	highlights []errors.Highlight
	hlColors   map[errors.Highlight]string
}

type row struct {
	num        uint32
	digitWidth int
	activeHl   *errors.Highlight
	lastCol    uint32
	hasTokens  bool
}

// printBox prints the highlighted lines of the file and labels from the error.
// All lines, including the first line, contain the margin.
func (r *Reporter) printBox(fileName string,
	mainHl *errors.Highlight, highlights []errors.Highlight,
	startLine, endLine uint32, digitWidth, margin int, hlColor string,
) {
	var (
		file     = r.checkForFile(fileName)
		toks     = file.tokens
		hlColors = make(map[errors.Highlight]string)
		hlIndex  = 0
		state    = &state{
			tokens:     toks,
			digitWidth: digitWidth,
			margin:     margin,
			highlights: highlights,
			hlColors:   hlColors,
		}
		colors = []string{
			// Repeated if there are more than 3 highlights
			r.ColorPalette.Highlight1, r.ColorPalette.Highlight2, r.ColorPalette.Highlight3,
		}
	)
	// Map highlights to colors
	for _, hl := range highlights {
		if mainHl != nil && hl == *mainHl {
			hlColors[hl] = hlColor // Primary highlight
		} else {
			hlColors[hl] = colors[hlIndex%len(colors)]
			hlIndex++
		}
	}

	currTok := file.getTokenIndexForLine(startLine)
	var lastPrintedLine uint32 = startLine - 1
	for num := startLine; num <= endLine; num++ {
		var hasTokens bool
		for i := currTok; i < len(toks) && toks[i].Position.Line <= num; i++ {
			if ranges.FromToken(toks[i]).End.Line >= num {
				hasTokens = true
				break
			}
		}

		rw := &row{
			num:        num,
			digitWidth: digitWidth,
			activeHl:   r.getActiveMultilineHighlight(state, num),
			hasTokens:  hasTokens,
		}

		if rw.num > lastPrintedLine+1 && lastPrintedLine != 0 {
			// If we skipped a line or multiple lines natively in the iteration, render a broken separator
			r.printSkipLine(state, lastPrintedLine, rw)
		}

		r.printSourceLine(state, &currTok, rw)

		if rw.hasTokens || rw.num == startLine || rw.num == endLine {
			r.newline()
		}

		r.printHighlights(state, rw)

		lastPrintedLine = rw.num
	}
}

// printSkipLine optionally prints the skipped line separator (e.g. ` - ┤ `)
// when there corresponds a visible gap between consecutively printed lines.
func (r *Reporter) printSkipLine(st *state, lastPrintedLine uint32, nextRow *row) {
	// Dashed line number
	fmt.Fprintf(r.buf, ansi.Color(r.ColorPalette.BoxColor, "%*c%s%c "),
		st.margin, ' ', // Margin
		char.RepeatRune(r.CharacterSet.SkipLine, nextRow.digitWidth+1), // Dashes in line num pos
		r.CharacterSet.SkipLineL,                                      // Box divider
	)
	// Find the highlight
	var activeHl *errors.Highlight
	for _, hl := range st.highlights {
		if hl.Range.Start.Line <= lastPrintedLine && hl.Range.End.Line >= nextRow.num {
			h := hl
			activeHl = &h
			break
		}
	}
	// Print the highlight with ellipsis
	if activeHl != nil {
		fmt.Fprintf(r.buf,
			ansi.Color(st.hlColors[*activeHl], "%c %[2]c%[2]c%[2]c"),
			r.CharacterSet.HighlightMultilineSkip,
			r.CharacterSet.HighlightMultilineSkipEllipsis,
		)
	}
	r.newline()
}

// getActiveMultilineHighlight iterates through the highlights to determine if
// a multiline highlight overlaps with the designated line, returning the
// overlapping highlight or nil.
func (r *Reporter) getActiveMultilineHighlight(st *state, line uint32) *errors.Highlight {
	for i, hl := range st.highlights {
		rng := hl.Range
		if !rng.IsSingleLine() &&
			rng.Start.Line <= line && rng.End.Line >= line {
			return &st.highlights[i]
		}
	}
	return nil
}

// extractLineFromToken safely isolates the substring of a token that belongs to a specific line.
// This is critical since a token might be a multiline string and we print out output row by row.
func (r *Reporter) extractLineFromToken(tok lexer.Token, line uint32) string {
	tokRange := ranges.FromToken(tok)
	if tokRange.IsSingleLine() {
		return tok.Source
	}

	lines := strings.Split(tok.Source, "\n")
	lineIdx := int(line - tok.Position.Line)
	if lineIdx >= 0 && lineIdx < len(lines) {
		return lines[lineIdx]
	}
	return ""
}

// printSourceLine outputs the row prefix (margin, line number, margin border) and precisely loops
// through each token mapped to this exact line for highlighted placement. It returns the final column space index evaluated.
func (r *Reporter) printSourceLine(st *state, currTok *int, rw *row) {
	boxColor := r.ColorPalette.BoxColor
	r.appendSpace(st.margin)
	numStr := fmt.Sprintf("%*d", rw.digitWidth, rw.num)
	r.appendString(numStr, boxColor)
	r.appendSpace(1)
	r.appendRune(r.CharacterSet.BoxL, boxColor)

	if rw.activeHl != nil {
		if rw.activeHl.Range.Start.Line < rw.num && rw.activeHl.Range.End.Line > rw.num {
			r.appendSpace(1)
			r.appendRune(r.CharacterSet.ArrowV, st.hlColors[*rw.activeHl])
		} else if rw.activeHl.Range.End.Line == rw.num {
			r.appendSpace(1)
			r.appendRune(r.CharacterSet.ArrowV, st.hlColors[*rw.activeHl])
		}
	}

	var lastCol uint32 = 1
	// Output each token intersecting the current line horizontally, padding any blank column distances safely.
	for *currTok < len(st.tokens) && st.tokens[*currTok].Position.Line <= rw.num {
		tok := st.tokens[*currTok]
		tokRange := ranges.FromToken(tok)

		// Token ended before this line, we safely advance and skip
		if tokRange.End.Line < rw.num || tok.Source == "\n" {
			*currTok++
			continue
		}

		// Calculate columns to pad before text, clamping safely at 0 to avoid printer.go's spacing negative panic
		startCol := tok.Position.Col
		if rw.num > tok.Position.Line {
			startCol = 1 // Token continued onto this line from the very beginning
		}

		padding := int(startCol) - int(lastCol)
		if padding > 0 {
			r.appendSpace(padding)
		}

		text := r.extractLineFromToken(tok, rw.num)
		if text != "" {
			r.buf.WriteString(r.colorizeText(st.tokens, *currTok, text))
			lastCol = startCol + uint32(len([]rune(text)))
		}

		// Keep the token stuck iterating into future rows unless it explicitly terminates on this line.
		if tokRange.End.Line <= rw.num {
			*currTok++
		} else {
			break
		}
	}
	rw.lastCol = lastCol
}

// printHighlights iterates through all highlight instances overlapping the given line to properly
// generate horizontal arrows underneath or connected sidebars bridging multiple rows based on position bounds.
func (r *Reporter) printHighlights(st *state, rw *row) {
	boxColor := r.ColorPalette.BoxColor
	var lineHls []errors.Highlight
	for _, hl := range st.highlights {
		if hl.Range.Start.Line == rw.num || hl.Range.End.Line == rw.num {
			lineHls = append(lineHls, hl)
		}
	}

	for _, hl := range lineHls {
		c := st.hlColors[hl]
		r.appendSpace(st.margin)
		r.appendString(fmt.Sprintf("%*s", rw.digitWidth, ""), boxColor)
		r.appendSpace(1)
		r.appendRune(r.CharacterSet.HighlightLine, boxColor)
		r.appendSpace(1)

		if hl.Range.IsSingleLine() {
			// Provide an uninterrupted vertical link line if there's an outer overarching multiline highlight
			if rw.activeHl != nil && rw.activeHl != &hl {
				r.appendRune(r.CharacterSet.ArrowV, st.hlColors[*rw.activeHl])
				r.appendSpace(1)
			}
			startCol := int(hl.Range.Start.Col)
			endCol := int(hl.Range.End.Col)
			if startCol < 1 {
				startCol = 1
			}
			r.appendSpace(startCol - 1)
			if startCol+1 >= endCol {
				r.appendRune(r.CharacterSet.HighlightSingle, c)
			} else {
				length := max(endCol-startCol, 1)
				r.appendString(string(char.RepeatRune(r.CharacterSet.HighlightMulti, length)), c)
			}
			if hl.Message != "" {
				r.appendSpace(1)
				r.appendString(hl.Message, c)
			}
		} else if hl.Range.Start.Line == rw.num {
			r.appendRune(r.CharacterSet.HighlightMultilineTL, c)
			startCol := int(hl.Range.Start.Col)
			if startCol > 1 {
				r.appendString(string(char.RepeatRune(r.CharacterSet.HighlightMultilineOffset, startCol-1)), c)
			}
			length := int(rw.lastCol) - startCol
			if length < 1 {
				length = 5
			}
			r.appendString(string(char.RepeatRune(r.CharacterSet.HighlightMulti, length)), c)
		} else if hl.Range.End.Line == rw.num {
			r.appendRune(r.CharacterSet.HighlightMultilineBL, c)
			endCol := int(hl.Range.End.Col)
			if endCol > 1 {
				r.appendString(string(char.RepeatRune(r.CharacterSet.HighlightMulti, endCol-1)), c)
			}
			if hl.Message != "" {
				r.appendSpace(1)
				r.appendString(hl.Message, c)
			}
		}
		r.newline()
	}
}

// colorize wraps colorizeText using the token's original source.
func (r *Reporter) colorize(tokens []lexer.Token, i int) string {
	return r.colorizeText(tokens, i, tokens[i].Source)
}

// colorizeText colors arbitrary text based on a token's kind and context.
func (r *Reporter) colorizeText(tokens []lexer.Token, i int, text string) string {
	tok := tokens[i]
	color := r.ColorPalette.TokenColors[tok.Kind]
	if r.ColorPalette == nil || color == "" {
		return text
	}

	nextTok := func(i int) lexer.TokenType {
		if len(tokens) <= i+1 {
			return 0
		}
		return tokens[i+1].Kind
	}
	prevTok := func(i int) lexer.TokenType {
		if i == 0 {
			return 0
		}
		return tokens[i-1].Kind
	}

	next := nextTok(i)
	prev := prevTok(i)

	switch {
	case tok.Kind != lexer.Identifier:
		break
	case isPrimitive(tok.Source),
		prev == lexer.Arrow && next == lexer.LeftCurlyBrace,
		prev == lexer.Type,
		next == lexer.Stroke,
		next == lexer.Question:
		color = r.ColorPalette.TypeColor
	case prev == lexer.Func, next == lexer.LeftParenthesis:
		color = r.ColorPalette.FunctionColor
		if isBuiltinFunc(tok.Source) {
			color = ansi.CodeBlue
		}
	}
	return ansi.Color(color, text)
}

func isPrimitive(name string) bool {
	_, ok := ast.PrimitiveTypeMap[name]
	return ok
}

var builtinFuncs = map[string]struct{}{
	"print": {}, "crashout": {}, "assert": {}, "TODO": {}, "clone": {},
}

func isBuiltinFunc(name string) bool {
	_, ok := builtinFuncs[name]
	return ok
}
