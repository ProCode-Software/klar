package flags

type Flags uint32

const (
	// Marshalling

	NoSortFields       Flags = 1 << iota // Don't alphabetically sort fields: order is random
	InsertFinalNewline                   // Add newline at end of output

	// Unmarshalling

	NoUnknownFields     // Error on unknown field when unmarshalling
	NoVariables         // Don't resolve variables or namespaces
	CaseSensitiveFields // Field names must be same case as given in klarml struct tag
	ClampNumbers        // Out of range numbers are clamped or truncated
	AllowJSONStructTags // Use json: struct tags if klarml: doesn't exist
	ValidateUTF8        // Validate UTF-8 strings
	NoSingleItemToArray // Don't put single values into arrays
	KeyedEmbeddedFields // If a struct has an embedded field, it may be keyed in the source or not
	IgnoreArrayLength   // Don't validate array lengths; skip remaining items

	// Unmarshalling to any

	UseFloat64         // All numbers are decoded as float64
	UseInt64           // Integers are decoded as int64
	UseByteArray       // Strings are decoded as []byte
	UseRuneArray       // Same as UseByteArray but []rune
	BoolIsString       // true and false literals are strings
	NumberIsString     // Numeric literals are strings
	EmptyValueIsString // Empty markup values are decoded as ""

	StrictFields          = NoUnknownFields | CaseSensitiveFields | NoSingleItemToArray
	AllLiteralsAreStrings = BoolIsString | NumberIsString | EmptyValueIsString
)

func (f Flags) Has(flag Flags) bool {
	return (f & flag) != 0
}

func Parse(flags ...Flags) Flags {
	var f Flags
	for _, flag := range flags {
		f |= flag
	}
	return f
}
