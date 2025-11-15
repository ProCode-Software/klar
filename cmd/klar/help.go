package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/version"
)

type HelpBuilder struct {
	strings.Builder
}

func ShowHelp(full bool) {
	tw := tabwriter.NewWriter(os.Stdout, 20, 0, 2, ' ', 0)
	cmd := func(name, desc string) { fmt.Fprintf(tw, "  %s\t%s\n", name, desc) }
	header := func(name string) {
		fmt.Fprintln(tw, ansi.Bold(name)+ansi.Dim(":"))
	}
	shortHead := func(name string) {
		fmt.Fprint(tw, ansi.Bold(name)+ansi.Dim(": "))
	}
	print := func(c func(string) string, s string) { fmt.Fprint(tw, c(s)) }

	shortHead(ansi.BoldBrightWhite("Klar"))
	print(ansi.Cyan, "A simple, modern, and clean programming language ")
	print(ansi.Gray, "v"+version.KlarVersion+"\n\n")

	shortHead("Usage")
	fmt.Fprint(tw, ansi.BoldGreen("klar ")+ansi.Yellow("<command> ")+ansi.Cyan("[args]"))
	print(ansi.Dim, " | ")
	fmt.Fprint(tw, ansi.BoldGreen("klar ")+ansi.Yellow("<file>"))
	print(ansi.Dim, " | ")
	fmt.Fprint(tw, ansi.BoldGreen("klar ")+ansi.Cyan("-c "+ansi.Blue("<script>\n\n")))

	header("Commands")
	cmd(ansi.Green("run"), "Run a Klar file or project")
	cmd(ansi.Green("repl"), "Start an interactive REPL session with Klar\n")

	cmd(ansi.Magenta("build"), "Compile a Klar project")
	cmd(ansi.Magenta("test"), "Test a Klar project\n")

	cmd(ansi.Blue("init"), "Create a new Klar project")
	cmd(ansi.Blue("add"), "Install dependencies for a project\n")
	tw.Flush()

	fmt.Fprintf(tw, "Use %s for more information about a command.\n\n",
		ansi.Cyan("klar help <subcommand>"))

	if full {
		tw.Init(os.Stdout, 0, 0, 2, ' ', 0)
		header("Flags")
		cmd(ansi.Cyan("-c")+ansi.Dim("")+ansi.Blue(" <script>"), "Run a Klar script from a string")
		cmd(ansi.Cyan("-v")+ansi.Dim(", ")+ansi.Cyan("--version"), "Print the Klar version")
		cmd(ansi.Cyan("-h")+ansi.Dim(", ")+ansi.Cyan("--help"), "Print this help message\n")
		tw.Flush()
	}

	fmt.Println(ansi.Bold("GitHub")+ansi.Dim(":"),
		ansi.Magenta("https://github.com/ProCode-Software/klar"))
}

const helpTemplate = ``
