package errors

const (
	_ ErrorCode = ReferenceErrorPrefix + iota

	ErrVarExists    // Can't redeclare variable
	ErrVarUndefined // Variable doesn't exist
)
