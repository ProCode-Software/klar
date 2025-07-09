package errors

import (
	"fmt"
	"reflect"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
)

const (
	_ ErrorCode = SyntaxErrorPrefix + iota

	ErrUnexpectedToken
	ErrExpectedToken // Expected kind of token but got different type

	// Import
	ErrAliasInUnqualifiedImport // Alias is not allowed before unqualified import
	ErrImportExpectedModule     // Unqualified import without module name
	ErrImportInvalidWildcard    // Wildcard must be last part of module
	ErrImportTooManyWildcard    // More than 1 wildcard
	ErrWildcardAndUnqImport     // Using unqualified import with wildcard
	ErrWildcardAndAlias         // Using alias with wildcard
	ErrEmptyUnqImport           // Empty unqualified import
	ErrImportsGoFirst           // Imports always go before other declarations

	// Punctuation
	ErrUnterminatedString  // A string that was left open
	ErrUnterminatedComment // Block comment was left open
	ErrUnterminatedRegex   // Missing / in regex literal
	ErrMisplacedShebang

	// Literal
	ErrStringEscape     // Invalid string escape
	ErrUnicodeEscTooBig // Unicode escape over 0x10FFFF
	ErrConsecutiveSep   // Number has consecutive _
	ErrMisplacedSep     // Number has separator somewhere where it's not supposed to
	ErrTrailingSep      // Number has misplaced _
	ErrExpectedHex      // Expected hex digit
	ErrExpectedOctal
	ErrExpectedBinary
	ErrExpectedDecimal
	ErrExpectedParamInLambda // Non-variable or variable tuple used in lambda
	ErrInvalidVersionLit     // Invalid version literal syntax

	ErrExpectedSymbolAssign  // Assignment to non-variable or property
	ErrReservedKeyword       // Reserved keyword used as an identifier
	ErrExpectedExpression    // Required expression but got a statement
	ErrInvalidLabelShorthand // Function label shorthand must be an identifier or string member
	ErrInvalidLabel          // Function label can't be number
	ErrGenericInFuncAlias    // Function aliases can't have generics
	ErrMissingFuncParamType  // Required function parameter type

	// Type
	ErrNotEnoughEnumItems      // At least two enum members required
	ErrEnumParamAndValue       // Enum items with parameters can't have a value assigned
	ErrExpectedTypeAssignment  // Need = or { after type (maybe got EOS)
	ErrCannotTellStructOrEnum  // Don't know if enum or struct from one identifier
	ErrRequiredStructFieldType // Struct fields need an explicit type
	ErrEmptyGeneric            // At least one parameter requried in generic
	ErrParenRequiredFunc       // Parentheses required for params: (Int) -> Int instead of Int -> Int

	// When
	ErrForInvalidCond // Expected assignment or expression in for loop
	ErrInvalidPublic
	ErrUnderscoreWithRest // ... instead of ..._ or _...
	ErrNotAllowedInGuard  // When expression not allowed in when case guard

	ErrRedeclaredVar        // Can't redeclare variable or function
	ErrRedeclaredType       // Redeclared type
	ErrRedeclaredEnum       // Redeclared enum member
	ErrRedeclaredField      // Struct or interface field redeclared
	ErrMethAndFieldSameName // Field and method have the same name
	ErrMethodInOtherScope   // Method must be in the same scope as struct definition
	ErrProvenUnreachable    // Unreachable statement after return/break/next
	ErrUnusedValue          // Unused literal expression statement
	ErrReturnOutsideFunc    // Return statement not allowed outside of function
)

// A ParseError is a basic Klar parse error.
type ParseError struct {
	Position  lexer.Position
	Range     ranges.Range
	Ranges    []ranges.Range
	ErrorCode ErrorCode
	Token     lexer.Token
	Node      ast.Node
	Hints     []string
	Params    map[string]any
}

func (e *ParseError) SetParam(key string, value any) ParseError {
	e.Params[key] = value
	return *e
}

func (e ParseError) Error() string {
	return "SyntaxError: " + e.error()
}

