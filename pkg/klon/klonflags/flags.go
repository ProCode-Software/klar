package klonflags

type Flags uint32

const (
	// Marshalling

	NoSortFields       Flags = 1 << iota // Don't alphabetically sort fields (order is random for maps)
	InsertFinalNewline                   // Add newline at end of output
	PreserveFieldCase                    // Don't convert field names to camel case

	// Unmarshalling

	NoUnknownFields     // Error on unknown field when unmarshalling
	NoVariables         // Don't resolve variables
	CaseSensitiveFields // Field names must be same case as given in klon struct tag or camel case
	ClampNumbers        // Out of range numbers are clamped or truncated
	AllowJSONStructTags // Use json: struct tags if klon: doesn't exist
	NoSingleItemToArray // Don't put single values into arrays
	KeyedEmbeddedFields // If a struct has an embedded field, it may be keyed in the source
	IgnoreArrayLength   // Don't validate array lengths; skip remaining items
	NoMissingFields     // Error on missing field when unmarshalling
	ZeroNullValues      // Zero Go values if the AST value is `none`

	// Unmarshalling to any

	UseFloat64         // All numbers are decoded as float64
	UseInt64           // Integers are decoded as int64
	UseInt             // Integers are decoded as int
	UseByteSlice       // Strings are decoded as []byte
	UseRuneSlice       // Same as [UseByteSlice] but decodes as []rune
	BoolIsString       // true and false literals are strings
	NumberIsString     // Numeric literals are strings
	EmptyValueIsString // Empty markup values are decoded as "
	OmitNullFields     // Don't include fields set to `none`

	StrictFields = NoUnknownFields | CaseSensitiveFields |
		NoSingleItemToArray | NoMissingFields
	AllLiteralsAreStrings = BoolIsString | NumberIsString | EmptyValueIsString
)

func (f Flags) Has(flag Flags) bool { return (f & flag) != 0 }
