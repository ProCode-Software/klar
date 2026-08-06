package glas

import (
	"io"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/command"
)

func ShowHelp(w io.Writer, full bool) {
	const glasColor = 111
	hb := command.NewHelpBuilder(w)

	// Title
	hb.Println(
		ansi.Bold(ansi.Bit8(glasColor, "Glas"))+ansi.BoldDim(":"),
		ansi.Bit8(192, "The Klar package manager"), // 69/121/122/123/156-159/192/193/195
		ansi.Gray("v"+cli.KlarVersionAndCommit),
	)

	// Usage
	hb.ShortTitle("Usage")
	hb.Println(
		ansi.Bold(ansi.Bit8(glasColor, "glas")),
		ansi.Yellow("<command>"), ansi.Cyan("[args]"),
	)

	hb.Title("Commands")

	for i, g := range groups {
		color := func(s string) string { return ansi.Color(g.color, s) }
		if i == 0 {
			hb.Color = color
		} else {
			hb.Split(color)
		}
		for _, cmd := range g.commands {
			hb.Command(cmd, Commands[cmd].ShortDescription)
		}
	}
	hb.Flush()

	hb.Println(
		"\nUse",
		ansi.Bit8(glasColor, "glas"), ansi.Yellow("help"), ansi.Cyan("<subcommand>"),
		"for more information about a command.",
	)

	// Social Links
	hb.ShortTitle("GitHub")
	hb.Println(ansi.Magenta(cli.KlarGitHub))
}

var groups = []struct {
	color    string
	commands []string
}{
	{ansi.ColorBit8(159), []string{"add", "update", "remove", "install"}},
	{ansi.ColorBit8(3), []string{"list", "outdated", "why", "audit", "info", "clean"}},
}
