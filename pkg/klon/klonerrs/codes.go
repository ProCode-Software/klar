package klonerrs

type Code int

const (
	_ Code = iota

	ErrUnexpectedToken // Token not supposed to be there
	ErrExpectedToken   // Expected kind of token but got different type

	// Punctuation =====

	ErrUnterminatedString  // A string that was left open
	ErrUnterminatedList    // A list that was left open
	ErrUnterminatedObject  // An object that was left open
	ErrUnterminatedVar     // A variable reference that was left open
	ErrUnterminatedComment // A block comment that was left open
	ErrExpectedCurlyInVar  // Missing '{' in variable reference
	ErrUnmatchedBracket    // Closing bracket without an opening one
	ErrIllegalCharacter    // Invalid Unicode character
	ErrExpectedEOF         // Content found after what should be the end of the document

	// Literal =====

	ErrInvalidIdentifier // Variable name starting with a digit
	ErrNegativeNumber    // Unexpected negative number
	ErrTruncatedNumber   // Float value where an integer was expected
	ErrUnknownEscape     // Invalid backslash escape sequence
	ErrInvalidKey        // Non-string/number/bool used as a key
	ErrExpectedValue     // Expected a value but found something else
	ErrExpectedClassName // Missing or invalid class name after '@'
	ErrExpectedKeyValue  // Expected a key-value pair
	ErrDuplicateField    // Field already defined in the object
	ErrMisplacedRest     // Rest found outside of an Object or List

	// Structural =====

	ErrDashWithoutNewline // Dash for nesting not preceded by a newline
	ErrDashAtTopLevel     // Dash found at the beginning of the document
	ErrDashSkip           // Dash depth increased by more than 1
	ErrMaxDepth           // Nesting depth exceeded MaxDepth

	// Variables =====

	ErrUndefinedVar       // Use of an undefined variable
	ErrVarNotTopLevel     // Variable declaration outside of top level
	ErrInvalidVarDecl     // Variable declaration using braces
	ErrExpectedVarInArrow // Missing variable after '<-'
	ErrVarAlreadyDeclared // Variable is already declared
	ErrVarCycle           // Variable cycle

	// Decode =====

	ErrTypeMismatch        // Mixed keyed and unkeyed entries in a block
	ErrWrongArrayLength    // Mismatched array length during decoding
	ErrUnsupportedValue    // Value type that cannot be decoded into the target Go type
	ErrInvalidEnumOption   // Invalid enum option
	ErrInvalidRest         // Rest must be an object or list (depending on context)
	ErrFieldNotFound       // Field not found in the target struct
	ErrVarsDisabled        // Variables can't be resolved due to [klonflags.NoVariables]
	ErrCantConvertToString // Can't convert value to string
)
