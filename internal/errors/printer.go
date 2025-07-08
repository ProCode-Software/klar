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
	lexer.Hashbang:     tokenColorComment,
	lexer.Type:         tokenColorStorage,
	lexer.Func:         tokenColorStorage,
	lexer.String:       cli.ANSIGreen,
	lexer.Numeric:      tokenColorNumber,
	lexer.Boolean:      tokenColorBoolNil,
	lexer.Nil:          tokenColorBoolNil,
	lexer.Illegal:      tokenColorIllegal,
	lexer.And:          tokenColorOperator,
	lexer.Or:           tokenColorOperator,

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
	"print": true, "panic": true, "assert": true, "TODO": true,
}
var BuiltinTypes = ast.PrimitiveTypeMap

func init() {
	for _, op := range lexer.OperatorMap {
		if _, exists := TokenColors[op]; !exists {
			TokenColors[op] = tokenColorOperator
		}
	}
	for _, kw := range lexer.KeywordMap {
		if _, exists := TokenColors[kw]; !exists {
			TokenColors[kw] = tokenColorKeyword
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
	if _, isType := BuiltinTypes[tok.Source]; isType || tok.Source == "List" {
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
		b                 strings.Builder
		lastTok           lexer.Token
		highlightColor    = cli.ANSIBoldRed
	)
	if _, isWarning := err.(Warning); isWarning {
		highlightColor = cli.ANSIBoldYellow
	}
	for i, tok := range options.Tokens {
		if tok.Position.Line > errPos.Line {
			if i > 0 {
				lastTok = options.Tokens[i-1]
			}
			break
		}
		if tok.Position.Line < minPos.Line {
			continue
		}
		if tok.Position.Line > currLine {
			currLine, currCol = tok.Position.Line, 1
			b.WriteByte('\n')
			b.WriteString(ansi(lineNumberColor, fmt.Sprintf("%4d | ", currLine)))
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
		case err.AtRange().RangeIn(ranges.FromToken(tok)):
			add = ansi(cli.ANSIRed, tok.Source)
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
		b.WriteString(addSpace(tok.Col - currCol))
		b.WriteString(add)
		currCol = tok.Col + len(tok.Source)
	}
	line := ansi(highlightColor, "^")
	if err.AtRange().Start.Line > 0 {
		rang := err.AtRange()
		var len int
		if rang.IsSingleLine() {
			len = rang.LineLength()
		} else {
			len = lastTok.Col - rang.Start.Col + 1
		}
		if len > 1 {
			line = strings.Repeat(ansi(highlightColor, "~"), len)
		}
	}
	if err, ok := err.(ParseError); ok &&
		errPos == err.Position && len(err.Token.Source) > 1 {
		line = strings.Repeat(ansi(highlightColor, "~"), len(err.Token.Source))
	}
	b.WriteString(fmt.Sprintf("\n%[1]*[2]s"+line, 7+errPos.Col-1, " "))
	out := strings.TrimPrefix(b.String(), "\n")
	fmt.Fprintln(os.Stderr, out)
}
