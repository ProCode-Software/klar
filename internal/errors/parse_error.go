package errors

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
)

// An ErrorCode is a type of syntax or type error.
//
//go:generate stringer -type=ErrorCode

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

	// Literal
	ErrInvalidNumber  // Invalid number format
	ErrStringEscape   // Invalid string escape
	ErrConsecutiveSep // Number has consecutive _
	ErrMisplacedSep   // Number has misplaced _
	ErrExpectedHex    // Expected hex digit
	ErrExpectedOctal
	ErrExpectedBinary
	ErrExpectedDecimal
	ErrExpectedParamInLambda

	ErrExpectedSymbolAssign  // Assignment to non-variable or property
	ErrReservedKeyword       // Reserved keyword used as an identifier
	ErrExpectedExpression    // Required expression but got a statement
	ErrInvalidLabelShorthand // Function label shorthand must be an identifier or string member

	// Type
	ErrNotEnoughEnumItems      // At least two enum members required
	ErrExpectedTypeAssignment  // Need = or { after type (maybe got EOS)
	ErrRequiredStructFieldType // Struct fields need an explicit type
	ErrExpectedParamInGeneric  // At least one parameter requried in generic
	ErrParenRequiredFunc       // Parentheses required for params: (Int) -> Int instead of Int -> Int

	ErrForInvalidCondition // Expected assignment or expression in for loop
	ErrInvalidPublic
)

type ErrorParams map[string]any

// A ParseError is a basic Klar parse error.
type ParseError struct {
	KlarError
	Position lexer.Position
	Type     ErrorCode
	Token    lexer.Token
	Node     ast.Node
	Params   map[string]any
}

func (e ParseError) Error() string {
	var (
		tok  = e.Token
		kind = tok.Kind
		src  = tok.Source
	)
	switch e.Type {
	default:
		if e.Node != nil {
			return fmt.Sprintf(
				"SyntaxError: %s: %s here", e.Type.String(), e.Node.Kind(),
			)
		}
		return fmt.Sprintf("SyntaxError: %s: %s (%s)",
			e.Type.String(), Quote(tok), FormatTokenType(kind),
		)
	case ErrExpectedExpression:
		return "SyntaxError: I expected an expression, but got " +
			FormatTokenType(kind) + " instead"
	case ErrExpectedSymbolAssign:
		return "SyntaxError: You can only assign to a variable or property, not " +
			e.Node.Kind()
	case ErrExpectedToken:
		if src == ";" {
			return "SyntaxError: Semicolons aren't allowed in Klar. Use line breaks to terminate statements"
		}
		return fmt.Sprintf(
			"SyntaxError: I expected %s, but got %s instead",
			FormatTokenType(e.Params["expected"].(lexer.TokenType)),
			Quote(tok),
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
		return "SyntaxError: A module name can't start with '.'"
	case ErrUnexpectedToken:
		switch {
		case kind == lexer.Illegal:
			return "SyntaxError: I don't know what to do with this " + Quote(tok)
		case src == ";":
			return "SyntaxError: Semicolons aren't allowed in Klar. Use line breaks to terminate statements"
		default:
			return "SyntaxError: I didn't expect this " + QuoteWithoutA(tok)
		}
	case ErrUnterminatedString:
		return fmt.Sprintf("SyntaxError: The string starting at %v was left open", e.Position)
	case ErrExpectedTypeAssignment:
		if kind == lexer.EndOfStatement {
			return "SyntaxError: Types must be assigned a value"
		}
		return "SyntaxError: I expected a type assignment, but got " + Quote(tok) + " instead"
	case ErrRequiredStructFieldType:
		return "SyntaxError: Struct fields need an explicit type"
	case ErrNotEnoughEnumItems:
		return "SyntaxError: This enum must have at least 2 items, but it has only 1"
	case ErrStringEscape:
		reason := e.Params["reason"].(lexer.EscapeError)
		kind := e.Params["type"].(lexer.EscapeType)
		switch reason {
		case lexer.ErrEscapeExpHex:
			return "SyntaxError: I expected a hexadecimal digit (0-9, a-f or A-F)"
		case lexer.ErrEscapeUnknown:
			return "SyntaxError: I don't understand this escape code"
		case lexer.ErrEscapeTooLong, lexer.ErrEscapeTooShort:
			if kind == lexer.EscUnicode {
				return "SyntaxError: Expected between 1-6 hex digits in Unicode escape"
			} else {
				return "SyntaxError: I expected an expression"
			}
		default:
			return "SyntaxError: Invalid string escape"
		}
	case ErrForInvalidCondition:
		return "Expected an assignment or expression in for condition"
	case ErrExpectedParamInGeneric:
		return "At least 1 type parameter is required in generic"
	}
}

func UnexpectedTokenError(token lexer.Token) ParseError {
	return ParseError{Position: token.Position, Token: token, Type: ErrUnexpectedToken}
}

func ExpectedTokenError(expTokenKind lexer.TokenType, gotToken lexer.Token) ParseError {
	return ParseError{
		Position: gotToken.Position,
		Token:    gotToken,
		Type:     ErrExpectedToken,
		Params: ErrorParams{
			"expected": expTokenKind,
		},
	}
}
func UnterminatedStringError(startPos lexer.Position) ParseError {
	return ParseError{Position: startPos, Type: ErrUnterminatedString}
}

func InvalidEscapeError(e lexer.StringEscape, pos lexer.Position) ParseError {
	return ParseError{
		Position: pos,
		Type:     ErrStringEscape,
		Params: ErrorParams{
			"reason": e.Invalid,
			"type":   e.Type,
			"escape": e.Value,
		},
	}
}

func NewTokenError(err ErrorCode, token lexer.Token) ParseError {
	return ParseError{Type: err, Position: token.Position, Token: token}
}

func NewNodeError(err ErrorCode, node ast.Node) ParseError {
	return ParseError{Type: err, Node: node}
}

func NewPositionError(err ErrorCode, pos lexer.Position) ParseError {
	return ParseError{Type: err, Position: pos}
}

func NewTokenPosError(err ErrorCode, pos lexer.Position, tok lexer.Token) ParseError {
	return ParseError{Type: err, Position: pos, Token: tok}
}
