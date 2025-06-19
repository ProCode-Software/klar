package errors

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

const (
	_ ErrorCode = SyntaxErrorPrefix + iota

	ErrUnexpectedToken
	ErrExpectedToken // Expected kind of token but got different type

	// Import
	ErrExpectedDotInBraceImport // Dot required before brace in unqualified import
	ErrAliasInUnqualifiedImport // Alias is not allowed before unqualified import
	ErrImportExpectedModule     // Unqualified import without module name
	ErrImportPrefixDot          // Module name beginning with .
	ErrImportInvalidWildcard    // Wildcard must be last part of module
	ErrImportTooManyWildcard    // More than 1 wildcard
	ErrWildcardAndUnqImport     // Using unqualified import with wildcard
	ErrWildcardAndAlias         // Using alias with wildcard
	ErrEmptyUnqImport           // Empty unqualified import
	ErrImportsGoFirst           // Imports always go before other declarations

	// Punctuation
	ErrUnterminatedString  // A string that was left open
	ErrUnterminatedComment // Block comment was left open
	ErrUnterminatedBrace   // Missing end of [, (, {, or < (generic)
	ErrMisplacedShebang

	// Literal
	ErrInvalidNumber    // Invalid number format
	ErrStringEscape     // Invalid string escape
	ErrUnicodeEscTooBig // Unicode escape over 0x10FFFF
	ErrConsecutiveSep   // Number has consecutive _
	ErrMisplacedSep     // Number has separator somewhere where it's not supposed to
	ErrTrailingSep      // Number has misplaced _
	ErrExpectedHex      // Expected hex digit
	ErrExpectedOctal
	ErrExpectedBinary
	ErrExpectedDecimal
	ErrExpectedParamInLambda

	ErrExpectedSymbolAssign  // Assignment to non-variable or property
	ErrReservedKeyword       // Reserved keyword used as an identifier
	ErrExpectedExpression    // Required expression but got a statement
	ErrInvalidLabelShorthand // Function label shorthand must be an identifier or string member
	ErrInvalidLabel          // Function label can't be number
	ErrMissingFuncParamType  // Required function parameter type

	// Type
	ErrNotEnoughEnumItems      // At least two enum members required
	ErrExpectedTypeAssignment  // Need = or { after type (maybe got EOS)
	ErrRequiredStructFieldType // Struct fields need an explicit type
	ErrExpectedParamInGeneric  // At least one parameter requried in generic
	ErrParenRequiredFunc       // Parentheses required for params: (Int) -> Int instead of Int -> Int

	// When
	ErrForInvalidCondition // Expected assignment or expression in for loop
	ErrInvalidPublic
	ErrUnderscoreWithRest // ... instead of ..._ or _...
	ErrNotAllowedInGuard  // When expression not allowed in when case guard

	ErrRedeclaredVar  // Can't redeclare variable or function
	ErrRedeclaredType // Redeclared type
	ErrRedeclaredEnum // Redeclared enum member
	ErrMethAndFieldSameName
	ErrMethodInOtherScope // Method must be in the same scope as struct definition
)

// A ParseError is a basic Klar parse error.
type ParseError struct {
	Position  lexer.Position
	Range     ranges.Range
	Ranges    []ranges.Range
	ErrorCode ErrorCode
	Token     lexer.Token
	Node      ast.Node
	Params    map[string]any
}

func (e *ParseError) SetParam(key string, value any) ParseError {
	e.Params[key] = value
	return *e
}

