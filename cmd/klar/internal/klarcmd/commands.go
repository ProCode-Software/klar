package klarcmd

import (
	"github.com/ProCode-Software/klar/cmd/klar/internal/build"
	"github.com/ProCode-Software/klar/cmd/klar/internal/help"
	"github.com/ProCode-Software/klar/cmd/klar/internal/repl"
	"github.com/ProCode-Software/klar/cmd/klar/internal/run"
	"github.com/ProCode-Software/klar/internal/command"
)

var KlarCommands = map[string]*command.Command{
	"build": {
		ShortDescription: "Compile a project to JavaScript",
		LongDescription:  build.LongDescription,
		SeeAlso:          s{"run", "test", "check"},
		Flags:            build.Flags,
		Run:              build.Build,
		Examples: ex{
			{"build", nil, nil, "Build the current project to the default output directory"},
			{"build", s{"./src/foo"}, s{"-t", "node"}, "Build the module at src/foo for Node with default settings"},
			{"build", s{"-"}, s{"-o", "index.js"}, "Read a script from standard input and compile it to index.js"},
			{"build", s{"@foo.bar.baz"}, s{"-v"}, "Compile the module foo.bar.baz with default settings and verbose output"},
		},
	},
	"repl": {
		ShortDescription: "Start an interactive REPL session with Klar",
		LongDescription:  repl.LongDescription,
		Run:              repl.Run,
		SeeAlso:          s{"run", "build"},
	},
	"run": {
		ShortDescription: "Run a Klar project, file, or module",
		LongDescription:  run.LongDescription,
		Run:              run.Run,
		SeeAlso:          s{"build", "test"},
	},
	"help": {
		ShortDescription: "Get help on the Klar CLI",
		LongDescription:  help.LongDescription,
		Run:              help.Run,
		Flags:            help.Flags,
		Examples: ex{
			{"help", s{"help"}, nil, "Show a description and usage for the 'klar help' command"},
			{"help", s{"build"}, nil, "Get help on the 'build' command"},
			{"help", s{"js"}, nil, "Not just commands are supported (show info about JavaScript compilation)"},
		},
	},
}

type (
	s  = []string
	ex = []command.ExampleCmd
)

// Set command names
func init() {
	for name, cmd := range KlarCommands {
		cmd.Name = name
	}
	command.Commands = KlarCommands
}
