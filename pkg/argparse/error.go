package argparse

import (
	"fmt"
	"strings"
)

type (
	// UnknownFlagError occurs when a flag that is not defined is provided
	UnknownFlagError struct{ Flag string }
	// MissingArgsError occurs when not enough arguments are provided
	MissingArgsError struct{ Missing string }
	// ExtraArgsError occurs when too many arguments are provided
	ExtraArgsError struct{ Extra []string }
	// RepeatedFlagError occurs when a flag is provided more than once
	RepeatedFlagError struct{ Flag string }
	// HelpError occurs when the --help or -h flag is provided
	HelpError struct{}
	// InvalidValueError occurs when a flag is provided with an invalid value.
	// For enums, [InvalidOptionError] is reported instead.
	InvalidValueError struct {
		Type        FlagType
		Flag, Input string
	}
	// MissingValueError occurs when a flag is provided without a value
	MissingValueError struct {
		Flag string
		Type FlagType
	}
	// InvalidOptionError occurs when a value that is not part of the expected
	// options is provided for a [TypeEnum] flag.
	InvalidOptionError struct {
		Flag, Input string
		ExpOptions  []string
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