func (e ParseError) Error() string {
	var (
		tok  = e.Token
		kind = tok.Kind
		src  = tok.Source
	)
	switch e.ErrorCode {
	default:
		if e.Node != nil {
			return fmt.Sprintf(
				"SyntaxError: %s: in %s", e.ErrorCode.String(), e.Node.Kind(),
			)
		}
		return fmt.Sprintf("SyntaxError: %s: %s %s",
			e.ErrorCode.String(), kind.String(), QuoteToken(tok),
		)
	case ErrExpectedExpression:
		return "SyntaxError: This isn't an expression"
	case ErrExpectedSymbolAssign:
		return "Can't assign to this kind of expression"
	case ErrExpectedToken:
		expToken := e.Params["expected"].(lexer.TokenType)
		expected := FormatTokenType(expToken)
		if src == ";" {
			return "SyntaxError: Semicolons aren't allowed in Klar. Use line breaks to terminate statements"
		}
		switch expToken {
		case lexer.RightCurlyBrace, lexer.RightParenthesis, lexer.GreaterThan, lexer.RightBracket:
			beginMap := map[lexer.TokenType]lexer.TokenType{
				lexer.RightCurlyBrace:  lexer.LeftCurlyBrace,
				lexer.RightParenthesis: lexer.LeftParenthesis,
				lexer.GreaterThan:      lexer.LessThan,
				lexer.RightBracket:     lexer.LeftBracket,
			}
			begin := beginMap[expToken]
			if e.Params["isMap"] == true {
				begin = lexer.HashLeftCurlyBrace
			}
			return fmt.Sprintf("SyntaxError: Expected %s to close %s",
				expected, FormatTokenType(begin),
			)
		}
		return fmt.Sprintf(
			"SyntaxError: I expected %s, but found %s instead",
			expected, NameToken(tok),
		)
	case ErrWildcardAndUnqImport:
		return "SyntaxError: Can't have both '*' and unqualified import in import statement"
	case ErrImportTooManyWildcard:
		return "SyntaxError: There can only be one '*' in module name"
	case ErrWildcardAndAlias:
		return "SyntaxError: Can't use '*' with alias in unqualified import"
	case ErrExpectedDotInBraceImport:
		return "SyntaxError: There should be a '.' before '{' in unqualified import statement"
	case ErrEmptyUnqImport:
		return "SyntaxError: Expected at least 1 unqualified import"
	case ErrImportExpectedModule:
		return "SyntaxError: I expected a module name before '.{' in unqualified import"
	case ErrImportInvalidWildcard:
		return "SyntaxError: '*' should the the last sub-namespace of a module"
	case ErrImportPrefixDot:
		return "SyntaxError: A module name can't start with '.'"
	case ErrUnexpectedToken:
		switch {
		case src == ";":
			return "SyntaxError: Semicolons aren't allowed in Klar. Use line breaks to terminate statements"
		case kind == lexer.EOF:
			return "SyntaxError: Unexpected end of file"
		case kind == lexer.Newline:
			return "SyntaxError: Unexpected newline"
		default:
			return "SyntaxError: I didn't expect " + NameToken(tok)
		}
	case ErrUnterminatedString:
		return fmt.Sprintf("SyntaxError: The string starting at %s was left open", e.Position)
	case ErrExpectedTypeAssignment:
		if kind == lexer.EndOfStatement {
			return "SyntaxError: Types must be assigned a value"
		}
		return "SyntaxError: I expected a type assignment, but found " + NameToken(tok) + " instead"
	case ErrRequiredStructFieldType:
		return "SyntaxError: Struct fields need an explicit type"
	case ErrNotEnoughEnumItems:
		return "SyntaxError: This enum must have at least 2 items, but it has only 1"
	case ErrExpectedHex:
		return "SyntaxError: I expected a hexadecimal digit (0-9, a-f or A-F)"
	case ErrExpectedBinary:
		return "SyntaxError: I expected a binary digit (0-1)"
	case ErrExpectedOctal:
		return "SyntaxError: I expected an octal digit (0-7)"
	case ErrExpectedDecimal:
		return "SyntaxError: I expected a decimal digit (0-9)"
	case ErrUnicodeEscTooBig:
		return "SyntaxError: This Unicode escape should be in the range 0 to 10FFFF"
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
			}
			return "SyntaxError: I expected an expression"
		default:
			return "SyntaxError: Invalid string escape"
		}
	case ErrForInvalidCondition:
		return "SyntaxError: Expected an assignment or expression in for condition"
	case ErrExpectedParamInGeneric:
		return "SyntaxError: At least 1 type parameter is required in generic"
	case ErrInvalidPublic:
		return "SyntaxError: Expected a declaration after public modifier"
	case ErrTrailingSep:
		return "SyntaxError: An underscore can't be at the end of a number"
	case ErrConsecutiveSep:
		return "SyntaxError: Numbers can't have consecutive underscores"
	case ErrMisplacedSep:
		return "SyntaxError: An underscore isn't allowed here"
	case ErrNotAllowedInGuard:
		return "SyntaxError: Case guards can't contain when expressions"
	case ErrUnterminatedComment:
		return "SyntaxError: The comment starting at " + e.Position.String() +
			" was left open"
	case ErrMisplacedShebang:
		return "SyntaxError: Shebang must be on the first line of the file (without any lines or spaces before)"
	case ErrMissingFuncParamType:
		return "SyntaxError: Function parameters must have a type"
	case ErrImportsGoFirst:
		return "SyntaxError: Imports must go before other declarations"
	case ErrMethodInOtherScope:
		return fmt.Sprintf(
			"SyntaxError: Method %s must be declared in the same scope as type %s",
			e.Params["name"], e.Params["structName"],
		)
	case ErrRedeclaredType, ErrRedeclaredVar, ErrRedeclaredEnum:
		var (
			code      = e.ErrorCode
			origPos   = e.Params["origPos"]
			name      = e.Params["name"].(string)
			origType  = e.Params["origType"].(string)
			newType   = e.Params["newType"].(string)
			first, as string
		)
		switch code {
		case ErrRedeclaredType:
			first = "Type "
		case ErrRedeclaredEnum:
			first = "Enum member "
		}
		if origType != newType {
			as = " as " + WithA(origType)
		}
		return fmt.Sprintf("SyntaxError: %s%s was already declared%s at %s",
			first,
			Quote(name), as, origPos,
		)
	}
}

