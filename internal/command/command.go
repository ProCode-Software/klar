package command

import (
	"github.com/ProCode-Software/klar/internal/cli"
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
	Flags           *cli.ArgParser
	LongDescription string
	SeeAlso         []string
	Examples        []ExampleCmd
}

type RunFunc func(r *Runner)

type Runner struct {
	Parser *cli.ArgParser
}

func (r *Runner) Arg(i int) string           { return r.Parser.ArgAt(i) }
func (r *Runner) Flag(n string) any          { return r.Parser.Flag(n) }
func (r *Runner) IsDefault(n string) bool    { return r.Parser.IsDefault(n) }
func (r *Runner) AllFlags() map[string]any   { return r.Parser.Flags }
func (r *Runner) AllArgs() []string          { return r.Parser.Args }
func (r *Runner) StringFlag(n string) string { return r.Flag(n).(string) }
func (r *Runner) BoolFlag(n string) bool     { return r.Flag(n).(bool) }

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
		cmd.Flags = cli.NewArgParser(0)
	}
	cmd.Flags.Parse()
	cmd.Run(&Runner{Parser: cmd.Flags})
}
