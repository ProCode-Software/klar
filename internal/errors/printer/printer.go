package printer

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/char"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/cli/icons"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type Printer struct {
	Color    bool
	MaxLines int

	TokenColors   map[lexer.TokenType]string
	TypeColor     string
	FunctionColor string
	EscapeColor   string

	tokens map[string][]lexer.Token // File paths
	rel    map[string]string
}

func (p *Printer) LoadTokens(filePath, relPath string, tokens []lexer.Token) {
	if p.tokens == nil {
		p.tokens = map[string][]lexer.Token{}
	}
	if p.rel == nil {
		p.rel = map[string]string{}
	}
	p.tokens[filePath] = tokens
	p.rel[filePath] = relPath
	if p.TokenColors == nil {
		p.TokenColors = defaultColors
		p.TypeColor = colorType
		p.FunctionColor = colorFunc
	}
}

var caret = []byte{'^'}

func GetMessage(err errors.CompileError) string {
	var (
		title, msg, desc string
		parts            = strings.SplitAfterN(err.Error(), ": ", 3)
		first            = parts[0]
		titleColor       = ansi.CodeBoldBrightRed
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
	if _, ok := err.(errors.Warning); ok {
		titleColor = ansi.CodeBoldBrightYellow
	}
	var code string
	if err.Code() != 0 {
		code = ansi.Dim(" (" + err.Code().Format() + ")")
	}
	return ansi.Color(titleColor, title) + ansi.BoldDim(": ") + 
		ansi.Bold(msg) + desc + code
}

func ColorizeLine(file string, pos lexer.Position) string {
	var (
		b         strings.Builder
		colon     = ansi.Dim(":")
		formatPos = func(pos uint32) string {
			return ansi.Yellow(strconv.Itoa(int(pos)))
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

func space(n uint32) []byte {
	if n > 10000000 {
		panic("overflow of n")
	}
	arr := make([]byte, n)
	for i := range n {
		arr[i] = ' '
	}
	return arr
}

func (p *Printer) prevTok(tokens []lexer.Token, i int) (tok lexer.TokenType) {
	if i == 0 {
		return
	}
	return tokens[i-1].Kind
}

func (p *Printer) nextTok(tokens []lexer.Token, i int) (tok lexer.TokenType) {
	if len(tokens) <= i+1 {
		return
	}
	return tokens[i+1].Kind
}

func isPrimitive(name string) bool {
	_, ok := ast.PrimitiveTypeMap[name]
	return ok
}

func isBuiltinFunc(name string) bool {
	_, ok := builtinFuncs[name]
	return ok
}

func (p *Printer) colorize(tokens []lexer.Token, i int) string {
	tok := tokens[i]
	color := p.TokenColors[tok.Kind]
	if !p.Color {
		color = ""
	}
	next := p.nextTok(tokens, i)
	prev := p.prevTok(tokens, i)
	switch {
	case tok.Kind != lexer.Identifier:
		break
	case isPrimitive(tok.Source),
		prev == lexer.Arrow && next == lexer.LeftCurlyBrace,
		prev == lexer.Type,
		next == lexer.Stroke,
		next == lexer.Question:
		color = p.TypeColor
	case prev == lexer.Func, next == lexer.LeftParenthesis:
		color = p.FunctionColor
		if isBuiltinFunc(tok.Source) {
			color = colorBuiltin
		}
	}
	return ansi.Color(color, tok.Source)
}

func (p *Printer) PrintError(err errors.CompileError) {
	if p.MaxLines <= 0 {
		p.MaxLines = 3
	}
	var (
		b              bytes.Buffer
		currTok        int
		toks                  = p.tokens[err.GetFile()]
		at                    = err.At()
		start                 = uint32(max(1, int64(at.Start.Line)-int64(p.MaxLines)+1))
		end                   = start + uint32(p.MaxLines) - 1
		lastCol        uint32 = 1
		digitLen              = uint32(len(strconv.FormatUint(uint64(end), 10)))
		lineColor             = ansi.CodeBlue
		highlightColor        = ansi.CodeBrightRed
		relPath               = p.rel[err.GetFile()]
		box                   = func(char rune) {
			b.WriteString(ansi.Color(lineColor, string(char)))
		}
	)
	if toks == nil {
		panic("tokens not defined for file: " + err.GetFile())
	}
	if relPath == "" {
		relPath = err.GetFile()
	}
	if at.Start.Line == 0 {
		b.WriteString(ColorizeLine(relPath, err.At().Start))
		b.WriteByte('\n')
		goto printMsg
	}
	if _, ok := err.(errors.Warning); ok {
		highlightColor = ansi.CodeBrightYellow
	}
	// Error file path
	b.Write(space(digitLen + 1))
	box(icons.BoxTopLeft)
	box(icons.BoxLine)
	b.WriteByte(' ')
	b.WriteString(ColorizeLine(relPath, err.At().Start))
	b.WriteByte('\n')

	// Get first token
	for i, tok := range toks {
		if tok.Position.Line == start {
			currTok = i
			break
		}
	}
	// Print each line
	for line := start; line <= end; line++ {
		if currTok >= len(toks) {
			break
		}
		// Line number
		b.WriteString(fmt.Sprintf("%s%*d ", ansi.Partial(lineColor), digitLen, line))
		box(icons.BoxSide)
		b.WriteByte(' ')
		// Each token on line
		for lastCol = 1; currTok < len(toks) && toks[currTok].Line == line; currTok++ {
			tok := toks[currTok]
			if tok.Source == "\n" {
				continue
			}
			tokRange := ranges.FromToken(tok)
			b.Write(space(tok.Col - lastCol)) // NOTE: Crashes if negative
			if at.RangeIn(tokRange) {
				b.WriteString(ansi.Color(highlightColor, tok.Source))
			} else {
				b.WriteString(p.colorize(toks, currTok))
			}
			lastCol = tokRange.End.Col
		}
		b.WriteByte('\n')
		// Error highlight
		if at.Start.Line == line {
			var highlight []byte
			if isSingleChar(at) {
				highlight = caret
			} else {
				hlen := at.End.Col - at.Start.Col
				if !at.IsSingleLine() {
					hlen = lastCol - at.Start.Col
				}
				highlight = char.Repeat('~', int(hlen))
			}
			b.Write(space(digitLen + 1))
			box(icons.Bullet)
			b.WriteByte(' ')
			b.Write(space(at.Start.Col - 1))

			b.WriteString(ansi.Partial(ansi.CodeBold + highlightColor))
			b.Write(highlight)
			b.WriteString(ansi.Partial(ansi.CodeReset))

			b.WriteByte('\n')
			break
		}
	}
printMsg:
	b.WriteString(GetMessage(err))
	b.WriteByte('\n')
	os.Stderr.Write(b.Bytes())
	for _, hint := range err.GetHints() {
		cli.HintIndent(hint)
	}
}

/* Multiline errors:

2 │ a = 2 + `Hello
3 │ ╭───────~~~~~~
4 │ │ World!` + 5
5 │ ╰~~~~~~~~

*/
