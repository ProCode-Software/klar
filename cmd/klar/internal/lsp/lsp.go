package lsp

import (
	"os"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/lsp"
	"github.com/ProCode-Software/klar/internal/util"
	"github.com/ProCode-Software/klar/pkg/argparse"
	"golang.org/x/term"
)

var Flags = lsp.Flags

func Run(*argparse.Parser) {
	width, _, err := term.GetSize(int(os.Stderr.Fd()))
	if err != nil {
		width = 80
	}
	// Show a message if the user manually runs the command
	if term.IsTerminal(int(os.Stdin.Fd())) {
		ansi.ColorFprintln(os.Stderr, ansi.CodeBold, "Hello human! 👋\n")
		util.Wrap(description, util.WrapAllWriter(os.Stderr), width, width, 0)
		ansi.TagFprintfln(os.Stderr, "\n\n<y!>If you have run 'klar lsp' manually, you can press <c!>Ctrl+C</c!> to exit.</y!>")
	}
	lsp.Main()
}

const LongDescription = "Note: " + description

const description = `This command is intended for programmatic use by editors. Humans won't need to manually run this. For information about setting up your IDE for Klar development, see [docs url].

The Klar Language Server (KlarLS) is an implementation of the Language Server Protocol (LSP) for the Klar programming language. Editors that support the LSP protocol can start KlarLS by running 'klar lsp' with the Klar CLI.`
