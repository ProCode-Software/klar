package klarml

type Flags uint32

const (
	// Marshalling

	NoSortFields       Flags = 1 << iota // Don't alphabetically sort fields: order is random
	InsertFinalNewline                   // Add newline at end of output

	// Unmarshalling

	NoUnknownFields       // Error on unknown field when unmarshalling
	NoVariables           // Don't resolve variables or namespaces
	CaseSensitiveFields   // Field names must be same case as given in klarml struct tag
	UseFloat64            // If type any, all numbers are decoded as float64
	UseInt64              // If type any, integers are decoded as int64
	UseByteArray          // Strings are decoded as []byte if type is any
	UseRuneArray          // Same as UseByteArray but []rune
	BoolsAreStrings       // If type any, true and false literals are strings
	NumbersAreString      // If type any, numeric literals are strings
	EmptyValuesAreStrings // Empty markup values are decoded as "" if type any

	StrictFields                  = NoUnknownFields | CaseSensitiveFields
	AllLiteralsAreStrings         = AllNonEmptyLiteralsAreStrings | EmptyValuesAreStrings
	AllNonEmptyLiteralsAreStrings = BoolsAreStrings | NumbersAreString
)

func (f Flags) Has(flag Flags) bool {
	return f&flag != 0
}
