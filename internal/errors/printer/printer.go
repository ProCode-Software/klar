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

type file struct {
	tokens []lexer.Token
	rel    string
}

type Printer struct {
	Color    bool
	MaxLines int

	TokenColors   map[lexer.TokenType]string
	TypeColor     string
	FunctionColor string
	EscapeColor   string

	files map[string]file
}

func (p *Printer) LoadTokens(filePath, relPath string, tokens []lexer.Token) {
	if p.files == nil {
		p.files = map[string]file{}
	}
	p.files[filePath] = file{tokens, relPath}
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
	if _, ok := err.(*errors.Warning); ok {
		titleColor = ansi.CodeBoldBrightYellow
	}
	var code string
	if err.GetCode() != 0 {
		code = ansi.Dim(" (" + err.GetCode().Format() + ")")
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
	if pos.Line == 0 || pos.Col == 0 {
		return b.String()
	}
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

func digitLen(x uint32) uint32 {
	if x < 10 {
		return 1
	} else if x < 100 {
		return 2
	} else if x < 1000 {
		return 3
	}
	return uint32(len(strconv.FormatUint(uint64(x), 10)))
}

func (p *Printer) PrintError(err errors.CompileError) {
	if p.MaxLines <= 0 {
		p.MaxLines = 3
	}
	var (
		b    bytes.Buffer
		f    = p.files[err.GetFile()]
		toks = f.tokens

		relPath = f.rel
		at      = err.GetRange()
		start   = uint32(max(1, int64(at.Start.Line)-int64(p.MaxLines)+1)) // 3 lines above
		end     = start + uint32(p.MaxLines) - 1

		lastCol uint32 = 1
		currTok int

		digitLen       = digitLen(end)
		lineColor      = ansi.CodeBlue
		highlightColor = ansi.CodeBrightRed
		box            = func(char rune) {
			b.WriteString(ansi.Color(lineColor, string(char)))
		}
	)
	if f.tokens == nil {
		panic("tokens not defined for file: " + err.GetFile())
	}
	if relPath == "" {
		relPath = err.GetFile()
	}
	if at.Start.Line == 0 {
		b.Write(space(2))
		box(icons.BoxTopLeft)
		box(icons.BoxHorizontal)
		b.WriteByte(' ')
		b.WriteString(ansi.Cyan(relPath))
		b.WriteByte('\n')
		goto printMsg
	}
	if _, ok := err.(*errors.Warning); ok {
		highlightColor = ansi.CodeBrightYellow
	}
	// Error file path
	b.Write(space(digitLen + 1))
	box(icons.BoxTopLeft)
	box(icons.BoxHorizontal)
	b.WriteByte(' ')
	b.WriteString(ColorizeLine(relPath, err.GetRange().Start))
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
		box(icons.BoxVertical)
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

			b.WriteString(ansi.Partial(highlightColor))
			b.Write(highlight)
			b.WriteString(ansi.Partial(ansi.CodeReset))

			b.WriteByte('\n')
			break
		}
	}
printMsg:
	b.WriteString(GetMessage(err))
	b.WriteByte('\n')
	for _, hint := range err.GetHints() {
		b.WriteString(ansi.BoldBrightBlue("  Hint"))
		b.WriteString(ansi.BoldDim(": "))
		cli.Wrap(hint.Message, &b, 80, 4)
		b.WriteByte('\n')
	}
	os.Stderr.Write(b.Bytes())
}

/* Multiline errors:

2 │ a = 2 + `Hello
3 │ ╭───────~~~~~~
4 │ │ World!` + 5
5 │ ╰~~~~~~~~

*/
