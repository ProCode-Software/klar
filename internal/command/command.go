package command

import (
	"io"
	"os"

	"github.com/ProCode-Software/klar/internal/cli/argparse"
)

var ExecName string = "klar"

type ExampleCmd struct {
	Command     string
	Args        []string
	Flags       []string
	Description string
}

type Command struct {
	Name             string
	ShortDescription string
	Usage            string
	Aliases          []string
	Run              RunFunc

	// Shown in command help
	Subcommands     []*Command
	Flags           *argparse.Parser
	LongDescription string
	SeeAlso         []string
	Examples        []ExampleCmd
}

type RunFunc func(r *Runner)

type Runner struct {
	Parser *argparse.Parser
}

func (r *Runner) Arg(i int) string                   { return r.Parser.ArgAt(i) }
func (r *Runner) Flag(n string) any                  { return r.Parser.Flag(n) }
func (r *Runner) AllFlags() map[string]argparse.Flag { return r.Parser.Flags }
func (r *Runner) AllArgs() []string                  { return r.Parser.Args }
func (r *Runner) StringFlag(n string) string {
	return r.Flag(n).(*argparse.StringFlag).Val
}

func (r *Runner) BoolFlag(n string) bool {
	return r.Flag(n).(*argparse.BoolFlag).Val
}

func Lookup(
	name string, commands map[string]*Command, aliases map[string]string,
) *Command {
	if cmd, ok := commands[name]; ok {
		return cmd
	}
	if aliases == nil {
		return nil
	}
	return commands[aliases[name]]
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
	}
	cmd.Run(&Runner{Parser: cmd.Flags})
}

func (c *Command) handleFlagError(err error) {
	if err == argparse.ErrHelp {
		c.Help(os.Stdout)
		os.Exit(0)
	}
	switch err.(type) {
	case *argparse.ErrInvalidBool:
	case *argparse.ErrExtraneousArgs:
	case *argparse.ErrInvalidNumber:
	case *argparse.ErrInvalidOption:
	case *argparse.ErrMissingArgs:
	case *argparse.ErrMissingValue:
	case *argparse.ErrUnknownFlag:
	}
}

func (c *Command) Help(file io.Writer) {
	
}
func (c *Command) Print(file io.Writer) {

}