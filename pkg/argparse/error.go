package argparse

import (
	"fmt"
	"strings"
)

type (
	UnknownFlagError   struct{ Flag string }
	MissingArgsError   struct{ Missing string }
	ExtraArgsError     struct{ Extra []string }
	RepeatedFlagError  struct{ Flag string }
	ValueRequiredError struct{ Flag string }
	HelpError          struct{}
	InvalidValueError  struct {
		Type        FlagType
		Flag, Input string
	}
	MissingValueError struct {
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
	return "missing arguments: " + err.Missing
}

func (err *ExtraArgsError) Error() string {
	return "too many arguments: " + strings.Join(err.Extra, ", ")
}

func (err *ValueRequiredError) Error() string {
	return "flag " + err.Flag + " must have a value"
}

func (err *InvalidValueError) Error() string {
	return fmt.Sprintf("invalid %s for flag %s: %s",
		TypeNames[err.Type], err.Flag, err.Input,
	)
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

func (err *RepeatedFlagError) Error() string {
	return fmt.Sprintf("flag %s already provided", err.Flag)
}

func (HelpError) Error() string { return "--help flag provided" }

var TypeNames = []string{
	TypeString: "string",
	TypeInt:    "integer",
	TypeFloat:  "float",
	TypeBool:   "boolean",
	TypeEnum:   "option",
	TypeList:   "list",
}
