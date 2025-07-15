package errors

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/cli/icons"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type Printer struct {
	Color         bool
	MaxLines      int
	TokenColors   map[lexer.TokenType]string
	TypeColor     string
	FunctionColor string

	tokens    []lexer.Token
	IsRuntime bool
}

func (p *Printer) LoadTokens(tokens []lexer.Token) {
	p.tokens = tokens
	if p.TokenColors == nil {
		p.TokenColors = defaultColors
		p.TypeColor = colorType
		p.FunctionColor = colorFunc
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
	return ansi.CodeBoldRed + title + ansi.CodeResetBoldDim + ": " +
		ansi.CodeResetBold + msg + ansi.CodeResetBold + desc + ansi.CodeReset
}

func ColorizeLine(file string, pos lexer.Position) string {
	var (
		b         strings.Builder
		colon     = ansi.Dim(":")
		formatPos = func(pos int) string {
			return ansi.Yellow(strconv.Itoa(pos))
		}
	)
	b.WriteString(ansi.Cyan(file))
	b.WriteString(colon)
	b.WriteString(formatPos(pos.Line))
	b.WriteString(colon)
	b.WriteString(formatPos(pos.Col))
	return b.String()
}

func isSingleChar(r ranges.Range) bool {
	return r.IsSingleLine() && r.End.Col == r.Start.Col+1
}

func space(n int) []byte {
	return bytes.Repeat([]byte{' '}, n)
}

func (p *Printer) prevTok(i int) (tok lexer.TokenType) {
	if len(p.tokens) < 1 {
		return
	}
	return p.tokens[i-1].Kind
}

func (p *Printer) nextTok(i int) (tok lexer.TokenType) {
	if len(p.tokens) < i+1 {
		return
	}
	return p.tokens[i+1].Kind
}

func isPrimitive(name string) bool {
	_, ok := ast.PrimitiveTypeMap[name]
	return ok
}

func (p *Printer) colorize(i int) string {
	tok := p.tokens[i]
	color := p.TokenColors[tok.Kind]
	next := p.nextTok(i)
	prev := p.prevTok(i)
	switch {
	case tok.Kind != lexer.Identifier:
		break
	case prev == lexer.Func,
		next == lexer.LeftParenthesis:
		color = p.FunctionColor
	case isPrimitive(tok.Source),
		prev == lexer.Arrow && next == lexer.LeftCurlyBrace,
		prev == lexer.Type, next == lexer.Stroke, next == lexer.Question:
		color = p.TypeColor
	}
	return ansi.Color(color, tok.Source)
}

func (p *Printer) PrintError(err KlarError) {
	var (
		b         strings.Builder
		currTok   int
		at        = err.At()
		start     = max(1, at.Start.Line-p.MaxLines+1)
		end       = start + p.MaxLines - 1
		lastCol   = 1
		digitLen  = len(strconv.Itoa(end))
		lineColor = ansi.CodeBlue
		box       = func(char rune) {
			b.WriteString(ansi.Color(lineColor, string(char)))
		}
	)
	if p.IsRuntime {
		lineColor = ansi.CodeMagenta
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
	for line := start; line <= end; line++ {
		if currTok >= len(p.tokens) {
			break
		}
		// Line number
		b.WriteString(fmt.Sprintf("%s%*d ", lineColor, digitLen, line))
		box(icons.BoxLeft)
		b.WriteByte(' ')
		// Each token on line
		for ; currTok < len(p.tokens) && p.tokens[currTok].Line == line; currTok++ {
			tok := p.tokens[currTok]
			if tok.Source == "\n" {
				continue
			}
			tokRange := ranges.FromToken(tok)
			b.Write(space(tok.Col - lastCol))
			if at.RangeIn(tokRange) {
				b.WriteString(ansi.Red(tok.Source))
			} else {
				b.WriteString(p.colorize(currTok))
			}
			lastCol = ranges.FromToken(tok).End.Col
		}
		b.WriteByte('\n')
		// Error highlight
		if at.Start.Line == line {
			var highlight string
			if isSingleChar(at) {
				highlight = "^"
			} else {
				hlen := at.End.Col - at.Start.Col
				if !at.IsSingleLine() {
					hlen = lastCol - at.Start.Col
				}
				highlight = strings.Repeat("~", hlen)
			}
			b.Write(space(digitLen + 1))
			box(icons.BoxLeft)
			b.WriteByte(' ')
			b.Write(space(at.Start.Col - 1))
			b.WriteString(ansi.BoldRed(highlight))
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
