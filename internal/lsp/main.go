package lsp

import (
	"os"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/util"
	"github.com/ProCode-Software/klar/pkg/argparse"
)

func Main() {
	l, err := util.SetLogger(Flags.Flag("verbose").Bool(), false)
	if err != nil {
		cli.Failure("Failed to enable logger:", err)
	}
	s := NewServer(os.Stdin, os.Stdout, l)
	s.Listen()
}

var Flags = argparse.NewParser().
	BoolFlag("verbose", "Enable verbose logging", false, "v")
