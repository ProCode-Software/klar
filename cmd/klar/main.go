package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/cmd/klar/internal/klarcmd"
	"github.com/ProCode-Software/klar/cmd/klar/internal/run"
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
		runString(args[2])
	case "--help", "-h":
		ShowHelp(true)
	case "-v", "--version":
		fmt.Printf("Klar %s\n", version.KlarVersion)
	case "test", "glas":
		cli.Failure("Not implemented: ", fmt.Sprintf(
			"Command '%s' is not implemented yet.", cmdName,
		))
	case "help":
		if len(args) < 3 || args[2] == "" {
			ShowHelp(true)
			os.Exit(0)
		}
		// klar help cmd -> klar cmd --help
		if command.Lookup(args[2], Commands, Aliases) != nil {
			os.Args = []string{"klar", os.Args[1], "--help"}
		}
		fallthrough
	default:
		if args[1][0] == '-' {
			// Invalid usage
			os.Exit(2)
		}
		cmd := command.Lookup(cmdName, Commands, Aliases)
		if cmd != nil {
			command.Run(cmd)
			os.Exit(0)
		}
		// Equivalent to `klar run [file]`
		os.Args = append([]string{"klar", "run"}, os.Args[1:]...)
		command.Run(Commands["run"])
	}
}

func tryPipe() {
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
		return
	}
	// Is pipe
	run.RunInput(os.Stdin, "standardInput")
	os.Exit(0)
}

func runString(s string) {
	run.RunInput(strings.NewReader(s), "string")
}
