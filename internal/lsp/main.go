package lsp

import (
	"io"
	"log/slog"
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
	s := NewServer(os.Stdout, os.Stdout, l)
	s.Listen()
}

func NewServer(in io.Reader, out io.Writer, l *slog.Logger) *Server {
	return &Server{
		in: in, out: out,
		Logger: l,
		fs:     &FileSystem{},
		pkgs:   make(map[string]*Package),
	}
}

var Flags = argparse.NewParser().
	BoolFlag("verbose", "Enable verbose logging", false, "v")
