package klarerrs

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/module/imports"
	"github.com/ProCode-Software/klar/internal/ranges"
)

func UnexpectedToken(token lexer.Token) *Error {
	return &Error{
		Range: ranges.FromToken(token),
		Info:  TokenInfo(token),
		Code:  ErrUnexpectedToken,
	}
}

func ExpectedTokenf(msg string, exp lexer.TokenType, got lexer.Token) *Error {
	return &Error{
		Range:  ranges.FromToken(got),
		Info:   TokenInfo(got),
		Code:   ErrExpectedToken,
		Params: ErrorParams{"expected": exp, "msg": msg},
	}
}

func ExpectedToken(expTokenKind lexer.TokenType, gotToken lexer.Token) *Error {
	return &Error{
		Range:  ranges.FromToken(gotToken),
		Info:   TokenInfo(gotToken),
		Code:   ErrExpectedToken,
		Params: ErrorParams{"expected": expTokenKind},
	}
}

func Token(err Code, token lexer.Token) *Error {
	return &Error{
		Code:  err,
		Range: ranges.FromToken(token),
		Info:  TokenInfo(token),
	}
}

func Node(err Code, node ast.Node) *Error {
	return &Error{
		Code:  err,
		Node:  node,
		Range: node.GetRange(),
	}
}

func Position(err Code, pos lexer.Position) *Error {
	return &Error{Code: err, Range: ranges.Offset(pos, 0, 1)}
}

func Range(err Code, rang ranges.Range) *Error {
	return &Error{Code: err, Range: rang}
}

func Slice[T ast.Node](err Code, nodes []T) *Error {
	return &Error{Code: err, Range: ranges.FromSlice(nodes)}
}

func TokenPos(err Code, pos lexer.Position, tok lexer.Token) *Error {
	return &Error{
		Code:  err,
		Range: ranges.Offset(pos, 0, 1),
		Info:  TokenInfo(tok),
	}
}

func TooManyErrors() *Error { return &Error{Code: ErrTooManyErrors} }

func Undefined(name string, rang ranges.Range) *Error {
	return &Error{
		Code:  ErrUndefined,
		Name:  name,
		Range: rang,
		Label: Quote(name) + " doesn't exist",
	}
}

func ImportError(code Code, p imports.ImportPath, err error) *Error {
	return &Error{
		Code: code,
		Info: ModuleErrorInfo{ImportPath: p.String(), Err: err},
	}
}

func ReferenceError(code Code, r ranges.Range, name string) *Error {
	return &Error{
		Code:  code,
		Range: r,
		Name:  name,
	}
}

func TypeError(code Code, r ranges.Range, expectedType, gotType string) *Error {
	return &Error{
		Code:  code,
		Range: r,
		Info:  TypeErrorInfo{ExpectedType: expectedType, GotType: gotType},
	}
}
