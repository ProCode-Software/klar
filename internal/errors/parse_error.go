package errors

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// An ErrorCode is a type of syntax or type error.
//
//go:generate stringer -type=ErrorCode
type ErrorCode int

// ==================
// SYNTAX ERRORS
// ==================
const (
	_ ErrorCode = iota

	ErrUnexpectedToken
	ErrExpectedToken      // Expected kind of token but got different type
	ErrExpectedEOS        // Expected end of statement (newline)
	ErrNotEnoughEnumItems // At least two enum members required

	ErrUnterminatedString  // A string that was left open
	ErrUnterminatedComment // Block comment was left open
	ErrUnterminatedBrace   // Missing end of [, (, {, or < (generic)
	ErrStringEscape        // Invalid string escape

	ErrReservedKeyword // Reserved keyword used as an identifier
)

type ErrorParams map[string]any

// A ParseError is a basic Klar parse error.
type ParseError struct {
	Position lexer.Position
	Type     ErrorCode
	Token    lexer.Token
	ASTItem  ast.ASTItem
	Params   map[string]any
}

func (e ParseError) Error() string {
	switch e.Type {
	case ErrExpectedToken:
		return fmt.Sprintf(
			"SyntaxError: Expected token '%s', got '%s'",
			lexer.TokenTypes[e.Params["expected"].(lexer.TokenType)],
			e.Token.Source,
		)
	case ErrUnexpectedToken:
		switch {
		default:
			return fmt.Sprintf("SyntaxError: Unexpected token '%s' (type %s)",
				e.Token.Source,
				lexer.TokenTypes[e.Token.Kind],
			)
		case e.Token.Kind == lexer.Illegal:
			return "SyntaxError: I don't know what token '" + e.Token.Source + "' is"
		case e.Token.Source == ";":
			return "SyntaxError: Semicolons don't terminate statements in Klar, use line break instead"
		}
	case ErrUnterminatedString:
		return fmt.Sprintf("SyntaxError: The string starting at %v was left open", e.Position)
	default:
		return fmt.Sprintf("SyntaxError: %s, error token is '%s'", e.Type.String(), e.Token.Source)
	}
}

func UnknownTokenError(token lexer.Token) ParseError {
	return ParseError{
		Position: token.Position,
		Token:    token,
		Type:     ErrUnexpectedToken,
	}
}

func ExpectedTokenError(
	expTokenKind lexer.TokenType, gotToken lexer.Token, position lexer.Position,
) ParseError {
	return ParseError{
		Position: position,
		Token:    gotToken,
		Type:     ErrUnexpectedToken,
		Params: ErrorParams{
			"expected": expTokenKind,
		},
	}
}
func UnterminatedStringError(startPos lexer.Position) ParseError {
	return ParseError{
		Position: startPos,
		Type:     ErrUnterminatedString,
	}
}

func InvalidEscapeError(
	reason lexer.StringEscapeErrorType, pos lexer.Position, esc string,
) ParseError {
	return ParseError{
		Position: pos,
		Type:     ErrStringEscape,
		Params: ErrorParams{
			"reason": reason,
			"escape": esc,
		},
	}
}
