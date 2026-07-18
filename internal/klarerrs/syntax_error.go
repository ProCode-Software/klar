package klarerrs

import (
	"fmt"
	"reflect"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
)

const (
	_ Code = SyntaxErrorPrefix + iota

	ErrUnexpectedToken // Token not supposed to be there
	ErrExpectedToken   // Expected kind of token but got different type

	// Import =====

	ErrImportExpectedModule    // Unqualified import without module name
	ErrImportInvalidWildcard   // Wildcard must be last part of module
	ErrDotBeforeUnqualifiedImp // Use '.{' instead of '{' directly after import path
	ErrWildcardWithUnqualified // Using unqualified import with wildcard
	ErrEmptyUnqualifiedImport  // Empty unqualified import
	ErrImportsGoFirst          // Imports always go before other declarations

	// Punctuation =====

	ErrUnterminatedString    // A string that was left open
	ErrMultilineQuotedString // String quoted with " or ' contains newline
	ErrUnterminatedComment   // Block comment was left open
	ErrUnterminatedRegex     // Missing / in regex literal
	ErrMisplacedShebang      // Shebang not on first line
	ErrInvalidComma          // Comma statement
	ErrCurlyQuote            // Unicode curly quote used instead of ASCII straight quote
	ErrInvalidCharacter      // Invalid Unicode character
	ErrMisplacedBOM          // Byte order mark must be at the beginning of the file

	// Literal =====

	ErrStringEscape            // Invalid string escape
	ErrUnicodeEscapeTooBig     // Unicode escape over 0x10FFFF
	ErrNumberTooBig            // Numeric literal is too small or large
	ErrConsecutiveSeparator    // Number has consecutive _
	ErrMisplacedSeparator      // Number has separator somewhere where it's not supposed to
	ErrTrailingSeparator       // Number has misplaced _
	ErrExpectedHex             // Expected hex digit (0-9, a-f, A-F)
	ErrExpectedBinary          // Expected binary digit (0 or 1)
	ErrExpectedDecimal         // Expected decimal digit (0-9)
	ErrInvalidVersion          // Invalid version literal syntax
	ErrUnderscoreValue         // Use of _ as a value
	ErrEmptyRegexInterpolation // Empty regex interpolation
	ErrInvalidDecimalPoint     // Decimal point can only be used in decimal (base 10) format
	ErrIdentMustHaveLetter     // Identifiers must have at least 1 letter

	// Assignment =====

	ErrColonEqual            // := used instead of = in default value assignment
	ErrAssignmentAsExpr      // Assignment used as expression
	ErrEmptyDestructure      // Empty destructure target: (), #{}, or []
	ErrInvalidAssignment     // Assignment to non-variable or property
	ErrNonNameDeclaration    // Non-name on left-hand side of variable declaration
	ErrInvalidTypeAnnotation // Type annotation on existing variable assignment
	ErrDestructPatAfterColon // Non-identifier after : in destructure
	ErrDestructInvalidEqual  // Default value provided in non-object destructure
	ErrMismatchedAssignment  // Mismatched number of variables and values in assignment

	// Declaration =====

	ErrGenericInFuncAlias   // Function aliases can't have generics
	ErrSelfLabelInFuncAlias // Function aliases can't have a self label
	ErrMissingFuncParamType // Required function parameter type
	ErrNonNameFuncAlias     // Function alias target is not symbol or member
	ErrComputedFuncAlias    // Function alias target can't be a computed index
	ErrInvalidMethodAlias   // Method alias must be '.target'
	ErrInvalidPublic        // Public modifier applied to non-declaration
	ErrPublicGoesFirst      // Public modifier always goes first
	ErrDuplicateModifier    // More than 1 of the same modifier
	ErrFuncDotAfterSelf     // Expected . after (self: type). This is unlike Go
	ErrSelfNameDiscard      // Can't discard self name in method declaration
	ErrChainedDefault       // Default value specified with multiple keys
	ErrDiscardIntfField     // Interface field/method can't be '_'

	// Expression =====

	ErrReservedKeyword            // Reserved keyword used as an identifier
	ErrInvalidLabelShorthand      // Function label shorthand must be an identifier or string member
	ErrNumericLabel               // Function label can't be number
	ErrUnderscoreLabel            // Can't use _ as a label
	ErrReturnPipelineNotLast      // Return step in pipeline must be the last
	ErrInvalidObjectPipeStep      // Step in object pipeline must be method call or assignment
	ErrMultipleKeysInMapRest      // Expected 1 key in map rest (comma not allowed)
	ErrExpectedExprAfterOpenRange // Invalid: 1..<
	ErrEllipsisForOpenRangeStep   // ..< instead of ... in 1..<10...5
	ErrMustBeFuncCall             // Expression after go or try must be a function call
	ErrSelfExecFunc               // Self-executing functions are not allowed in Klar
	ErrParenAroundLambdaType      // Type for param is not in parentheses
	ErrParenAroundLambdaDefault   // Default value for param is not in parentheses
	ErrChainedNotEqual            // Can't use '!=' operator in chained comparison
	ErrMultiDirectionCompareChain // Inconsistent direction of operators in chain: e.g. < and >
	ErrStepInListSlice            // Step not allowed in list slice
	ErrExpectedInterpolationEnd   // Expected end of string/regex interpolation
	ErrInvalidForExprOperator     // Invalid or expected an operator in for expression

	// Type =====

	ErrExpectedTypeAssignment  // Need = or { after type (maybe got EOS)
	ErrRequiredStructFieldType // Struct fields need an explicit type
	ErrEmptyGeneric            // At least one parameter required in generic
	ErrParenFuncTypeParams     // Parentheses required for params: (Int) -> Int instead of Int -> Int
	ErrIntfDefaultValue        // Interface items can't have a default value
	ErrMixTypeTupleLabels      // Mix of 'label: type' and 'type' in type tuple
	ErrMissingLabelsType       // Labels don't have a type
	ErrIntfMultiKeyMethod      // Comma label syntax that includes a method: x, y, z()
	ErrInvalidGenericType      // Only enums can be generic (for now)
	ErrInvalidArrow            // -> can only be used with enum
	ErrRedeclaredField         // Struct or interface field redeclared
	ErrRedeclaredGeneric       // Generic parameter redeclared
	ErrNonNameInGeneric        // Left-hand side of generic type isn't a name
	ErrRedeclaredParamLabel    // Parameter label for enum item or function already declared

	// When =====

	ErrNoForIterator      // Expected assignment or expression in for loop
	ErrUnderscoreWithRest // ... instead of ..._ or _...
	ErrNotAllowedInWhen   // When expression not allowed in when case guard
	ErrRequiredBraces     // Required braces around statement in when case
	ErrWrongSubjectCount  //

	// Misc =====
	ErrTryBlock     // Klar doesn't have try-catch blocks
	ErrIfStatement  // Klar doesn't have if statements
	ErrTripleEqual  // JavaScript !== or === used in Klar
	ErrInvalidLoop  // Invalid loop kind in 'next' or 'stop' statement
	ErrPositiveSign // Leading '+' sign not allowed in Klar
	ErrDoubleNot    // Double '!!' not allowed in Klar

	// Analysis-time syntax errors =====

	ErrRedeclared             // Can't redeclare variable or function
	ErrTopLevel               // Multiple files in a module have top-level statements
	ErrMethodInOtherScope     // Method must be in the same scope as struct definition
	ErrAlwaysUnreachable      // Unreachable statement after return/stop/next/crashout
	ErrUnusedValue            // Unused expression statement
	ErrReturnOutsideFunc      // Return statement not allowed outside of function
	ErrImportShadow           // Import shadows top-level object
	ErrVarConstMixInDecl      // Var and const declared in the same declaration
	ErrMultipleVariadicParam  // More than 1 unlabelled variadic parameter defined
	ErrVariadicNotLast        // Variadic parameter must be the last param (if unlabelled)
	ErrDuplicateInheritedType // Inherited type specified twice
	ErrNoDeclAfterAttr        // Attribute must be followed by a declaration
	ErrMisplacedControlStmt   // Can't use 'stop'/'next' statement outside of 'when'/'for'/'while' loop
	ErrRedeclaredLoopLabel    // Loop label redeclared in the same function
	ErrRedeclaredOverload     // Overload redeclared with same params
	ErrVariadicDefault        // Variadic parameter can't a default value
	ErrBlockInWhenExpr        // Block bodies are only allowed in when statements
	ErrInvalidRestExpr        // Rest expression used outside of list, call, tuple, or map
	ErrReturnInPipelineExpr   // Return statement not allowed in pipeline expression
	ErrOptionalOptional       // Can't nest optional types (T??)
)

