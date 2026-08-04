package command

import (
	"fmt"
	"io"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
)

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
