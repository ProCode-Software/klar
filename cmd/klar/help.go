package main

import (
	"fmt"
	"io"
	"math/rand"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
)

func ShowHelp(w io.Writer, full bool) {
	hb := NewHelpBuilder(w)

	hb.ShortTitleNoNewline(ansi.BoldBrightCyan("Klar"))
	hb.Print(RandomDescriptions[rand.Intn(len(RandomDescriptions))], " ",
		ansi.Gray("v"+cli.KlarVersion), "\n")

	klar := ansi.BoldGreen("klar ")
	pipe := " | "
	hb.ShortTitle("Usage")
	hb.Print(
		klar, ansi.Yellow("<command> "), ansi.Cyan("[args]"), pipe,
		klar, ansi.Yellow("<file>"), pipe,
		klar, ansi.Cyan("-c "), ansi.DimCyan("<script>"), "\n",
	)

	hb.Title("Commands")

	hb.Color = ansi.BrightMagenta
	hb.Command("run", "Run a Klar file or project")
	hb.Command("repl", "Start an interactive REPL session with Klar")

	hb.Split(ansi.BrightYellow)
	hb.Command("build", "Compile a Klar project")
	hb.Command("check", "Typecheck a Klar project")
	hb.Command("format", "Format source code")
	hb.Command("lint", "Lint source code")
	hb.Command("new", "Create a new Klar project")
	hb.Command("test", "Run tests for a Klar project")

	hb.Split(ansi.BrightCyan)
	hb.Command("clean", "Clean build cache")
	hb.Command("upgrade", "Upgrade Klar to the latest version")
	hb.Command("zen", "Show the Zen of Klar")
	hb.Command("help", "Get help for a command or show this message")
	hb.Flush()

	hb.Print("\nUse ", ansi.Cyan("klar help <subcommand>"),
		" for more information about a command.\n")

	if full {
		FlagHelp(hb)
	}

	hb.ShortTitle("GitHub")
	hb.Print(ansi.Magenta("https://github.com/ProCode-Software/klar"), "\n")
}

func FlagHelp(hb *HelpBuilder) {
	hb.Title("Flags")
	hb.Color = ansi.Cyan
	hb.TW.WriteCells(hb.Color("-c")+ansi.DimCyan(" <script>"), "Evaluate code from a string")
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

// Flush flushes the tab writer, panicking if an error occurs.
func (hb *HelpBuilder) Flush() {
	if _, err := hb.TW.Flush(); err != nil {
		panic(err)
	}
}

const helpTemplate = ``
