package klarcmd

import (
	"github.com/ProCode-Software/klar/cmd/klar/internal/build"
	"github.com/ProCode-Software/klar/cmd/klar/internal/repl"
	"github.com/ProCode-Software/klar/cmd/klar/internal/run"
	"github.com/ProCode-Software/klar/internal/command"
)

var KlarCommands = map[string]*command.Command{
	"build": {
		ShortDescription: "Compile a project to JavaScript",
		LongDescription:  KlarBuildHelp,
		SeeAlso:          s{"run", "test", "check"},
		Flags:            build.Flags,
		Run:              build.Build,
		Examples: ex{
			{"build", nil, nil, "Build the current project to the default output directory"},
			{"build", s{"a", "b"}, s{"-t", "file", "-x", "f"}, "Build the current project to the default output directory"},
		},
	},
	"repl": {
		ShortDescription: "Start an interactive REPL session with Klar",
		LongDescription:  KlarREPLHelp,
		Run:              repl.Run,
		SeeAlso:          s{"run", "build"},
	},
	"run": {
		ShortDescription: "Run a Klar project, file, or module",
		LongDescription:  KlarRunHelp,
		Run:              run.Run,
		SeeAlso:          s{"build", "test"},
	},
}

type (
	s        = []string
	ex = []command.ExampleCmd
)

// Set command names
func init() {
	command.Commands = KlarCommands
	for name, cmd := range KlarCommands {
		cmd.Name = name
	}
}
