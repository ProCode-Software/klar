package klarcmd

import (
	"github.com/ProCode-Software/klar/cmd/klar/internal/build"
	"github.com/ProCode-Software/klar/cmd/klar/internal/command"
	"github.com/ProCode-Software/klar/cmd/klar/internal/repl"
)

var KlarCommands = map[string]*command.Command{
	"build": {
		ShortDescription: "Compile a project to JavaScript",
		LongDescription:  KlarBuildHelp,
		SeeAlso:          s{"run", "test", "check"},
		Flags:            build.Flags,
		Run:              build.Build,
		Examples: examples{
			{"build", nil, nil, "Build the current project to the default output directory"},
		},
	},
	"repl": {
		ShortDescription: "Start an interactive REPL session with Klar",
		LongDescription:  KlarREPLHelp,
		Run:              repl.Run,
		SeeAlso:          s{"run", "build"},
	},
}

type (
	s        = []string
	examples = []command.ExampleCmd
)

// Set command names
func init() {
	for name, cmd := range KlarCommands {
		cmd.Name = name
	}
}
