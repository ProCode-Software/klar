package klarcmd

import (
	"github.com/ProCode-Software/klar/cmd/klar/internal/build"
	"github.com/ProCode-Software/klar/cmd/klar/internal/check"
	"github.com/ProCode-Software/klar/cmd/klar/internal/clean"
	"github.com/ProCode-Software/klar/cmd/klar/internal/format"
	"github.com/ProCode-Software/klar/cmd/klar/internal/help"
	klarnew "github.com/ProCode-Software/klar/cmd/klar/internal/new"
	"github.com/ProCode-Software/klar/cmd/klar/internal/repl"
	"github.com/ProCode-Software/klar/cmd/klar/internal/run"
	"github.com/ProCode-Software/klar/cmd/klar/internal/upgrade"
	"github.com/ProCode-Software/klar/cmd/klar/internal/zen"
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
			// TODO: Give these example projects creative names
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
	"clean": {
		ShortDescription: "Clean build cache",
		LongDescription:  clean.LongDescription,
		SeeAlso:          s{"build", "upgrade"},
		Run:              clean.Run,
		Flags:            clean.Flags,
	},
	"zen": {
		ShortDescription: "Show the Zen of Klar",
		LongDescription:  zen.LongDescription,
		Run:              zen.Run,
		SeeAlso:          s{"help"},
	},
	"new": {
		ShortDescription: "Create a new Klar project",
		LongDescription:  klarnew.LongDescription,
		Flags:            klarnew.Flags,
		Run:              klarnew.Run,
		SeeAlso:          s{"run", "new", "repl"},
		Examples: ex{
			{"new", nil, nil, "Interactively create a new project in the current folder"},
			{"new", s{"myProject"}, nil, "Create a new project in the 'myProject' folder"},
			{"new", s{"pkg/foo"}, nil, "Add a new package in the current project"},
			{"new", nil, s{"--type", "library", "--add-tests"}, "Create a new library in the current folder, with template tests"},
		},
	},
	"check": {
		ShortDescription: "Typecheck a Klar project",
		Run:              check.Run,
		LongDescription:  format.LongDescription,
	},
	"format": {
		ShortDescription: "Format source code",
		Run:              format.Run,
		LongDescription:  format.LongDescription,
	},
	"lint": {
		ShortDescription: "Lint source code",
	},
	"test": {
		ShortDescription: "Run tests for a Klar project",
	},
	"upgrade": {
		ShortDescription: "Upgrade Klar to the latest version",
		Run:              upgrade.Run,
		LongDescription:  upgrade.LongDescription,
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
		if cmd.Flags != nil {
			cmd.Usage = cmd.Flags.Pattern
		}
	}
	command.Commands = KlarCommands
}