func (e *Error) handleSyntaxError() string {
	var (
		info, _ = e.Info.(SyntaxErrorInfo)
		tok     = info.Token
		kind    = tok.Kind
		src     = tok.Source
	)
	switch e.Code {
	default:
		title := "error code " + e.Code.String() + " doesn't have a message "
		if e.Node != nil {
			panic(title + "[node = " + reflect.TypeOf(e.Node).Name() + "]")
		}
		panic(title + "[token = " + tok.String() + "]")
	case ErrAssignmentAsExpr:
		return "An assignment can't be used as an expression in Klar"
	case ErrInvalidAssignment:
		return "You can only assign to a variable, field, list slice, or destructuring pattern"
		// Can't assign to this kind of expression
	case ErrInvalidComma:
		return "Expected an assignment, or a newline to separate multiple statements"
	case ErrUnderscoreValue:
		return "Can't use '_' as a value: '_' is only allowed as a name placeholder or as a discard in declarations"
	case ErrInvalidTypeAnnotation:
		return "A type annotation is only allowed on a new variable"
	case ErrExpectedToken:
		expToken := e.TokenTypeParam("expected")
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
		if msg := e.StringParam("msg"); msg != "" {
			return fmt.Sprintf("Expected %s %s", expected, msg)
		}
		return fmt.Sprintf("I expected %s, but found %s instead", expected, NameToken(tok))
	case ErrWildcardWithUnqualified:
		return "Can't have both '*' and unqualified import in import statement"
	case ErrEmptyUnqualifiedImport:
		return "Expected at least 1 unqualified import"
	case ErrImportExpectedModule:
		return "I expected a module name before '.{' in unqualified import"
	case ErrImportInvalidWildcard:
		return "'*' should be at the end of the module name"
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
		return fmt.Sprintf(
			"The string starting at %s was left open",
			e.Params["start"].(lexer.Position),
		)
	case ErrMultilineQuotedString:
		return "Only strings quoted with backticks '`' can contain line breaks"
	case ErrUnterminatedRegex:
		return fmt.Sprintf(
			"The regular expression starting at %s was left open",
			e.Params["start"].(lexer.Position),
		)
	case ErrExpectedTypeAssignment:
		if kind == lexer.Newline {
			return "A type must be assigned a value"
		}
		return "I expected '{' or '=' after type, but found " + NameToken(tok) + " instead"
	case ErrRequiredStructFieldType:
		return "A struct field must have an explicit type"
	case ErrMustBeFuncCall:
		return "The expression after '" + e.TokenTypeParam("expr").String() +
			"' must be a function call"
	case ErrExpectedHex:
		return "I expected a hexadecimal digit (0-9, a-f or A-F)"
	case ErrExpectedBinary:
		return "I expected a binary digit (0-1)"
	case ErrExpectedDecimal:
		return "I expected a decimal digit (0-9)"
	case ErrUnicodeEscapeTooBig:
		return "This Unicode escape must be in the range 0 to 10FFFF"
	case ErrStringEscape:
		kind := e.Params["type"].(lexer.EscapeType)
		switch code := e.Params["reason"].(lexer.EscapeErrorCode); code {
		case lexer.ErrEscapeExpHex:
			if kind == lexer.EscHex {
				return "I expected 2 hexadecimal digits (0-9, a-f or A-F) " + `after '\x'`
			}
			// Unicode
			return "I expected 1-6 hexadecimal digits (0-9, a-f or A-F) between { }"
		case lexer.ErrCharEscapeUnknown:
			esc := e.StringParam("escape")
			return "Unknown character escape " + Quote(esc)
		case lexer.ErrEscapeTooShort:
			if kind == lexer.EscInterpolation {
				return "A string interpolation can't be empty"
			}
			fallthrough
		case lexer.ErrUnicodeEscapeTooLong:
			return "I expected 1-6 hex digits between { } in Unicode escape"
		case lexer.ErrEscapeUnterm:
			k := "escape"
			if kind == lexer.EscInterpolation {
				k = "interpolation"
			}
			return "Expected '}' to end string " + k
		default:
			panic(fmt.Sprintf("unknown EscapeErrorCode: %d", code))
			// return "Invalid string escape"
		}
	case ErrCurlyQuote:
		alt := e.StringParam("alt")
		return "Use the straight quotation mark " + Quote(alt) + " instead of a curly quotation mark"
	case ErrNoForIterator:
		return "Missing variables or expression in 'for' loop"
	case ErrEmptyGeneric:
		return "At least 1 type is required inside '<...>'"
	case ErrRedeclaredGeneric:
		return "Generic " + Quote(e.Name) + " was already declared"
	case ErrInvalidPublic:
		return "Expected a declaration after 'public' modifier"
	case ErrTrailingSeparator:
		return "An underscore can't be at the end of a number"
	case ErrConsecutiveSeparator:
		return "A number can't contain consecutive underscores"
	case ErrMisplacedSeparator:
		return "An underscore must separate successive digits"
	case ErrInvalidDecimalPoint:
		return "A decimal can only be used in base 10 numbers"
	case ErrNotAllowedInWhen:
		// location: pattern or guard
		return "A 'when' " + e.StringParam("location") + " can't contain " + e.StringParam("expr")
	case ErrUnterminatedComment:
		return "The comment starting at " + e.Range.Start.String() + " was left open"
	case ErrMisplacedShebang:
		return "A shebang must be on the first line of the file (without any lines or spaces before)"
	case ErrMissingFuncParamType:
		return "A function parameter must have an explicit type"
	case ErrImportsGoFirst:
		return "'import' statements must go before other statements in the file"
	case ErrNumericLabel:
		return "A number can't be used as a parameter label"
	case ErrUnderscoreLabel:
		return "Can't use _ as a parameter label"
	case ErrChainedDefault:
		e.Hint("If you're trying to assign a default value to the last parameter, separate the parameter from the other chained parameters.")
		return "A default value can't be specified with chained variables"
	case ErrInvalidLabelShorthand:
		if e.BoolParam("computed") {
			return "A parameter label shorthand can't be a computed property"
		}
		return "Only a variable or property can be used as a label shorthand"
	case ErrMethodInOtherScope:
		if e.BoolParam("initializer") {
			return "An initializer must be declared in the same scope as the type"
		}
		return "A method must be declared in the same scope as its self type"
	case ErrInvalidVersion:
		return fmt.Sprintf(
			"'%s' isn't a valid version", e.Node.(*ast.VersionLiteral).Version,
		)
	case ErrSelfExecFunc:
		return "Self-executing functions aren't allowed in Klar"
	case ErrParenFuncTypeParams:
		return "Parentheses are required around function parameter types"
	case ErrIdentMustHaveLetter:
		return "An identifier must contain at least 1 letter"
	case ErrInvalidObjectPipeStep:
		return "A object pipeline step must be an assignment or method call"
	case ErrInvalidGenericType:
		return "Only enums can have generic parameters"
	case ErrAlwaysUnreachable:
		kind := e.StringParam("kind")
		return "Everything after " + kind + " is always unreachable"
	case ErrReservedKeyword:
		return fmt.Sprintf(
			"Can't use %s as an identifier because it is a reserved keyword",
			QuoteToken(tok),
		)
	case ErrDestructInvalidEqual:
		return "A default value can only be provided in a map destructure pattern"
	case ErrDuplicateModifier:
		return "Can't use the " + FormatTokenType(e.TokenTypeParam("modifier")) +
			" modifier more than once in this declaration"
	case ErrGenericInFuncAlias:
		return "Generic parameters aren't allowed in function aliases"
	case ErrUnderscoreWithRest:
		return "Don't use '_' with a rest; use just '...' instead"
	case ErrReturnPipelineNotLast:
		return "The 'return' in a pipeline must be the last step"
	case ErrPublicGoesFirst:
		return "'public' must be the first modifier"
	case ErrEmptyDestructure:
		return "A destructure pattern can't be empty"
	case ErrColonEqual:
		return "Use '=' instead of ':=' to set a default"
	case ErrEllipsisForOpenRangeStep:
		return "Use '...' instead of '..<' to define a range step"
	case ErrExpectedExprAfterOpenRange:
		return "I expected an expression after '..<'"
	case ErrRequiredBraces:
		return "Braces are required around this statement"
	case ErrDestructPatAfterColon:
		return "Only an identifier is allowed after ':' in object destructure"
	case ErrMultipleKeysInMapRest:
		return "A spread must be its own key"
	case ErrNonNameDeclaration:
		return "Expression to the left of ':=' must be a name"
	case ErrMixTypeTupleLabels:
		return "Can't mix 'label: type' and 'type' syntax in tuple or parameters"
	case ErrNonNameFuncAlias:
		return "The right-hand side of a function alias must be a function or method name"
	case ErrIntfDefaultValue:
		return "An interface field can't have a default value"
	case ErrIntfMultiKeyMethod:
		return "Function declarations cannot appear in comma-separated keys; split the function into its own entry"
	case ErrMismatchedAssignment:
		exp, got := e.IntParam("left"), e.IntParam("right")
		detail := fmt.Sprintf("left has %d, but right has %d", exp, got)
		if got < exp {
			return "Not enough values on the right-hand side of this assignment: " + detail
		}
		return "Too many values on the right-hand side of this assignment: " + detail
	case ErrFuncDotAfterSelf:
		return "Expected a '.' between ')' and the name in function declaration"
	case ErrMultiDirectionCompareChain:
		return "'<'/'<=' and '>'/'>=' can't be mixed in a single comparison chain: they must follow the same direction"
	case ErrChainedNotEqual:
		return "The '!=' operator isn't allowed to be chained in a comparison chain"
	case ErrStepInListSlice:
		e.Hint("A step requires the entire list to be iterated over and copied, defeating the purpose of slicing. Instead, manually iterate over the list.")
		return "A step is not allowed in the range of a list slice"
	case ErrExpectedInterpolationEnd:
		kind := "string"
		if e.BoolParam("regex") {
			kind = "regex"
		}
		return "I expected '}' here to end the " + kind + " interpolation"
	case ErrIfStatement:
		return "Klar doesn't have if statements; use 'when' instead"
	case ErrTryBlock:
		return "Klar doesn't have try-catch statements"
	case ErrTripleEqual:
		op := FormatTokenType(e.TokenTypeParam("op"))
		return "In Klar, comparisons are always strict; use just " + op + " instead"
	case ErrSelfNameDiscard:
		e.Hint("Remove the label")
		return "Can't use '_' as name of self in method declaration"
	case ErrDotBeforeUnqualifiedImp:
		return "I expected '.' before unqualified imports"
	case ErrInvalidLoop:
		kind := e.TokenTypeParam("stmt")
		var loop string
		if kind == lexer.Next {
			loop = "continue"
		} else {
			loop = "stop"
		}
		return "You can only " + loop + " a for, when, or while loop"
	case ErrParenAroundLambdaDefault:
		return "Parameters must be in parentheses in order to set default values"
	case ErrParenAroundLambdaType:
		return "Parameters must be in parentheses in order to annotate types"
	case ErrInvalidArrow:
		return "'->' can only be used in an enum declaration"
	case ErrUnusedValue:
		return "This value is never used"
	case ErrDiscardIntfField:
		e.Hint("Remove the field")
		return "An interface field can't be '_'"
	case ErrComputedFuncAlias:
		return "The target of a function alias can't be computed"
	case ErrInvalidCharacter:
		return "This isn't a valid Unicode character"
	case ErrEmptyRegexInterpolation:
		return "A regex interpolation can't be empty"
	case ErrPositiveSign:
		e.Hint(
			"A leading '+' sign doesn't affect a number's value. Remove it.\n" +
				"To convert a number to an integer or float, use the 'Int()' or 'Float()' function.",
		)
		return "A '+' prefix isn't allowed in Klar"
	case ErrDoubleNot:
		if e.IntParam("count")%2 == 0 {
			e.Hint("Remove all of them.")
		} else {
			e.Hint("Keep only one of them.")
		}
		return "Multiple '!' are not allowed"
	case ErrInvalidForExprOperator:
		return "Expected '->', an arithmetic assignment, or block in 'for' expression"
	case ErrMisplacedBOM:
		return "Byte order mark must be the first character in the file"
	case ErrSelfLabelInFuncAlias:
		return "Function aliases can't have a named self"
	case ErrMissingLabelsType:
		if e.IntParam("length") == 1 {
			return "Missing type for this label"
		}
		return "Missing type for these labels"
	case ErrRedeclared:
		var (
			oldKind = e.StringParam("oldKind")
			newKind = e.StringParam("newKind")
			as      string
		)
		if oldKind != newKind {
			as = " as a " + oldKind
		}
		return Quote(e.Name) + " was already declared" + as
	case ErrTopLevel:
		return "Only 'main.klar' and single-file modules can have top-level statements"
	case ErrImportShadow:
		name := e.StringParam("name")
		importPath := e.StringParam("import")
		if importPath != "" {
			importPath = " from " + Quote(importPath)
		}
		return "The import " + Quote(name) + importPath +
			" has the same name as an existing object in this module"
	case ErrRedeclaredField:
		name := e.Node.(ast.Identifier).Name
		kind := e.StringParam("kind")
		if kind == "enum" {
			return "Item " + Quote(name) + " was already declared in this enum"
		}
		return "The field " + Quote(name) + " was already declared in this " + kind
	case ErrVarConstMixInDecl:
		return "Can't declare variable and constants in the same statement"
	case ErrDuplicateInheritedType:
		name := e.StringParam("name")
		return "Type " + Quote(name) + " was already inherited"
	case ErrNoDeclAfterAttr:
		return "Attributes must be followed by a declaration"
	case ErrMisplacedControlStmt:
		kind := "next"
		if _, ok := e.Node.(*ast.StopStatement); ok {
			kind = "stop"
		}
		return "A " + Quote(kind) +
			" statement can only be used within a 'for', 'when', or 'while' statement"
	case ErrRedeclaredOverload:
		return "Overload " + Quote(e.Name) + " was already declared"
	case ErrVariadicDefault:
		return "A variadic parameter can't have a default value"
	case ErrNonNameInGeneric:
		return "The left-hand side of a generic type reference must be a type name"
	case ErrNumberTooBig:
		return "The number " + Quote(e.Name) + " is too large"
	/* case ErrRedeclaredParamLabel:
	return "" */
	case ErrReturnOutsideFunc:
		return "A 'return' statement is only allowed within a function body"
	case ErrBlockInWhenExpr:
		return "A block can only be used as the body of a 'when' case in a 'when' statement, not expression"
	case ErrVariadicNotLast:
		return "The variadic parameter must be the last parameter in the function"
	case ErrOptionalOptional:
		return "An optional type can't be optional"
	case ErrWrongSubjectCount:
		exp, got := e.IntParam("expected"), e.IntParam("got")
		title := "Too many subjects in this case"
		if got < exp {
			title = "Not enough subjects in this case"
		}
		return fmt.Sprintf(
			"%s: Expected %s, but this case has %d",
			title, FormatCount(exp, "subject"), got,
		)
	case ErrInvalidMethodAlias:
		return "The right-hand side of a method alias must be in the format '.method'"
	}
}
