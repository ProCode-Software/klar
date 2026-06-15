package main

import (
	"fmt"
	"os"

	"github.com/ProCode-Software/klar/cmd/glas/internal/glascmd"
	"github.com/ProCode-Software/klar/cmd/klar/klarcmd"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/command"
)

var (
	commands = glascmd.Commands
	aliases  = glascmd.Aliases
	Version  = cli.KlarVersion
)

func main() {
	defer cli.HandleSignalExit()
	args := os.Args
	if len(args) < 2 {
		ShowHelp(os.Stderr)
		cli.Exit(2)
	}
	cmdName := args[1]
	switch cmdName {
	case "--help", "-h":
		ShowHelp(os.Stdout)
	case "-v", "--version":
		fmt.Printf("Glas/Klar %s\n", Version)
	case "": // TODO
		cli.Failure(ansi.ColorSprintf(
			ansi.CodeBold,
			"Command %s isn't implemented yet", ansi.Cyan(cmdName),
		))
	case "help":
		// glas help | glas help "" | glas help glas
		if len(args) < 3 || args[2] == "" || args[2] == "glas" {
			ShowHelp(os.Stdout)
			cli.Exit(0)
		}
		// glas help cmd -> glas cmd --help
		// If it's not a command, run `glas help` with the topic.
		cmd := args[2]
		if command.Lookup(cmd, commands, aliases) != nil {
			os.Args[1], cmdName = cmd, cmd
			os.Args[2] = "--help"
		}
		fallthrough
	default:
		if args[1][0] == '-' {
			// Expected a command
			cli.Exit(2)
		}
		cmd := command.Lookup(cmdName, commands, aliases)
		if cmd != nil {
			command.Run(cmd)
			break
		}
		// Unknown command
		cli.ColorErrorfln("<**>Can't find Glas command <c>%s</c></**>", cmdName)
		klarCmd := klarcmd.LookupKlarCmd(cmdName)
		if klarCmd == nil {
			ansi.Println("\n<y>Run <c>glas help</c> to see available commands.</>")
		} else {
			promptKlarRun(klarCmd, cmdName)
		}
		cli.Exit(2)
	}
}

func promptKlarRun(cmd *command.Command, providedName string) {
	shouldRun := cli.Confirm(ansi.Sprintf(
		"\n<b!>Hint</b!><dim>:</dim> I found the command <m>klar %s</m>. Do you want to run it?",
		providedName,
	), true)
	if !shouldRun {
		return
	}
	cli.Exit(0)
}