func UnexpectedToken(token lexer.Token) ParseError {
	return ParseError{Position: token.Position, Token: token, ErrorCode: ErrUnexpectedToken}
}

func ExpectedToken(expTokenKind lexer.TokenType, gotToken lexer.Token) ParseError {
	return ParseError{
		Position:  gotToken.Position,
		Token:     gotToken,
		ErrorCode: ErrExpectedToken,
		Params: ErrorParams{
			"expected": expTokenKind,
		},
	}
}

func StringEscape(e lexer.StringEscape) ParseError {
	return ParseError{
		Position:  ranges.Sub(e.ErrorPosition, 0, 1),
		ErrorCode: ErrStringEscape,
		Params: ErrorParams{
			"reason": e.Invalid,
			"type":   e.Type,
			"escape": e.Value,
		},
	}
}

func Token(err ErrorCode, token lexer.Token) ParseError {
	return ParseError{ErrorCode: err, Position: token.Position, Token: token}
}

func Node(err ErrorCode, node ast.Node) ParseError {
	return ParseError{
		ErrorCode: err,
		Node:      node,
		Range:     node.Base().Range,
		Position:  node.Base().Start,
	}
}

func Position(err ErrorCode, pos lexer.Position) ParseError {
	return ParseError{ErrorCode: err, Position: pos}
}

func Range(err ErrorCode, rang ranges.Range) ParseError {
	return ParseError{ErrorCode: err, Range: rang, Position: rang.Start}
}

func TokenPos(err ErrorCode, pos lexer.Position, tok lexer.Token) ParseError {
	return ParseError{ErrorCode: err, Position: pos, Token: tok}
}

func Redeclared(name, kind string, p1, p2 ranges.Range) ParseError {
	var code ErrorCode
	if kind == "Type" {
		code = ErrRedeclaredType
	} else {
		code = ErrRedeclaredVar
	}
	return ParseError{
		Range:     p2,
		Position:  p2.Start,
		ErrorCode: code,
		Params: ErrorParams{
			"origPos": p1.Start,
			"name":    name,
		},
	}
}
