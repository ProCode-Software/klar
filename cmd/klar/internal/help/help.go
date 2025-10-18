package help

import (
	"github.com/ProCode-Software/klar/internal/cli/argparse"
	"github.com/ProCode-Software/klar/internal/command"
)

func Run(c *command.Runner) {
}

var Flags = argparse.NewParser("[command]").
	BoolFlag("docs", "Show online documentation for the command", false, "d")

var LongDescription = `Shows details about a Klar CLI command or topic, including
its usage and available options. This includes the 'klar help' command itself, which displays this message.

Using 'klar help command' is equivalent to 'klar command --help'. If no command is passed, it shows the list of CLI commands and options.

When the '--docs' or '-d' flag is passed, the full documentation for the topic is opened in the browser.`