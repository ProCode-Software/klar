package lsp

import (
	"log/slog"

	"github.com/ProCode-Software/klar/pkg/lsp"
)

func (s *Server) didOpen(params lsp.DidOpenTextDocumentParams) {
	td := params.TextDocument
	// TODO: I don't know how to handle file:// schemes
	s.fs.WriteFile(string(td.Uri), []byte(td.Text))
	s.Debug("Opened file", slog.String("uri", string(td.Uri)))
}

func (s *Server) didChange(params lsp.DidChangeTextDocumentParams) {
	td := params.TextDocument
	for _, change := range params.ContentChanges {
		if change.Value != 1 {
			// In our capabilities (see [Server.getCapabilities]), we said we
			// support full document changes
			panic("textDocument/didChange: expected the whole document to be sent")
		}
		change := change.B()
		s.fs.WriteFile(string(td.Uri), []byte(change.Text))
	}
	// It would be terrible to log each change
}
