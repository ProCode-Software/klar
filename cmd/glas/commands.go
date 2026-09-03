package glas

import (
	"github.com/ProCode-Software/klar/cmd/glas/internal/add"
	"github.com/ProCode-Software/klar/internal/command"
)

var Commands = map[string]*command.Command{
	"install": {
		ShortDescription: "Install all dependencies for the project",
		SeeAlso:          s{"add", "update"},
	},
	"add": {
		ShortDescription: "Add dependencies to a package",
		Run:              add.Run,
		Flags:            add.Flags,
		LongDescription:  add.LongDescription,
		Examples: []command.ExampleCmd{
			{"add", s{"github.com/ProCode-Software/klar"}, nil, "Install a package from GitHub (or any other Git repository)"},
			{"add", s{"proicons"}, s{"--npm"}, "Install 'proicons' from NPM"},
			{"add", s{"../greeter"}, nil, "Install a package stored locally"},
			{"add", s{"./pkg/server"}, nil, "Install a package from your project"},
		},
	},
	"update": {
		ShortDescription: "Update dependencies to their latest versions",
	},
	"outdated": {
		ShortDescription: "Show dependencies with updates available",
	},
	"remove": {
		ShortDescription: "Remove dependencies from the project",
	},
	"list": {
		ShortDescription: "List the project's dependencies",
	},
	"publish": {
		ShortDescription: "Prepare the package for publishing",
	},
	"clean": {
		ShortDescription: "Remove unused dependencies from the project",
	},
	"why": {
		ShortDescription: "Show why an indirect dependency is included by showing its dependents",
	},
	"info": {
		ShortDescription: "Show information about a package online",
	},
	"docs": {
		ShortDescription: "Show documentation for a package or module",
	},
	"audit": {
		ShortDescription: "Perform a security audit on the project's dependencies",
	},
	"clone": {
		ShortDescription: "Clone a package's source code into a project",
	},
}

type s = []string

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
