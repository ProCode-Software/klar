package klarerrs

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)


func UnexpectedToken(token lexer.Token) *Error {
	return &Error{
		Range:     ranges.FromToken(token),
		Info:     TokenInfo(token),
		Code: ErrUnexpectedToken,
	}
}

func ExpectedTokenf(msg string, exp lexer.TokenType, got lexer.Token) *Error {
	return &Error{
		Range:     ranges.FromToken(got),
		Info: TokenInfo(got),
		Code: ErrExpectedToken,
		Params:    ErrorParams{"expected": exp, "msg": msg},
	}
}

func ExpectedToken(expTokenKind lexer.TokenType, gotToken lexer.Token) *Error {
	return &Error{
		Range:     ranges.FromToken(gotToken),
		Token:     gotToken,
		Code: ErrExpectedToken,
		Params:    ErrorParams{"expected": expTokenKind},
	}
}

func Token(err Code, token lexer.Token) *Error {
	return &Error{
		Code: err,
		Range:     ranges.FromToken(token),
		Token:     token,
	}
}

func Node(err Code, node ast.Node) *Error {
	return &Error{
		Code: err,
		Node:      node,
		Range:     node.GetRange(),
	}
}

func Position(err Code, pos lexer.Position) *Error {
	return &Error{Code: err, Range: ranges.Offset(pos, 0, 1)}
}

func Range(err Code, rang ranges.Range) *Error {
	return &Error{Code: err, Range: rang}
}

func Slice[T ast.Node](err Code, nodes []T) *Error {
	return &Error{
		Code: err,
		Range: ranges.Range{
			Start: nodes[0].GetRange().Start,
			End:   nodes[len(nodes)-1].GetRange().End,
		},
	}
}

func TokenPos(err Code, pos lexer.Position, tok lexer.Token) *Error {
	return &Error{
		Code: err,
		Range:     ranges.Offset(pos, 0, 1),
		Token:     tok,
	}
}
