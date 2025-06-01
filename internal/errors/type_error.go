package errors

const (
	_ ErrorCode = TypeErrorPrefix + iota

	ErrUntypedNil        // nil requires contextual type
	ErrUntypedEmptyList  // Can't infer type from empty list
	ErrAssignToConst     // Attempted reassignment to constant reference
	ErrUncheckedOptional // Required to check if optional is nil
	ErrUncheckedResult   // Required to check Result for error
	ErrUnusedLiteral     // Unused literal expression statement
)
