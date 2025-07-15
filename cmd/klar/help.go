package main

import (
	"fmt"
	"strings"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/version"
)

const maxLen = max(8, 5)

type HelpBuilder struct {
	strings.Builder
}

func command(name, desc string) string {
	l := len(name)
	if l <= maxLen {
		return fmt.Sprintf("    %-*s%s\n", maxLen+2, name, desc)
	}
	return fmt.Sprintf(name[:5]+"    %-*s%s%s\n", maxLen+2, name[5:l-3], name[l-3:], desc)
}

func header(title string) string {
	return ansi.Bold(title)
}

var HelpString = header("Klar: ") +
	ansi.Cyan("A simple, modern, and clean programming language ") +
	ansi.Dim("v"+version.KlarVersion) + "\n\n" +

	header("Usage: ") +
	ansi.BoldGreen("klar ") + ansi.Yellow("<command> ") + ansi.Cyan("[flags]") +
	ansi.Dim(" | ") +
	ansi.BoldGreen("klar ") + ansi.Yellow("<file>\n\n") +

	header("Commands:\n") +
	command(ansi.Green("run"), "Run a Klar file or project") +
	command(ansi.Green("repl"), "Start an interactive Klar read-eval-print loop (REPL)\n") +

	command(ansi.Magenta("build"), "Compile a Klar project") +
	command(ansi.Magenta("test"), "Test a Klar project\n") +

	command(ansi.Blue("init"), "Create a new Klar project") +
	command(ansi.Blue("add"), "Install dependencies for a project") +

	"\nUse " + ansi.Cyan("'klar help <subcommand>'") +
	" for more information about a command.\n\n" +

	header("GitHub: ") + ansi.Blue("https://github.com/ProCode-Software/klar\n")
