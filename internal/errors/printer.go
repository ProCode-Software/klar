package errors

import (
	"fmt"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

const (
	tokenColorKeyword     = cli.ANSIRed
	tokenColorOperator    = cli.ANSIRed
	tokenColorIllegal     = "\x1B[1;3;31m"
	tokenColorStorage     = cli.ANSIBlue
	tokenColorNumber      = cli.ANSIYellow
	tokenColorBoolNil     = tokenColorNumber
	tokenColorComment     = cli.ANSIDim
	tokenColorPunc        = cli.ANSIDim
	tokenColorDefault     = cli.ANSIReset
	tokenColorType        = cli.ANSICyan // Semantic
	tokenColorTypeBuiltin = tokenColorType
	tokenColorFunc        = cli.ANSIMagenta
	tokenColorBuiltinFunc = cli.ANSIYellow
	lineNumberColor       = cli.ANSIDim
)

type TokenColorMap = map[lexer.TokenType]string

var TokenColors = TokenColorMap{
	lexer.BlockComment: tokenColorComment,
	lexer.LineComment:  tokenColorComment,
	lexer.For:          tokenColorKeyword,
	lexer.Return:       tokenColorKeyword,
	lexer.Type:         tokenColorStorage,
	lexer.Func:         tokenColorStorage,
	lexer.Next:         tokenColorKeyword,
	lexer.When:         tokenColorKeyword,
	lexer.Public:       tokenColorKeyword,
	lexer.Import:       tokenColorKeyword,
	lexer.String:       cli.ANSIGreen,
	lexer.Numeric:      tokenColorNumber,
	lexer.Boolean:      tokenColorBoolNil,
	lexer.Nil:          tokenColorBoolNil,
	lexer.Illegal:      tokenColorIllegal,

	// Punctuation
	lexer.Dot:                tokenColorPunc,
	lexer.At:                 tokenColorPunc,
	lexer.Comma:              tokenColorPunc,
	lexer.Colon:              tokenColorPunc,
	lexer.HashLeftCurlyBrace: tokenColorPunc,
	lexer.LeftBracket:        tokenColorPunc,
	lexer.LeftCurlyBrace:     tokenColorPunc,
	lexer.LeftParenthesis:    tokenColorPunc,
	lexer.RightBracket:       tokenColorPunc,
	lexer.RightCurlyBrace:    tokenColorPunc,
	lexer.RightParenthesis:   tokenColorPunc,
}

var BuiltinFuncs = map[string]bool{
	"print": true, "panic": true, "assert": true,
}
var BuiltinTypes = ast.PrimitiveTypeMap

func init() {
	for _, op := range lexer.OperatorMap {
		if _, exists := TokenColors[op]; !exists {
			TokenColors[op] = tokenColorOperator
		}
	}
}

type PrintOptions struct {
	Tokens   []lexer.Token
	Color    bool
	MaxLines int
	Semantic bool // Determine colour by neighbouring tokens
}

func ansi(code string, str string) string {
	return code + str + cli.ANSIReset
}

func colorize(tok lexer.Token) string {
	return ansi(TokenColors[tok.Kind], tok.Source)
}

func semanticFunc(tok lexer.Token) string {
	if _, isType := BuiltinTypes[tok.Source]; isType {
		return ansi(tokenColorType, tok.Source)
	}
	if _, isBuiltin := BuiltinFuncs[tok.Source]; isBuiltin {
		return ansi(tokenColorBuiltinFunc, tok.Source)
	}
	return ansi(tokenColorFunc, tok.Source)
}

func isPrimitiveType(name string) bool {
	_, isBuiltin := BuiltinTypes[name]
	return isBuiltin
}

func semanticType(tok lexer.Token) string {
	if _, isBuiltin := BuiltinTypes[tok.Source]; isBuiltin {
		return ansi(tokenColorTypeBuiltin, tok.Source)
	}
	return ansi(tokenColorType, tok.Source)
}

func addSpace(num int) string {
	return strings.Repeat(" ", num)
}

func PrintError(err KlarError, options PrintOptions) {
	var (
		errPos            = err.At()
		minPos            = ranges.Sub(errPos, options.MaxLines-1, 0)
		currLine, currCol int
		out               string
	)
	for i, tok := range options.Tokens {
		if tok.Position.Line > errPos.Line {
			break
		}
		if tok.Position.Line < minPos.Line {
			continue
		}
		if tok.Position.Line > currLine {
			currLine, currCol = tok.Position.Line, 1
			out += "\n" + ansi(lineNumberColor, fmt.Sprintf("%4d | ", currLine))
		}
		if tok.Kind == lexer.Newline || tok.Kind == lexer.EndOfStatement {
			continue
		}
		nextIs := func(kind lexer.TokenType) bool {
			return len(options.Tokens) > i+1 && options.Tokens[i+1].Kind == kind
		}
		prevIs := func(kind lexer.TokenType) bool {
			return i > 0 && options.Tokens[i-1].Kind == kind
		}
		var add string
		switch {
		case !options.Color:
			add = tok.Source
		case tok.Kind != lexer.Identifier:
			fallthrough
		default:
			add = colorize(tok)
		case isPrimitiveType(tok.Source):
			add = ansi(tokenColorTypeBuiltin, tok.Source)
		case nextIs(lexer.LeftParenthesis):
			add = semanticFunc(tok)
		case prevIs(lexer.Type), prevIs(lexer.Func) && nextIs(lexer.Dot):
			add = semanticType(tok)
		}
		out += addSpace(tok.Col-currCol) + add
		currCol = tok.Col + len(tok.Source)
	}
	line := ansi(cli.ANSIBoldRed, "^")
	if err, ok := err.(ParseError); ok &&
		errPos == err.Position && len(err.Token.Source) > 1 {
		line = strings.Repeat(ansi(cli.ANSIBoldRed, "~"), len(err.Token.Source))
	}
	out += fmt.Sprintf("\n%[1]*[2]s"+line, 7+errPos.Col-1, " ")
	out = strings.TrimPrefix(out, "\n")
	fmt.Fprintln(os.Stderr, out)
}