func (e ParseError) error() string {
	var (
		tok  = e.Token
		kind = tok.Kind
		src  = tok.Source
	)
	switch e.ErrorCode {
	default:
		if e.Node != nil {
			kind := reflect.TypeOf(e.Node).Name()
			return fmt.Sprintf(
				"%s: in %s", e.ErrorCode.String(), kind,
			)
		}
		return fmt.Sprintf("%s: %s %s",
			e.ErrorCode.String(), kind.String(), QuoteToken(tok),
		)
	case ErrExpectedExpression:
		return "This isn't an expression"
	case ErrExpectedSymbolAssign:
		return "Can't assign to this kind of expression"
	case ErrExpectedToken:
		expToken := e.Params["expected"].(lexer.TokenType)
		expected := FormatTokenType(expToken)
		if src == ";" {
			return "Semicolons aren't allowed in Klar. Use line breaks to terminate statements"
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
			if e.Params != nil && e.Params["isMap"] == true {
				begin = lexer.HashLeftCurlyBrace
			}
			return fmt.Sprintf("Expected %s to close %s",
				expected, FormatTokenType(begin),
			)
		}
		return fmt.Sprintf(
			"I expected %s, but found %s instead",
			expected, NameToken(tok),
		)
	case ErrWildcardAndUnqImport:
		return "Can't have both '*' and unqualified import in import statement"
	case ErrImportTooManyWildcard:
		return "There can only be one '*' in module name"
	case ErrWildcardAndAlias:
		return "Can't use '*' with alias in unqualified import"
	case ErrEmptyUnqImport:
		return "Expected at least 1 unqualified import"
	case ErrImportExpectedModule:
		return "I expected a module name before '.{' in unqualified import"
	case ErrImportInvalidWildcard:
		return "'*' should be at the end of the module name"
	case ErrAliasInUnqualifiedImport:
		return "Can't use alias with an unqualified import"
	case ErrUnexpectedToken:
		switch {
		case src == ";":
			return "Semicolons aren't allowed in Klar. Use line breaks to terminate statements"
		case kind == lexer.EOF:
			return "Unexpected end of file"
		case kind == lexer.Newline:
			return "Unexpected newline"
		default:
			return "I didn't expect " + NameToken(tok)
		}
	case ErrUnterminatedString:
		return fmt.Sprintf("The string starting at %s was left open", e.Position)
	case ErrUnterminatedRegex:
		return fmt.Sprintf("The regular expression starting at %s was left open", e.Position)
	case ErrExpectedTypeAssignment:
		if kind == lexer.EndOfStatement {
			return "Types must be assigned a value"
		}
		return "I expected a type assignment, but found " + NameToken(tok) + " instead"
	case ErrRequiredStructFieldType:
		return "Struct fields need an explicit type"
	case ErrNotEnoughEnumItems:
		return "This enum must have at least 2 items, but it has only 1"
	case ErrExpectedHex:
		return "I expected a hexadecimal digit (0-9, a-f or A-F)"
	case ErrExpectedBinary:
		return "I expected a binary digit (0-1)"
	case ErrExpectedOctal:
		return "I expected an octal digit (0-7)"
	case ErrExpectedDecimal:
		return "I expected a decimal digit (0-9)"
	case ErrUnicodeEscTooBig:
		return "This Unicode escape should be in the range 0 to 10FFFF"
	case ErrStringEscape:
		reason := e.Params["reason"].(lexer.EscapeError)
		kind := e.Params["type"].(lexer.EscapeType)
		switch reason {
		case lexer.ErrEscapeExpHex:
			return "I expected a hexadecimal digit (0-9, a-f or A-F)"
		case lexer.ErrEscapeUnknown:
			return "I don't understand this escape code"
		case lexer.ErrEscapeTooLong, lexer.ErrEscapeTooShort:
			if kind == lexer.EscUnicode {
				return "Expected between 1-6 hex digits in Unicode escape"
			}
			return "I expected an expression"
		default:
			return "Invalid string escape"
		}
	case ErrForInvalidCond:
		return "Expected an assignment or expression in for condition"
	case ErrEmptyGeneric:
		return "At least 1 type parameter is required in generic"
	case ErrInvalidPublic:
		return "Expected a declaration after public modifier"
	case ErrTrailingSep:
		return "An underscore can't be at the end of a number"
	case ErrConsecutiveSep:
		return "Numbers can't have consecutive underscores"
	case ErrMisplacedSep:
		return "An underscore isn't allowed here"
	case ErrNotAllowedInGuard:
		return "Case guards can't contain 'when' expressions"
	case ErrUnterminatedComment:
		return "The comment starting at " + e.Position.String() +
			" was left open"
	case ErrMisplacedShebang:
		return "Shebang must be on the first line of the file (without any lines or spaces before)"
	case ErrCannotTellStructOrEnum:
		return "Expected ':' for struct field, or '|' or '=' for enum member"
	case ErrEnumParamAndValue:
		return "Enum members can't have both parameters and a value"
	case ErrMissingFuncParamType:
		return "Function parameters must have a type"
	case ErrImportsGoFirst:
		return "Imports must go before other declarations"
	case ErrInvalidLabelShorthand:
		if e.Params["computed"] == true {
			return "A label shorthand can't be a computed property"
		}
		return "Only variables and properties can be used as label shorthands"
	case ErrMethodInOtherScope:
		return fmt.Sprintf(
			"Method %s must be declared in the same scope as type %s",
			e.Params["name"], e.Params["structName"],
		)
	case ErrInvalidVersionLit:
		return fmt.Sprintf("Invalid version literal '%s'",
			e.Node.(*ast.VersionLiteral).Version,
		)
	case ErrExpectedParamInLambda:
		return "Expected a parameter name in lambda"
	case ErrParenRequiredFunc:
		return "Parentheses are required around function parameter types"
	case ErrProvenUnreachable:
		return fmt.Sprintf("Unreachable statement after '%s'", e.Params["type"])
	case ErrReservedKeyword:
		return fmt.Sprintf(
			"Can't use %s as an identifier because it is a reserved keyword",
			QuoteToken(tok),
		)
	case ErrGenericInFuncAlias:
		return "Generic parameters aren't allowed in function aliases"
	case ErrUnderscoreWithRest:
		return "'_' not allowed with rest expression, use '...' instead"
	case ErrUnusedValue:
		return "TypeError: This value is never used"
	case ErrRedeclaredField:
		kind := "Field"
		if e.Params["kind"] == "enum" {
			kind = "Enum item"
		}
		return fmt.Sprintf("TypeError: %s %s was already declared", kind, QuoteToken(tok))
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
		return fmt.Sprintf("%s%s was already declared%s at %s",
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
		Range:     node.GetRange(),
		Position:  node.GetRange().Start,
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
