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

	ErrUnexpectedToken // Token not supposed to be there
	ErrExpectedToken   // Expected kind of token but got different type

	// Import =====

	ErrAliasInUnqualifiedImport // Alias is not allowed before unqualified import
	ErrImportExpectedModule     // Unqualified import without module name
	ErrImportInvalidWildcard    // Wildcard must be last part of module
	ErrImportTooManyWildcard    // More than 1 wildcard
	ErrWildcardAndUnqImport     // Using unqualified import with wildcard
	ErrWildcardAndAlias         // Using alias with wildcard
	ErrEmptyUnqImport           // Empty unqualified import
	ErrImportsGoFirst           // Imports always go before other declarations

	// Punctuation =====

	ErrUnterminatedString  // A string that was left open
	ErrUnterminatedComment // Block comment was left open
	ErrUnterminatedRegex   // Missing / in regex literal
	ErrMisplacedShebang    // Shebang not on first line
	ErrInvalidComma        // Comma statement

	// Literal =====

	ErrStringEscape        // Invalid string escape
	ErrUnicodeEscTooBig    // Unicode escape over 0x10FFFF
	ErrConsecutiveSep      // Number has consecutive _
	ErrMisplacedSep        // Number has separator somewhere where it's not supposed to
	ErrTrailingSep         // Number has misplaced _
	ErrExpectedHex         // Expected hex digit (0-9, a-f, A-F)
	ErrExpectedOctal       // Expected octal digit (0-7)
	ErrExpectedBinary      // Expected binary digit (0 or 1)
	ErrExpectedDecimal     // Expected decimal digit (0-9)
	ErrInvalidLambdaParams // Non-variable or variable tuple used in lambda
	ErrInvalidVersionLit   // Invalid version literal syntax
	ErrUnderscoreValue     // Use of _ as a value

	// Assignment =====

	ErrInvalidUpdate         // ++ or -- used as an expression or prefix form
	ErrColonEqual            // := used instead of = in default value assignment
	ErrAssignmentAsExpr      // Assignment used as expression
	ErrEmptyDestructure      // Empty destructure target: (), #{}, or []
	ErrInvalidAssignment     // Assignment to non-variable or property
	ErrNonNameDeclaration    // Non-name on left-hand side of variable declaration
	ErrInvalidTypeAnnotation // Type annotation on existing variable assignment
	ErrDestructPatAfterColon // Non-identifier after : in destructure
	ErrDestructInvalidEqual  // Default value provided in non-object destructure

	// Declaration =====

	ErrGenericInFuncAlias   // Function aliases can't have generics
	ErrMissingFuncParamType // Required function parameter type
	ErrNonNameFuncAlias     // Function alias target is not symbol or member
	ErrInvalidOpaque        // Opaque on non-struct or interface
	ErrInvalidPublic        // Public modifier applied to non-declaration
	ErrPublicFirst          // Public modifier always goes first
	ErrDuplicateModifier    // More than 1 of the same modifier
	ErrFuncDotAfterSelf     // Expected . after (self: type). This is unlike Go

	// Expression =====

	ErrReservedKeyword              // Reserved keyword used as an identifier
	ErrNotAnExpression              // Required expression but got a statement
	ErrInvalidLabelShorthand        // Function label shorthand must be an identifier or string member
	ErrInvalidLabel                 // Function label can't be number
	ErrReturnPipelineNotLast        // Return step in pipeline must be the last
	ErrInvalidObjPipeStep           // Step in object pipeline must be method call or assignment
	ErrMultipleKeysInMapRest        // Expected 1 key in map rest (comma not allowed)
	ErrExpectedExprAfterClosedRange // Invalid: 1..<
	ErrEllipsisForClosedRange       // ..< instead of ... in 1..<10...5
	ErrMustBeFuncCall               // Expression after go or try must be a function call
	ErrSelfExecFuncNotAllowed       // Self-executing functions are not allowed in Klar

	// Type =====

	ErrExpectedTypeAssignment  // Need = or { after type (maybe got EOS)
	ErrRequiredStructFieldType // Struct fields need an explicit type
	ErrEmptyGeneric            // At least one parameter requried in generic
	ErrParenRequiredFunc       // Parentheses required for params: (Int) -> Int instead of Int -> Int
	ErrInterfaceDefaultValue   // Interface items can't have a default value
	ErrMixTypeTupleLabels      // Mix of 'label: type' and 'type' in type tuple
	ErrIntfMultiKeyMethod      // Comma label syntax that includes a method: x, y, z()

	// When =====

	ErrForInvalidCond     // Expected assignment or expression in for loop
	ErrUnderscoreWithRest // ... instead of ..._ or _...
	ErrNotAllowedInWhen   // When expression not allowed in when case guard
	ErrBraceAroundStmt    // Required braces around statement in when case

	// Misc =====
	ErrTryBlock    // Klar doesn't have try-catch blocks
	ErrIfStatement // Klar doesn't have if statements

	// Analysis-time syntax errors =====

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
	ErrorCode ErrorCode
	Token     lexer.Token
	Node      ast.Node
	Hints     []string
	Details   []Detail
	File      string
	Params    map[string]any
}

