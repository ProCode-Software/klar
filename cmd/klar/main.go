package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ProCode-Software/klar/cmd/klar/internal/klarcmd"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/run"
)

var (
	commands = klarcmd.KlarCommands
	aliases  = klarcmd.KlarCommandAliases
	profiler prof
)

var RandomDescriptions = []string{
	"A simple, modern, and clean programming language",
	"The progressive programming language",
	"Not another C-based programming language",
	"A koala-approved programming language!",
}

func main() {
	defer cli.HandleSignalExit()
	// startProf()
	// defer stopProf()
	args := os.Args
	if len(args) < 2 {
		tryPipe()
		ShowHelp(os.Stderr, false)
		cli.Exit(2)
	}
	cmdName := args[1]
	switch cmdName {
	case "-":
		tryPipe()
		command.Run(commands["repl"])
	case "-c":
		if len(args) < 3 {
			cli.Failure("Expected program as string\n\nUsage: ",
				ansi.BoldGreen("klar ")+ansi.Cyan("-c ")+ansi.Blue("<program>\n\n"),
				"Use "+ansi.Cyan("'klar --help'")+" for more information.",
			)
			cli.Exit(2)
		}
		run.RunInput(strings.NewReader(args[2]), "string")
	case "--help", "-h":
		ShowHelp(os.Stdout, true)
	case "-v", "--version":
		fmt.Printf("Klar %s\n", cli.KlarVersion)
	case "test", "glas", "upgrade", "new", "format", "check",
		"docs", "lint", "clean", "generate", "zen":
		cli.Failure(ansi.ColorSprintf(ansi.CodeBold,
			"Command %s isn't implemented yet", ansi.Cyan(cmdName),
		))
	case "help":
		if len(args) < 3 || args[2] == "" {
			ShowHelp(os.Stdout, true)
			cli.Exit(0)
		}
		// klar help cmd -> klar cmd --help
		cmd := args[2]
		if command.Lookup(cmd, commands, aliases) != nil {
			os.Args[1], cmdName = cmd, cmd
			os.Args[2] = "--help"
		}
		fallthrough
	default:
		if args[1][0] == '-' {
			// Invalid usage
			// TODO: show flags
			cli.Exit(2)
		}
		cmd := command.Lookup(cmdName, commands, aliases)
		if cmd != nil {
			command.Run(cmd)
			break
		}
		// Equivalent to `klar run [file]`
		os.Args = append([]string{"klar", "run"}, os.Args[1:]...)
		command.Run(commands["run"])
	}
}

func tryPipe() {
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
		return
	}
	// Pipe
	run.RunInput(os.Stdin, "standardInput")
	cli.Exit(0)
}
