package run

import (
	"os"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/pkg/argparse"
)

func Run(r *argparse.Parser) {
	input := r.ArgByName("input")
	if input != "" {
		if _, err := os.Stat(input); err != nil && os.Args[1] != "run" && input[0] != '.' {
			// Run via `klar ./file` rather than `klar run ./file`. Show
			// a more helpful error
			cli.ErrNotFound(input, "a command named")
		}
	}
}

const LongDescription = ``
