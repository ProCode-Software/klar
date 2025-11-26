package argparse

import (
	"fmt"
	"strings"
)

type (
	UnknownFlagError    struct{ Flag string }
	MissingArgsError    struct{ Missing []string }
	ExtraneousArgsError struct{ Extra []string }
	InvalidBoolError    struct{ Flag string }
	InvalidNumberError  struct{ Flag, Input string }
	HelpError           struct{}
	MissingValueError   struct {
		Flag string
		Type FlagType
	}
	InvalidOptionError struct {
		Flag       string
		ExpOptions []string
	}
)

func (err *UnknownFlagError) Error() string {
	return "unknown flag: " + FormatFlag(err.Flag)
}

func (err *MissingArgsError) Error() string {
	return "missing arguments: " + strings.Join(err.Missing, ", ")
}

func (err *ExtraneousArgsError) Error() string {
	return "too many arguments: " + strings.Join(err.Extra, ", ")
}

func (err *InvalidNumberError) Error() string {
	return fmt.Sprintf("invalid number for flag %s: %s", err.Flag, err.Input)
}

func (err *InvalidBoolError) Error() string {
	return "flag " + err.Flag + "is not a boolean flag"
}

func (err *InvalidOptionError) Error() string {
	return fmt.Sprintf("invalid option for flag %s: expected one of: %s",
		err.Flag, strings.Join(err.ExpOptions, ", "),
	)
}

func (err *MissingValueError) Error() string {
	return fmt.Sprintf(
		"missing value for flag %s: expected %s", err.Flag, TypeNames[err.Type],
	)
}

func (*HelpError) Error() string { return "help requested" }

var TypeNames = []string{
	TypeStringFlag: "string",
	TypeNumberFlag: "number",
	TypeBoolFlag:   "boolean",
	TypeEnumFlag:   "option",
	TypeListFlag:   "list",
}
