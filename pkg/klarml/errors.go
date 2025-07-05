package klarml

import "fmt"

type UntermStringErr struct {
	Quote byte
	Token Token
}
type ExpectedTokenErr struct {
	Expected TokenType
	Got      Token
}
type MixPropAndArrayErr struct {
	Position   Position
	AlreadyObj bool
}
type (
	UntermCommentErr   struct{ Token Token }
	UnexpectedTokenErr struct{ Token Token }
)

func pos(p Position) string {
	return fmt.Sprintf("line %d, col %d", p.Line, p.Col)
}

func (err UnexpectedTokenErr) Error() string {
	return fmt.Sprintf("unexpected '%s' at %s",
		err.Token.Source, pos(err.Token.Position),
	)
}

func (err UntermStringErr) Error() string {
	return fmt.Sprintf("expected `%c` to end string literal starting at %s",
		err.Quote, pos(err.Token.Position),
	)
}

func (err UntermCommentErr) Error() string {
	return fmt.Sprintf("expected '*/' to end block comment starting at %s",
		pos(err.Token.Position),
	)
}

func (err MixPropAndArrayErr) Error() string {
	line := err.Position.Line
	if err.AlreadyObj {
		return fmt.Sprintf("expected key for property on line %d", line)
	}
	return fmt.Sprintf("key not allowed for array item on line %d", line)
}

func (err ExpectedTokenErr) Error() string {
	return fmt.Sprintf("expected %v, found '%s' at %s",
		err.Expected, err.Got.Source, pos(err.Got.Position),
	)
}
