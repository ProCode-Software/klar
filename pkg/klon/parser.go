package klon

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
)

func (rd *reader) hasTokens() bool { return rd.currTok().Kind != EOF }

// currTok returns the current token.
func (rd *reader) currTok() Token {
	if rd.hasCurr {
		return rd.curr
	}
	rd.curr = rd.readToken()
	rd.hasCurr = true
	return rd.curr
}

// peekTok returns the token after the current token without advancing r.
func (rd *reader) peekTok() Token {
	if rd.hasPeek {
		return rd.peek
	}
	rd.peek = rd.readToken()
	rd.hasPeek = true
	return rd.peek
}

// advanceTok returns the current token and advances r.
func (rd *reader) advanceTok() Token {
	if rd.hasPeek {
		t := rd.curr
		rd.curr = rd.peek
		rd.hasCurr = true
		rd.hasPeek = false
		return t
	}
	t := rd.currTok()
	rd.hasCurr = false
	return t
}

func (rd *reader) skipLines() {
	for rd.currTok().Kind == Newline {
		rd.advanceTok()
	}
}

func (rd *reader) tokenError(code Code, tok Token, msg string, v ...any) {
	var text string
	if len(v) == 0 {
		text = msg
	} else {
		text = fmt.Sprintf(msg, v...)
	}
	rd.errs = append(rd.errs, &Error{
		Code:  code,
		Range: tok.Range(),
		Token: tok,
		Text:  text,
	})
}

func (rd *reader) rangeError(code Code, r ranges.Range, msg string, v ...any) {
	var text string
	if len(v) == 0 {
		text = msg
	} else {
		text = fmt.Sprintf(msg, v...)
	}
	rd.errs = append(rd.errs, &Error{Code: code, Range: r, Text: text})
}

func (rd *reader) expectError(
	exp TokenType, code Code, msg string, v ...any,
) Token {
	if curr := rd.currTok(); curr.Kind != exp {
		rd.tokenError(code, curr, msg, v...)
		return curr
	}
	return rd.advanceTok()
}

func (rd *reader) depthUp() {
	if rd.depth++; rd.depth > MaxDepth {
		rd.tokenError(ErrMaxDepth, rd.currTok(), "Too much nesting")
		panic(bailout{})
	}
}

func (rd *reader) depthDown() {
	if rd.depth--; rd.depth < 0 {
		panic("negative depth")
	}
}

func (rd *reader) addParseFlags(flags uint8) (old uint8) {
	old = rd.parseFlags
	rd.parseFlags |= flags
	return
}

func (rd *reader) removeParseFlags(flags uint8) (old uint8) {
	old = rd.parseFlags
	rd.parseFlags &^= flags
	return
}

func (rd *reader) resetParseFlags(old uint8) { rd.parseFlags = old }

func sliceRange[T ast.Node](items []T) ranges.Range {
	if len(items) == 0 {
		return ranges.Range{}
	}
	return ranges.Between(items[0].Pos(), items[len(items)-1].Pos())
}
