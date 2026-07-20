package command

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/pkg/argparse"
)

func (c *Command) handleFlagError(err error) {
	forMoreHelp := func() {
		ansi.TagFprintfln(os.Stderr, "\nUse <c!>--help</c!> for more information.")
	}
	formatOpts := func(opts []string) string {
		return ansi.BrightYellow(strings.Join(
			opts,
			ansi.Partial(ansi.CodeReset)+", "+ansi.Partial(ansi.CodeBrightYellow),
		))
	}
	switch err := err.(type) {
	case argparse.HelpError:
		c.Help(os.Stdout)
		cli.Exit(0)
	case *argparse.UnknownFlagError:
		cli.ColorErrorfln(
			"<**>I don't understand the <c>%s</c> flag</**>",
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
		cli.ColorErrorfln(
			"<**>Missing arguments:</**> <c!>%s</c!>\n\n%s",
			err.Missing, c.ArgUsage(),
		)
		forMoreHelp()
	case *argparse.ExtraArgsError:
		cli.ColorErrorfln(
			"<**>Too many arguments provided:</**> <c!>%s</c!>\n"+
				"Expected %s arguments, but %d were provided.\n\n%s",
			strings.Join(err.Extra, " "), klarerrs.FormatCount(len(c.Usage), "argument"),
			len(err.Extra), c.ArgUsage(),
		)
		forMoreHelp()
	case *argparse.RepeatedFlagError:
		cli.ColorErrorfln(
			"<**>The flag <c!>%s</c!> was provided more than once</**>",
			argparse.FormatFlag(err.Flag),
		)
	case *argparse.InvalidValueError:
		cli.ColorErrorfln(
			"<**>Invalid %s <c!>%s</c!> passed to flag <c!>%s</c!></**>",
			argparse.TypeNames[err.Type], err.Input, argparse.FormatFlag(err.Flag),
		)
		forMoreHelp()
	case *argparse.MissingValueError:
		typ := argparse.TypeNames[err.Type]
		cli.ColorErrorfln(
			"<**>Expected %s value for flag <c!>%s</c!></**>",
			klarerrs.WithA(typ), argparse.FormatFlag(err.Flag),
		)
		if c.Flags.FlagDefs[err.Flag].Type == argparse.TypeEnum {
			opts := slices.Sorted(maps.Keys(c.Flags.GetOptions(err.Flag)))
			fmt.Fprintln(os.Stderr, "\nExpected one of:\n ", formatOpts(opts))
		}
		forMoreHelp()
	case *argparse.InvalidOptionError:
		cli.ColorErrorfln(
			"<**><y!>%s</y!> isn't a valid option for flag <c!>%s</c!>.</**>\n\n"+
				"Expected one of:\n  %s",
			err.Input, argparse.FormatFlag(err.Flag), formatOpts(err.ExpOptions),
		)
		forMoreHelp()
	}
	cli.Exit(2)
}
