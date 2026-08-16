package lsp

import (
	"io"
	"log/slog"
	"os"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/util"
	"github.com/ProCode-Software/klar/pkg/argparse"
)

type Server struct {
	in  io.Reader // Usually stdin
	out io.Writer // Usually stdout
	*slog.Logger
}

func Main() {
	l, err := util.SetLogger(Flags.Flag("verbose").Bool(), false)
	if err != nil {
		cli.Failure("Failed to enable logger:", err)
	}
	s := Server{
		in: os.Stdin, out: os.Stdout,
		Logger: l,
	}
	s.Listen()
}

var Flags = argparse.NewParser().
	BoolFlag("verbose", "Enable verbose logging", false, "v")
