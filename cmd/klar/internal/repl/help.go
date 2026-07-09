package repl

import (
	"os"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"golang.org/x/term"
)

const LongDescription = `Starts a read-eval-print-loop (REPL) for Klar, which lets you type commands and Klar code to be evaluated in real time. It is useful for quickly running code snippets. Code can also be imported and run in the REPL from a Klar script, or exported to a script.

For available commands for the REPL, run 'klar repl <<< help' or type 'help' in the REPL.`

var actions = []struct{ name, args, desc, shortcut string }{
	{"exit", "", "Exit the REPL", "Ctrl+D"},
	{"help", "", "Display this help message", ""},
	{"load", "<file>", "Load and evaluate a Klar file in the REPL session", ""},
	{
		"multiline" + ansi.Reset() + ", " + ansi.Yellow("ml"), "",
		"Toggle multiline editing mode", "Ctrl+G",
	},
	{
		"save", "[file]",
		"Save all successfully evaluated commands in this REPL session to a file",
		"Ctrl+S",
	},
}

func (s *Session) PrintHelp() {
	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termWidth = 0
	}
	s.Oprintf(ansi.CodeBold, "Available REPL commands%s", ansi.Dim(":"))
	tw := cli.NewTabWriterOutput(s.Stdout())
	tw.Margin, tw.Spacing, tw.WrapIndent = 4, 4, 4
	tw.Flags |= cli.WrapTerminalColumns
	tw.TermWidth = termWidth
	tw.ReserveCapacity(len(actions), 2)
	for _, a := range actions {
		str := make([]string, 2)
		str[0] = ansi.Yellow(a.name) // Name
		if a.args != "" {
			str[0] += " " + ansi.Cyan(a.args)
		}
		str[1] = a.desc
		if a.shortcut != "" {
			str[1] += ansi.Gray(" (" + a.shortcut + ")")
		}
		tw.WriteCells(str...)
	}
	if _, err := tw.Flush(); err != nil {
		cli.InternalError("Failed to flush output while showing help: ", err)
	}
	s.Oprintf(ansi.CodeGray, "\nKlar v%s", cli.KlarVersion)
}

var ctrlCMessage = ansi.ColorSprintf(
	ansi.CodeYellow,
	"To exit, type %s, press %s, or press %s again.",
	ansi.Cyan("exit"), ansi.Cyan("Ctrl+D"), ansi.Cyan("Ctrl+C"),
)

var multilineEnabledMsg = ansi.ColorSprintf(
	ansi.CodeBrightGreen,
	"Multiline mode enabled. Press %s to disable. End a line with %s to send",
	ansi.Cyan("Ctrl+G"), ansi.Cyan("."),
)
var multilineDisabledMsg = ansi.ColorSprintf(ansi.CodeBrightGreen, "Multiline mode disabled.")
