package lsp

import (
	"fmt"
	"log/slog"

	"github.com/ProCode-Software/klar/pkg/lsp"
)

func (s *Server) didOpen(params lsp.DidOpenTextDocumentParams) {
	td := params.TextDocument
	uri := string(td.Uri)
	// TODO: I don't know how to handle file:// schemes
	s.fs.WriteFile(uri, []byte(td.Text))
	s.loadPackageFor(uri)
	s.Debug("Opened file", slog.String("uri", uri))
	s.compileModule(s.fs.Files[uri].ModulePath)
}

var zeroRange = lsp.Range{lsp.Position{0, 0}, lsp.Position{0, 0}}

func (s *Server) didChange(params lsp.DidChangeTextDocumentParams) {
	td := params.TextDocument
	uri := string(td.Uri)
	for _, change := range params.ContentChanges {
		// In our capabilities (see [Server.getCapabilities]), we said we
		// support full document changes
		//
		// TODO: The value of the change union will be the partial change, only due
		// to a bug with [rpc.Union]. Only Text (not Range) will be populated.
		var text string
		if change.Curr() == 0 {
			partial := change.A()
			if partial.Range != zeroRange {
				panic(fmt.Sprintf(
					"textDocument/didChange: change must not be partial: got range %+v",
					partial.Range,
				))
			}
			text = change.A().Text
		} else {
			text = change.B().Text
		}
		s.fs.WriteFile(uri, []byte(text))
	}
	s.loadPackageFor(uri)
	s.compileModule(s.fs.Files[uri].ModulePath)
	// It would be terrible to log each change
}