func (e *ParseError) SetParam(key string, value any) ParseError {
	if e.Params == nil {
		e.Params = make(ErrorParams, 1)
	}
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
	case ErrNotAnExpression:
		switch e.Node.(type) {
		case *ast.UpdateStatement:
			return "'++' and '--' can only be used as postfix statements"
		case *ast.AssignmentStatement, *ast.VariableDeclaration:
			return "An assignment can't be used as an expression in Klar"
		}
		return "This isn't an expression"
	case ErrAssignmentAsExpr:
		return "An assignment can't be used as an expression in Klar"
	case ErrInvalidAssignment:
		return "Can't assign to this kind of expression"
	case ErrInvalidComma:
		return "A newline must be used to separate multiple statements"
	case ErrUnderscoreValue:
		return "Can't use '_' as a value: '_' is only allowed as a name placeholder or as a discard in declarations"
	case ErrInvalidTypeAnnotation:
		return "A type annotation is only allowed on a new variable"
	case ErrExpectedToken:
		expToken := e.Params["expected"].(lexer.TokenType)
		expected := FormatTokenType(expToken)
		if src == ";" {
			return "A line break must be used to terminate a statement in Klar"
		}
		endTypeMap := map[lexer.TokenType]string{
			lexer.RightCurlyBrace:  "brace",
			lexer.RightParenthesis: "parenthesis",
			lexer.GreaterThan:      "angle bracket",
			lexer.RightBracket:     "bracket",
		}
		if endType, ok := endTypeMap[expToken]; ok {
			return fmt.Sprintf("Missing closing %s %s", endType, expected)
		}
		return fmt.Sprintf("I expected %s, but found %s instead", expected, NameToken(tok))
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
		return "Can't use an alias with an unqualified import"
	case ErrUnexpectedToken:
		switch {
		case src == ";":
			return "A line break must be used to terminate a statement in Klar"
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
			return "A type must be assigned a value"
		}
		return "I expected '{' or '=' after type, but found " + NameToken(tok) + " instead"
	case ErrRequiredStructFieldType:
		return "A struct field must have an explicit type"
	case ErrMustBeFuncCall:
		return "The expression after 'go' or 'try' must be a function call"
	case ErrExpectedHex:
		return "I expected 2 hexadecimal digits (0-9, a-f or A-F) after"
	case ErrExpectedBinary:
		return "I expected a binary digit (0-1)"
	case ErrExpectedOctal:
		return "I expected an octal digit (0-7)"
	case ErrExpectedDecimal:
		return "I expected a decimal digit (0-9)"
	case ErrUnicodeEscTooBig:
		return "This Unicode escape must be in the range 0 to 10FFFF"
	case ErrStringEscape:
		reason := e.Params["reason"].(lexer.EscapeError)
		kind := e.Params["type"].(lexer.EscapeType)
		switch reason {
		case lexer.ErrEscapeExpHex:
			return `I expected 2 hexadecimal digits (0-9, a-f or A-F) after '\x'`
		case lexer.ErrEscapeUnknown:
			esc := e.Params["escape"].(string)
			return "Unknown character escape " + Quote(esc)
		case lexer.ErrEscapeTooLong, lexer.ErrEscapeTooShort:
			if kind == lexer.EscUnicode {
				return "I expected 1-6 hex digits between { } in Unicode escape"
			}
			return "I expected an expression"
		default:
			return "Invalid string escape"
		}
	case ErrForInvalidCond:
		return "Expected an assignment or expression in for condition"
	case ErrEmptyGeneric:
		return "At least 1 type is required inside < >"
	case ErrInvalidPublic:
		return "Expected a declaration after public modifier"
	case ErrTrailingSep:
		return "An underscore can't be at the end of a number"
	case ErrConsecutiveSep:
		return "A number can't contain consecutive underscores"
	case ErrMisplacedSep:
		return "An underscore must separate successive digits"
	case ErrNotAllowedInWhen:
		return "A 'when' case can't contain 'when' expressions or lambdas"
	case ErrUnterminatedComment:
		return "The comment starting at " + e.Position.String() + " was left open"
	case ErrMisplacedShebang:
		return "A shebang must be on the first line of the file (without any lines or spaces before)"
	case ErrMissingFuncParamType:
		return "A function parameter must have an explicit type"
	case ErrInvalidOpaque:
		return "'opaque' modifier can only be applied to a struct or interface"
	case ErrImportsGoFirst:
		return "Imports must go before other declarations"
	case ErrInvalidLabel:
		return "A number can't be used as a parameter label"
	case ErrInvalidLabelShorthand:
		if e.Params["computed"] == true {
			return "A parameter label shorthand can't be a computed property"
		}
		return "Only a variable or property can be used as a label shorthand"
	case ErrMethodInOtherScope:
		return fmt.Sprintf(
			"Method %s must be declared in the same scope as type %s",
			e.Params["name"], e.Params["structName"],
		)
	case ErrInvalidVersionLit:
		return fmt.Sprintf("Invalid version literal '%s'",
			e.Node.(*ast.VersionLiteral).Version,
		)
	case ErrSelfExecFuncNotAllowed:
		return "Self-executing functions are not allowed in Klar"
	case ErrInvalidLambdaParams:
		return "Invalid parameter list before '->' in lambda"
	case ErrParenRequiredFunc:
		return "Parentheses are required around function parameter types"
	case ErrInvalidObjPipeStep:
		return "A object pipeline step must be an assignment or method call"
	case ErrProvenUnreachable:
		return fmt.Sprintf("Unreachable statement after '%s'", e.Params["type"])
	case ErrReservedKeyword:
		return fmt.Sprintf(
			"Can't use %s as an identifier because it is a reserved keyword",
			QuoteToken(tok),
		)
	case ErrDestructInvalidEqual:
		return "A default value can only be provided in a map destructure pattern"
	case ErrDuplicateModifier:
		modif := param[lexer.TokenType](e.Params, "modifier")
		return fmt.Sprintf("Modifer %s was already specified in this declaration",
			FormatTokenType(modif),
		)
	case ErrGenericInFuncAlias:
		return "Generic parameters aren't allowed in function aliases"
	case ErrUnderscoreWithRest:
		return "'_' not allowed with rest expression, use '...' instead"
	case ErrReturnOutsideFunc:
		return "Can't use return statement outside of a function"
	case ErrInvalidUpdate:
		return "'++' and '--' can only be used as a postfix statement"
	case ErrReturnPipelineNotLast:
		return "'return' in pipeline must be the last step"
	case ErrPublicFirst:
		return "'public' must be the first modifier"
	case ErrEmptyDestructure:
		return "A destructure pattern can't be empty"
	case ErrColonEqual:
		return "Expected '=' instead of ':='"
	case ErrEllipsisForClosedRange:
		return "Expected '...' instead of '..<'"
	case ErrExpectedExprAfterClosedRange:
		return "I expected an expression after '..<'"
	case ErrBraceAroundStmt:
		return "Braces are required around this statement"
	case ErrDestructPatAfterColon:
		return "Only an identifier is allowed after ':' in object destructure"
	case ErrMultipleKeysInMapRest:
		return "Expected a single key in map spread"
	case ErrNonNameDeclaration:
		return "Only names and destructure patterns are allowed on the left-hand side of a variable declaration"
	case ErrMixTypeTupleLabels:
		return "Can't mix 'label: type' and 'type' syntax in tuple or parameters"
	case ErrNonNameFuncAlias:
		return "Invalid function alias: target must be a function name"
	case ErrInterfaceDefaultValue:
		return "An interface field can't have a default value"
	case ErrIntfMultiKeyMethod:
		return "Function declarations cannot appear in comma-separated keys; split the function into its own entry"
	case ErrFuncDotAfterSelf:
		return "Expected '.' between ')' and identifier in function declaration"
	case ErrUnusedValue:
		return "This value is never used"
	case ErrRedeclaredField:
		kind := "Field"
		if e.Params["kind"] == "enum" {
			kind = "Enum item"
		}
		return fmt.Sprintf("TypeError: %s '%s' was already declared",
			kind, e.Node.(ast.Identifier).Name,
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
		Position:  ranges.Sub(*e.ErrorPosition, 0, 1),
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

func Slice[T ast.Node](err ErrorCode, nodes []T) ParseError {
	start := nodes[0].GetRange().Start
	return ParseError{
		ErrorCode: err,
		Range: ranges.Range{
			Start: start,
			End:   nodes[len(nodes)-1].GetRange().End,
		},
		Position: start,
	}
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
