package main

import (
	"io"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/command"
	"github.com/ProCode-Software/klar/internal/util"
)

// TODO: Should these be bright?
var RandomDescriptions = []string{
	ansi.Magenta("✨ A simple, modern, and clean programming language"),
	ansi.Green("⏩ The progressive programming language"),
	ansi.Yellow("Not another C-based programming language"),
	ansi.Cyan("🐨 A koala-approved programming language!"),
}

func ShowHelp(w io.Writer, full bool) {
	hb := command.NewHelpBuilder(w)

	// Title
	hb.Println(
		ansi.Bold(ansi.Bit8(85, "Klar"))+ansi.BoldDim(":"), // or 8-bit 48-50/85-86
		util.RandomSlice(RandomDescriptions),
		ansi.Gray("v"+cli.KlarVersionAndCommit),
	)

	// Usage
	klar := ansi.BoldMagenta("klar")
	hb.ShortTitle("Usage")
	hb.Println(
		klar, ansi.Yellow("<command>"), ansi.Cyan("[args]"), "|",
		klar, ansi.Yellow("<file>"), "|",
		klar, ansi.Cyan("-c"), ansi.Green("<script>"),
	)

	hb.Title("Commands")

	hb.Color = ansi.BrightGreen
	hb.Command("run", "Run a Klar file or project")
	hb.Command("repl", "Start an interactive REPL session with Klar")

	hb.Split(ansi.BrightBlue)
	hb.Command("build", "Compile a Klar project")
	hb.Command("new", "Create a new Klar project")
	hb.Command("format", "Format source code")
	hb.Command("lint", "Check your code for correctness")
	hb.Command("test", "Run tests for a Klar project")
	if full {
		hb.Command("lsp", "Start the Klar Language Server (for IDEs only)")
	}

	hb.Split(ansi.Cyan)
	hb.Command("clean", "Clean build cache")
	hb.Command("upgrade", "Upgrade Klar to the latest version")
	hb.Command("zen", "Show the Zen of Klar") // TODO: `klar lore` instead?
	hb.Command("help", "Get help for a command or show this message")
	hb.Flush()

	hb.Println(
		"\nUse",
		ansi.Magenta("klar"), ansi.Yellow("help"), ansi.Cyan("<subcommand>"),
		"for more information about a command.",
	)

	if full {
		FlagHelp(hb)
		// TODO: Should we show accepted env vars? Or show them in `klar help env`?
	}

	// Social Links
	hb.ShortTitle("GitHub")
	hb.Println(ansi.Magenta(cli.KlarGitHub))
}

func FlagHelp(hb *command.HelpBuilder) {
	hb.Title("Flags")
	hb.Color = ansi.Cyan
	hb.TW.WriteCells(hb.Color("-c")+ansi.Green(" <script>"), "Evaluate code from a string")
	hb.TW.WriteCells(hb.Color("-v")+", "+hb.Color("--version"), "Print the Klar version")
	hb.TW.WriteCells(hb.Color("-h")+", "+hb.Color("--help"), "Print this help message")
	hb.Flush()
}
