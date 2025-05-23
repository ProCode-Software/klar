package errors

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// An ErrorCode is a type of syntax or type error.
//
//go:generate stringer -type=ErrorCode

// ==================
// SYNTAX ERRORS
// ==================
const (
	_ ErrorCode = iota

	ErrUnexpectedToken
	ErrExpectedToken // Expected kind of token but got different type
	ErrExpectedEOS   // Expected end of statement (newline)

	// Import
	ErrExpectedDotInBraceImport     // Dot required before brace in unqualified import
	ErrAliasInUnqualifiedImport     // Alias is not allowed before unqualified import
	ErrImportExpectedModule         // Unqualified import without module name
	ErrImportExpectedIdentAfterType // type TypeName or type *
	ErrImportPrefixDot              // Module name beginning with .
	ErrImportInvalidWildcard        // Wildcard must be last part of module
	ErrImportTooManyWildcard        // More than 1 wildcard
	ErrWildcardAndUnqImport         // Using unqualified import with wildcard

	// Punctuation
	ErrUnterminatedString  // A string that was left open
	ErrUnterminatedComment // Block comment was left open
	ErrUnterminatedBrace   // Missing end of [, (, {, or < (generic)
	ErrStringEscape        // Invalid string escape

	ErrExpectedSymbolAssign // Assignment to non-variable or property
	ErrReservedKeyword      // Reserved keyword used as an identifier
	ErrExpectedExpression   // Required expression but got a statement

	// Type
	ErrNotEnoughEnumItems     // At least two enum members required
	ErrExpectedTypeAssignment // Need = or { after type (maybe got EOS)
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
	default:
		return fmt.Sprintf("SyntaxError: %s: '%s' (type %s)",
			e.Type.String(),
			e.Token.Source, e.Token.Kind.String(),
		)
	case ErrExpectedExpression:
		return "SyntaxError: I expected an expression, but got " + e.ASTItem.Kind() + "instead"
	case ErrExpectedSymbolAssign:
		return "SyntaxError: You can only assign to a variable or property, not " +
			e.ASTItem.Kind()
	case ErrExpectedToken:
		fmt.Println(e.Position, e.Token.Position)
		return fmt.Sprintf(
			"SyntaxError: Expected token '%s', but got '%s' instead",
			e.Params["expected"].(lexer.TokenType).String(),
			e.Token.Source,
		)
	case ErrWildcardAndUnqImport:
		return "SyntaxError: Can't have both '*' and unqualified import in import statement"
	case ErrImportTooManyWildcard:
		return "SyntaxError: There can only be one '*' in module name"
	case ErrExpectedDotInBraceImport:
		return "SyntaxError: There should be a '.' before '{' in unqualified import statement"
	case ErrImportExpectedModule:
		return "SyntaxError: I expected a module name before '.{' in unqualified import"
	case ErrImportInvalidWildcard:
		return "SyntaxError: '*' should the the last sub-namespace of a module"
	case ErrImportPrefixDot:
		return fmt.Sprintf(
			"SyntaxError: Module '%s' in import statement can't start with '.'",
			e.Params["module"].(string),
		)
	case ErrUnexpectedToken:
		switch {
		default:
			return fmt.Sprintf("SyntaxError: Unexpected token %#q (type %s)",
				e.Token.Source,
				e.Token.Kind.String(),
			)
		case e.Token.Source == ";":
			return "SyntaxError: Semicolons aren't allowed in Klar. Use line breaks to terminate statements"
		case e.Token.Kind == lexer.Illegal:
			return "SyntaxError: I don't know what token '" + e.Token.Source + "' is"
		}
	case ErrUnterminatedString:
		return fmt.Sprintf("SyntaxError: The string starting at %v was left open", e.Position)
	case ErrExpectedTypeAssignment:
		if e.Token.Kind == lexer.EndOfStatement || e.Token.Kind == lexer.EOF {
			return "SyntaxError: Types must be assigned a value"
		}
		return "SyntaxError: I expected a type assignment ('=', '{', or inherited type), but got '" + e.Token.Source + "' instead"
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
		Type:     ErrExpectedToken,
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

func NewTokenError(err ErrorCode, token lexer.Token) ParseError {
	return ParseError{
		Type:     err,
		Position: token.Position,
		Token:    token,
	}
}
