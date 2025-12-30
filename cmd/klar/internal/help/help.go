// TODO: incomplete
package help

import (
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/pkg/argparse"
	"github.com/ProCode-Software/klar/internal/command"
)

// Existing CLI commands already handled in main.go
func Run(c *command.Runner) {
	if len(c.Args) < 1 {
		cli.Failure("Expected a command or topic name when '-d' flag is used")
	}
	name := c.ArgByName("command")
	topic, ok := Topics[name]
	switch {
	case ok:
	case c.Flag("d").Bool():
		// Show documentation online
		cli.Eprintf("Searching documentation for %s on %s\n",
			ansi.Cyan(name), ansi.Magenta("site.url"),
		)
	default:
		cli.Error("Can't find a CLI command or topic named ", ansi.Cyan(name))
		cli.HintIndent(
			"To search online documentation, use the " + ansi.Cyan("-d") + " flag.",
		)
		cli.Exit(1)
	}
	_ = topic
}

var Flags = argparse.NewParser("[command]").
	BoolFlag("docs", "Show online documentation for the command", false, "d")

var LongDescription = `Shows details about a Klar CLI command or topic, including
its usage and available options. This includes the 'klar help' command itself, which displays this message.

Using 'klar help command' is equivalent to 'klar command --help'. If no command is passed, it shows the list of CLI commands and options.

When the '--docs' or '-d' flag is passed, the full documentation for the topic is opened in the browser.`
