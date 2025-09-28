package repl

import (
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/version"
)

func action(tw *cli.TabWriter, opts ...string) {
	str := make([]string, 2)
	str[0] = ansi.Yellow(opts[0]) // Name
	if len(opts) > 3 {            // Argument
		str[0] += ansi.Cyan(" " + opts[1])
		opts = opts[1:]
	}
	str[1] = opts[1]   // Description
	if len(opts) > 2 { // Shortcut
		str[1] += ansi.Gray(" (" + opts[2] + ")")
	}
	tw.Write(str...)
}

func (s *Session) PrintHelp() {
	s.Printf(ansi.CodeBold, "Available REPL commands%s", ansi.Dim(":"))
	tw := cli.NewTabWriterOutput(s.Stderr())
	tw.Margin = 4
	action(tw, "exit", "Exit the REPL", "Ctrl+D")
	action(tw, "help", "Display this help message")
	action(tw, "load <file>", "Load and evaluate a Klar file in the REPL session")
	action(tw, "multiline", "Toggle multiline editing mode", "Ctrl+M")
	action(tw, "save <file>",
		"Save all successfully evaluated commands in this REPL session to a file", "Ctrl+S")
	tw.Flush()
	s.Printf(ansi.CodeGray, "\nKlar v%s", version.KlarVersion)
}

var ctrlCMessage = ansi.Sprintf(ansi.CodeYellow,
	"To exit, type %s, press %s, or press %s again.",
	ansi.Cyan("exit"), ansi.Cyan("Ctrl+D"), ansi.Cyan("Ctrl+C"),
)
var multilineEnabledMsg = ansi.Sprintf(ansi.CodeGreen,
	"Multiline mode enabled. Press %s to disable.", ansi.Cyan("Ctrl+M"),
)
var multilineDisabledMsg = ansi.Sprintf(ansi.CodeGreen, "Multiline mode disabled.")
