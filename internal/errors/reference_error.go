package errors

const (
	_ ErrorCode = ReferenceErrorPrefix + iota

	ErrVarUndefined  // Variable doesn't exist
	ErrEnumUndefined // Enum item doesn't exist
	ErrTypeUndefined // Type doesn't exist
)
