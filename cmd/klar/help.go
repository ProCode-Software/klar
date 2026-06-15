package main

import (
	"fmt"
	"io"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/util"
)

// TODO: Should these be bright?
var RandomDescriptions = []string{
	ansi.Magenta("✨ A simple, modern, and clean programming language"),
	ansi.Green("⏩ The progressive programming language"),
	ansi.Yellow("Not another C-based programming language"),
	ansi.Cyan("🐨 A koala-approved programming language!"),
}

func KlarGradient(text string) string {
	// This is just for the VSCode color dialog
	rgba := func(r, g, b, _ uint8) [3]int { return [3]int{int(r), int(g), int(b)} }
	return ansi.Gradient(text, rgba(189, 247, 90, 1), rgba(91, 220, 230, 1))
}

func ShowHelp(w io.Writer, full bool) {
	hb := NewHelpBuilder(w)

	// Title
	hb.Println(
		ansi.Bold(KlarGradient("Klar:")),
		util.RandomSlice(RandomDescriptions),
		ansi.Gray("v"+cli.KlarVersion),
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
	hb.Command("check", "Typecheck a Klar project")
	hb.Command("format", "Format source code")
	hb.Command("lint", "Lint source code")
	hb.Command("new", "Create a new Klar project")
	hb.Command("test", "Run tests for a Klar project")

	hb.Split(ansi.Cyan)
	hb.Command("clean", "Clean build cache")
	hb.Command("upgrade", "Upgrade Klar to the latest version")
	hb.Command("zen", "Show the Zen of Klar")
	hb.Command("help", "Get help for a command or show this message")
	hb.Flush()

	hb.Println(
		"\nUse",
		ansi.Magenta("klar"), ansi.Yellow("help"), ansi.Cyan("<subcommand>"),
		"for more information about a command.",
	)

	if full {
		FlagHelp(hb)
	}

	// Social Links
	hb.ShortTitle("GitHub")
	hb.Println(ansi.Magenta("https://github.com/ProCode-Software/klar"))
}

func FlagHelp(hb *HelpBuilder) {
	hb.Title("Flags")
	hb.Color = ansi.Cyan
	hb.TW.WriteCells(hb.Color("-c")+ansi.Green(" <script>"), "Evaluate code from a string")
	hb.TW.WriteCells(hb.Color("-v")+", "+hb.Color("--version"), "Print the Klar version")
	hb.TW.WriteCells(hb.Color("-h")+", "+hb.Color("--help"), "Print this help message")
	hb.Flush()
}

// A HelpBuilder writes a list of commands and flags to an [io.Writer].
type HelpBuilder struct {
	TW    *cli.TabWriter
	Color func(string) string
}

func NewHelpBuilder(w io.Writer) *HelpBuilder {
	tw := cli.NewTabWriterOutput(w)
	tw.Margin = 2
	tw.MinWidth = 8
	tw.Spacing = 4
	return &HelpBuilder{TW: tw}
}

// ShortTitle writes a newline then s in title style followed by a space.
func (hb *HelpBuilder) ShortTitle(s string) {
	fmt.Fprintf(hb.TW.Output, "\n%s%s ", ansi.Bold(s), ansi.BoldDim(":"))
}

// ShortTitleNoNewline is ShortTitle, but does not print a newline before the title.
func (hb *HelpBuilder) ShortTitleNoNewline(s string) {
	fmt.Fprintf(hb.TW.Output, "%s%s ", ansi.Bold(s), ansi.BoldDim(":"))
}

// Title ends the previous group and writes a header.
func (hb *HelpBuilder) Title(title string) {
	hb.Flush()
	fmt.Fprintf(hb.TW.Output, "\n%s%s\n", ansi.Bold(title), ansi.BoldDim(":"))
}

// Split flushes the tabwriter, writes a newline, and sets the color of the new group.
func (hb *HelpBuilder) Split(color func(string) string) {
	hb.Color = color
	hb.Flush()
	fmt.Fprintln(hb.TW.Output)
}

// Command writes a group entry.
func (hb *HelpBuilder) Command(name, desc string) {
	hb.TW.WriteCells(hb.Color(name), desc)
}

// Print writes to the output [io.Writer].
func (hb *HelpBuilder) Print(s ...any) {
	fmt.Fprint(hb.TW.Output, s...)
}

func (hb *HelpBuilder) Println(s ...any) {
	fmt.Fprintln(hb.TW.Output, s...)
}

// Flush flushes the tab writer, panicking if an error occurs.
func (hb *HelpBuilder) Flush() {
	if _, err := hb.TW.Flush(); err != nil {
		panic(err)
	}
}
