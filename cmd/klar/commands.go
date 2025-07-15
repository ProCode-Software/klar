package main

import "github.com/ProCode-Software/klar/internal/cli"

var Commands = map[string]cli.CmdInfo{
	"build": {
		Name:        "build",
		Description: "Compiles a project to JavaScript",
	},
}
