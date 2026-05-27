package klon

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonerrs"
)

func (rd *reader) hasTokens() bool { return rd.currTok().Kind != EOF }

// currTok returns the current token.
func (rd *reader) currTok() Token {
	if rd.curr != nil {
		return *rd.curr
	}
	t := rd.readToken()
	rd.curr = &t
	return t
}

// peekTok returns the token after the current token without advancing r.
func (rd *reader) peekTok() Token {
	if rd.peek != nil {
		return *rd.peek
	}
	t := rd.readToken()
	rd.peek = &t
	return t
}

// advanceTok returns the current token and advances r.
func (rd *reader) advanceTok() Token {
	if rd.peek != nil {
		t := *rd.curr
		rd.curr = rd.peek
		rd.peek = nil
		if t.Kind == Newline {
			rd.lastDashes = -1
		}
		return t
	}
	t := rd.currTok()
	rd.curr = nil
	if t.Kind == Newline {
		rd.lastDashes = -1
	}
	return t
}

func (rd *reader) skipLines() {
	for rd.currTok().Kind == Newline {
		rd.advanceTok()
	}
}

func (rd *reader) tokenError(code klonerrs.Code, tok Token, msg string, v ...any) {
	var text string
	if len(v) == 0 {
		text = msg
	} else {
		text = fmt.Sprintf(msg, v...)
	}
	rd.addErrorOrWarning(&Error{
		Code:  code,
		Range: tok.Range(),
		Token: tok,
		Text:  text,
	})
}

func (rd *reader) rangeError(code klonerrs.Code, r ranges.Range, msg string, v ...any) {
	var text string
	if len(v) == 0 {
		text = msg
	} else {
		text = fmt.Sprintf(msg, v...)
	}
	rd.addErrorOrWarning(&Error{Code: code, Range: r, Text: text})
}

func (rd *reader) addErrorOrWarning(err *Error) {
	if rd.ctx != nil && rd.ctx.WarningKinds != nil && rd.ctx.Warnings != nil {
		if _, ok := rd.ctx.WarningKinds[err.Code]; ok {
			err.Warning = true
			rd.ctx.Warnings = append(rd.ctx.Warnings, err)
			return
		}
	}
	rd.errs = append(rd.errs, err)
}

func (rd *reader) expect(
	exp TokenType, code klonerrs.Code, msg string, v ...any,
) Token {
	if curr := rd.currTok(); curr.Kind != exp {
		rd.tokenError(code, curr, msg, v...)
		return curr
	}
	return rd.advanceTok()
}

func (rd *reader) depthUp() {
	if rd.depth++; rd.depth > MaxDepth {
		rd.tokenError(klonerrs.ErrMaxDepth, rd.currTok(), "Too much nesting")
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
