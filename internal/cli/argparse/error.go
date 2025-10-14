package argparse

import (
	"errors"
	"fmt"
	"strings"
)

var ErrHelp = errors.New("--help or -h flag passed")

type (
	ErrUnknownFlag    struct{ Flag string }
	ErrMissingArgs    struct{ Missing []string }
	ErrExtraneousArgs struct{ Extra []string }
	ErrInvalidBool    struct{ Flag string }
	ErrInvalidNumber  struct{ Flag, Input string }
	ErrMissingValue   struct {
		Flag string
		Type FlagType
	}
	ErrInvalidOption struct {
		Flag       string
		ExpOptions []string
	}
)

func (err *ErrUnknownFlag) Error() string {
	return "unknown flag: " + FormatFlag(err.Flag)
}

func (err *ErrMissingArgs) Error() string {
	return "missing arguments: " + strings.Join(err.Missing, ", ")
}

func (err *ErrExtraneousArgs) Error() string {
	return "too many arguments: " + strings.Join(err.Extra, ", ")
}

func (err *ErrInvalidNumber) Error() string {
	return fmt.Sprintf("invalid number for flag %s: %s", err.Flag, err.Input)
}

func (err *ErrInvalidBool) Error() string {
	return "flag " + err.Flag + "is not a boolean flag"
}

func (err *ErrInvalidOption) Error() string {
	return fmt.Sprintf("invalid option for flag %s: expected one of: %s",
		err.Flag, strings.Join(err.ExpOptions, ", "),
	)
}

func (err *ErrMissingValue) Error() string {
	return fmt.Sprintf(
		"missing value for flag %s: expected %s", err.Flag, TypeNames[err.Type],
	)
}

var TypeNames = []string{
	TypeStringFlag: "string",
	TypeNumberFlag: "number",
	TypeBoolFlag:   "boolean",
	TypeEnumFlag:   "option",
	TypeListFlag:   "list",
}
