package argparse

import (
	"errors"
	"strings"
)

var ErrHelp = errors.New("--help or -h flag passed")

type ErrUnknownFlags struct {
	Flags []string
}
type ErrMissingArgs struct {
	Missing []string
}
type ErrExtraneousArgs struct {
	Extra []string
}

func (err *ErrUnknownFlags) Error() string {
	return "unknown flags: " + strings.Join(err.Flags, ", ")
}
func (err *ErrMissingArgs) Error() string {
	return "missing arguments: " + strings.Join(err.Missing, ", ")
}
func (err *ErrExtraneousArgs) Error() string {
	return "too many arguments: " + strings.Join(err.Extra, ", ")
}