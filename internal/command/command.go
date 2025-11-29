package command

import (
	"os"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/argparse"
)

var (
	ExecName string = "klar"
	Commands map[string]*Command
)

type ExampleCmd struct {
	Command     string
	Args        []string
	Flags       []string
	Description string
}

type Command struct {
	Name             string
	ShortDescription string
	Usage            []string
	Aliases          []string
	Run              RunFunc

	// Shown in command help
	Subcommands     []*Command
	Flags           *argparse.Parser
	LongDescription string
	SeeAlso         []string
	Examples        []ExampleCmd
}

// TODO: documentation URL

type RunFunc func(r *Runner)

type Runner struct {
	Parser *argparse.Parser
}

func (r *Runner) Arg(i int) string                   { return r.Parser.ArgAt(i) }
func (r *Runner) Flag(n string) argparse.Flag        { return r.Parser.Flag(n) }
func (r *Runner) AllFlags() map[string]argparse.Flag { return r.Parser.Flags }
func (r *Runner) AllArgs() []string                  { return r.Parser.Args }
func (r *Runner) StringFlag(n string) string {
	return r.Flag(n).(*argparse.StringFlag).Val
}
func (r *Runner) NamedArg(name string) string { return r.Parser.ArgByName(name) }

func (r *Runner) BoolFlag(n string) bool {
	return r.Flag(n).(*argparse.BoolFlag).Val
}

func Lookup(
	name string, commands map[string]*Command, aliases map[string]string,
) *Command {
	if cmd, ok := commands[name]; ok {
		return cmd
	}
	if aliases != nil {
		return commands[aliases[name]]
	}
	return nil
}

func Run(cmd *Command) {
	if cmd == nil {
		panic("command: Run(nil)")
	}
	if cmd.Run == nil {
		panic("cannot run command '" + cmd.Name + ": Run function is not defined")
	}
	if cmd.Flags == nil {
		cmd.Flags = argparse.NewParser()
	}
	cmd.Flags.ShiftFirst = true
	cmd.Flags.InputArgs = os.Args[1:]
	if err := cmd.Flags.Parse(); err != nil {
		cmd.handleFlagError(err)
		return
	}
	cmd.Run(&Runner{Parser: cmd.Flags})
}

func (c *Command) handleFlagError(err error) {
	stdout, stderr := os.Stdout, os.Stderr
	_ = stderr
	switch err.(type) {
	case *argparse.HelpError:
		c.Help(stdout)
		cli.Exit(0)
	case *argparse.InvalidBoolError:
	case *argparse.ExtraneousArgsError:
	case *argparse.InvalidNumberError:
	case *argparse.InvalidOptionError:
	case *argparse.MissingArgsError:
	case *argparse.MissingValueError:
	case *argparse.UnknownFlagError:
	}
}
