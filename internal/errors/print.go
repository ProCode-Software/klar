package errors

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/cli/icons"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type Printer struct {
	Color       bool
	MaxLines    int
	TokenColors map[lexer.TokenType]string

	tokens    []lexer.Token
	IsRuntime bool
}

func (p *Printer) LoadTokens(tokens []lexer.Token) {
	p.tokens = tokens
	if p.TokenColors == nil {
		p.TokenColors = defaultColors
	}
}

func GetMessage(err KlarError) string {
	var (
		title, msg, desc string
		parts            = strings.SplitAfterN(err.Error(), ": ", 3)
		first            = parts[0]
	)
	switch len(parts) {
	case 3:
		desc = parts[2]
		fallthrough
	case 2:
		title = strings.TrimSuffix(first, ": ")
		msg = parts[1]
	default:
		title, msg = "Error", first
	}
	return ansi.BoldRed + title + ansi.ResetBold + ": " +
		ansi.Bold + msg + ansi.ResetBold + desc + ansi.Reset
}

func ColorizeLine(file string, pos lexer.Position) string {
	var (
		b         strings.Builder
		colon     = ansi.Color(ansi.Dim, ":")
		formatPos = func(pos int) string {
			return ansi.Color(ansi.Yellow, strconv.Itoa(pos))
		}
	)
	b.WriteString(ansi.Color(ansi.Cyan, file))
	b.WriteString(colon)
	b.WriteString(formatPos(pos.Line))
	b.WriteString(colon)
	b.WriteString(formatPos(pos.Col))
	return b.String()
}

/* func (p *Printer) groupRanges(err KlarError) []ranges.Range {
	first := err.At()
	var currLint int
	var groups [][]ranges.Range
	for _, r := range err.GetRanges() {
	}
} */

func (p *Printer) colorize(tok lexer.Token) string {
	return ansi.Color(p.TokenColors[tok.Kind], tok.Source)
}

func (p *Printer) canGroupRanges(err KlarError) bool {
	all := []ranges.Range{err.At()}
	all = append(all, err.GetRanges()...)
	var lines int
	for _, r := range all {
		if r.IsZero() {
			continue
		}
		lines += r.Lines()
		if lines > p.MaxLines+1 {
			return false
		}
	}
	return true
}

func firstLine(err KlarError) int {
	earliest := err.At().Start.Line
	for _, r := range err.GetRanges() {
		if r.IsZero() {
			continue
		}
		if line := r.Start.Line; line < earliest {
			earliest = line
		}
	}
	return earliest
}

func isSingleChar(r ranges.Range) bool {
	return r.IsSingleLine() && r.End.Col == r.Start.Col+1
}

func space(n int) []byte {
	return bytes.Repeat([]byte{' '}, n)
}

func orderedRanges(err KlarError) (all []ranges.Range) {
	r := err.GetRanges()
	all = make([]ranges.Range, 1, len(r)+1)
	all[0] = err.At()
	all = append(all, err.GetRanges()...)
	all = ranges.Sort(all...)
	return
}

func (p *Printer) PrintError(err KlarError) {
	var (
		b                  strings.Builder
		start              = max(1, firstLine(err)-p.MaxLines)
		end                = start + p.MaxLines
		lastCol            = 1
		currTok, currRange int
		digitLen           = len(strconv.Itoa(end))
		lineColor          = ansi.Blue
		allRanges          = orderedRanges(err)
		box                = func(char rune) {
			b.WriteString(ansi.Color(lineColor, string(char)))
		}
	)
	if p.IsRuntime {
		lineColor = ansi.Magenta
	}
	// Error file path
	b.Write(space(digitLen + 1))
	box(icons.BoxTopLeft)
	box(icons.BoxTop)
	b.WriteByte(' ')
	b.WriteString(ColorizeLine(err.GetFile(), err.At().Start))
	b.WriteByte('\n')

	// Get first token
	for i, tok := range p.tokens {
		if tok.Position.Line == start {
			currTok = i
			break
		}
	}

	// Print each line
	for line := start; line < end; line++ {
		if currTok >= len(p.tokens) {
			break
		}
		b.WriteString(fmt.Sprintf("%s%*d ", lineColor, digitLen, line))
		box(icons.BoxLeft)
		b.WriteByte(' ')
		for ; currTok < len(p.tokens) &&
			p.tokens[currTok].Line == line; currTok++ {
			tok := p.tokens[currTok]
			if tok.Source == "\n" {
				continue
			}
			tokRange := ranges.FromToken(tok)
			var color string
			if r := allRanges[currRange]; r.RangeIn(tokRange) {
				color = ansi.Red
			}
			b.Write(space(tok.Col - lastCol))
			b.WriteString(color + p.colorize(tok) + ansi.Reset)

			lastCol = ranges.FromToken(tok).End.Col
		}
		b.WriteByte('\n')
		if r := allRanges[currRange]; !r.IsZero() && r.Start.Line == line {
			b.Write(space(digitLen + 1))
			box(icons.BoxLeft)
			b.WriteByte(' ')
			if isSingleChar(r) {
				b.Write(space(r.Start.Col - 1))
				b.WriteString(ansi.Color(ansi.Red, "^"))
			} else {
				b.Write(space(r.Start.Col - 1))
				b.WriteString(ansi.Color(ansi.Red, strings.Repeat("~", r.End.Col-r.Start.Col)))
			}
			b.WriteByte('\n')
		}
		lastCol = 1
	}

	msg := GetMessage(err)
	b.WriteString(msg)
	for _, hint := range err.GetHints() {
		cli.HintIndent(hint)
	}
	fmt.Fprintln(os.Stderr, b.String())
}
