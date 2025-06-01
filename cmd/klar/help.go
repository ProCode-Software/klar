package main

import (
	"fmt"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/version"
)

type HelpBuilder struct {
	strings.Builder
}

func command(color, name, desc string) string {
	return fmt.Sprintf("    %s%-12s%s %s\n", color, name, cli.ANSIReset, desc)
}

func header(title string) string {
	return cli.ANSIBold + title + cli.ANSIReset
}

func formatVersion() string {
	return fmt.Sprintf("%s(v%s)%s\n\n", cli.ANSIDim, version.KlarVersion, cli.ANSIReset)
}

var HelpString = header("Klar: ") + cli.ANSIGreen + `A simple, modern, and clean programming language ` + cli.ANSIReset + formatVersion() +
	header("Usage: ") + cli.ANSICyan + "klar <command> [flags]" +
	cli.ANSIReset + " | " + cli.ANSICyan + "klar <file>\n\n" + cli.ANSIReset +

	header("Commands:\n") +
	command(cli.ANSIGreen, "run", "Run a Klar file or project") +
	command(cli.ANSIGreen, "repl", "Start an interactive Klar read-eval-print loop (REPL)\n") +

	command(cli.ANSIMagenta, "build", "Compile a Klar project") +
	command(cli.ANSIMagenta, "test", "Test a Klar project\n") +

	command(cli.ANSIBlue, "init", "Create a new Klar project") +
	command(cli.ANSIBlue, "add", "Install dependencies for a project") +

	`
Use ` + cli.ANSICyan + "'klar help <subcommand>'" + cli.ANSIReset +
	" for more information about a command.\n\n" +

	cli.ANSIYellow + `GitHub: ` +
	cli.ANSIBlue + "https://github.com/ProCode-Software/klar\n" + cli.ANSIReset
