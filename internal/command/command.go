package command

import (
	"os"

	"github.com/ProCode-Software/klar/pkg/argparse"
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
	*argparse.Parser
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
	cmd.Flags.StartOffset = 1 // Command name
	cmd.Flags.InputArgs = os.Args[1:] // klar
	if err := cmd.Flags.Parse(); err != nil {
		cmd.handleFlagError(err)
		return
	}
	cmd.Run(&Runner{Parser: cmd.Flags})
}


