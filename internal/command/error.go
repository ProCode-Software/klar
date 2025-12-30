package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/pkg/argparse"
)

func (c *Command) handleFlagError(err error) {
	forMoreHelp := func() {
		ansi.Fprintfln(os.Stderr, "\nUse <c!>--help</c!> for more information.")
	}
	switch err := err.(type) {
	case argparse.HelpError:
		c.Help(os.Stdout)
		cli.Exit(0)
	case *argparse.UnknownFlagError:
		cli.ColorErrorfln("<**>I don't understand the <c>%s</c> flag</**>",
			argparse.FormatFlag(err.Flag),
		)
		if len(c.Flags.FlagDefs) > 0 {
			fmt.Fprintln(os.Stderr, "\nThe available flags are:")
			os.Stderr.WriteString(c.FlagString(2))
		} else {
			fmt.Fprintln(os.Stderr, "This command doesn't accept flags.")
		}
		forMoreHelp()
	case *argparse.MissingArgsError:
		cli.ColorErrorfln("<**>Missing arguments:</**> <c!>%s</c!>\n\n%s",
			err.Missing, c.ArgUsage(),
		)
		forMoreHelp()
	case *argparse.ExtraArgsError:
		cli.ColorErrorfln("<**>Too many arguments provided:</**> <c!>%s</c!>\n"+
			"Expected %d arguments, but %d were provided.\n\n%s",
			strings.Join(err.Extra, " "), 0, len(err.Extra), c.ArgUsage(),
		)
		forMoreHelp()
	case *argparse.RepeatedFlagError:
		cli.ColorErrorfln("<**>The flag <c!>%s</c!> was provided more than once</**>",
			argparse.FormatFlag(err.Flag),
		)
	case *argparse.InvalidValueError:
		cli.ColorErrorfln("<**>Invalid %s <c!>%s</c!> passed to flag <c!>%s</c!></**>",
			argparse.TypeNames[err.Type], err.Input, argparse.FormatFlag(err.Flag),
		)
		forMoreHelp()
	case *argparse.MissingValueError:
		typ := argparse.TypeNames[err.Type]
		cli.ColorErrorfln("<**>Expected %s value for flag <c!>%s</c!></**>",
			errors.WithA(typ), argparse.FormatFlag(err.Flag),
		)
		forMoreHelp()
	case *argparse.InvalidOptionError:
		cli.ColorErrorfln(
			"<**><y!>%s</y!> isn't a valid option for flag <c!>%s</c!>.</**>\n\n"+
				"Expected one of:"+
				"\n  %s",
			err.Input, argparse.FormatFlag(err.Flag), ansi.BrightYellow(
				strings.Join(
					err.ExpOptions,
					ansi.Partial(ansi.CodeReset)+", "+ansi.Partial(ansi.CodeBrightYellow),
				),
			),
		)
		forMoreHelp()
	}
	cli.Exit(2)
}
