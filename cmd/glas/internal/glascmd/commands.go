package glascmd

import "github.com/ProCode-Software/klar/internal/command"

var Commands = map[string]*command.Command{
	"install": {
		ShortDescription: "Installs all dependencies for the project",
		SeeAlso:          s{"add", "update"},
	},
	"add":     {},
	"update":  {},
	"remove":  {},
	"list":    {},
	"publish": {},
}

type (
	s  = []string
	ex = []command.ExampleCmd
)

// Set command names
func init() {
	for name, cmd := range Commands {
		cmd.Name = name
		if cmd.Flags != nil {
			cmd.Usage = cmd.Flags.Pattern
		}
	}
	command.Commands = Commands
}
