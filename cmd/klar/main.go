package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/ProCode-Software/klar/cmd/klar/internal/klarcmd"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/version"
)

var (
	Commands = klarcmd.KlarCommands
	Aliases  = klarcmd.KlarCommandAliases
)

func main() {
	args := os.Args
	if len(args) < 2 {
		tryPipe()
		ShowHelp(false)
		os.Exit(2)
	}
	cmdName := args[1]
	switch cmdName {
	case "-":
		tryPipe()
		command.Run(Commands["repl"])
	case "-c":
		if len(args) < 3 {
			cli.Failure("Expected program as string\n\nUsage: ",
				ansi.BoldGreen("klar ")+ansi.Cyan("-c ")+ansi.Blue("<program>\n\n"),
				"Use "+ansi.Cyan("'klar --help'")+" for more information.",
			)
			os.Exit(2)
		}
		RunString(args[2])
	case "--help", "-h":
		ShowHelp(true)
	case "-v", "--version":
		fmt.Printf("Klar %s\n", version.KlarVersion)
	case "test", "glas":
		cli.Failure("Not implemented: ", fmt.Sprintf(
			"Command '%s' is not implemented yet.", cmdName,
		))
	case "help":
		if len(args) < 3 {
			ShowHelp(true)
			os.Exit(0)
		}
		// klar help cmd -> klar cmd --help
		os.Args[1] = os.Args[2]
		os.Args = append(os.Args, "--help")
		fallthrough
	default:
		cmd := command.Lookup(cmdName, Commands, Aliases)
		if cmd != nil {
			command.Run(cmd)
			os.Exit(0)
		}
		// Equivalent to `klar run [file]`
		os.Args = slices.Insert(os.Args, 1, "run")
		command.Run(Commands["run"])
	}
}
