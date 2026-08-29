package lsp

import (
	"fmt"
	"log/slog"

	"github.com/ProCode-Software/klar/pkg/lsp"
)

func (s *Server) didOpen(params lsp.DidOpenTextDocumentParams) {
	td := params.TextDocument
	path := StripScheme(td.Uri)
	s.fs.WriteFile(path, []byte(td.Text))
	// Set the language to either Klar or Klon
	file := s.fs.Files[path]
	file.SetLanguage(td.LanguageId)

	// Compile
	if file.IsKlar() {
		s.loadPackageFor(path)
		// The module may have been typechecked as a dependency, so don't
		// recompile unless it has changed (didChange)
		if kf := file.Klar; kf.Module == nil || kf.Module.Module == nil {
			s.compileKlarModule(kf.ModulePath)
		}
	}
	s.Debug(
		"Opened file",
		slog.String("uri", string(td.Uri)),
		slog.String("language", string(td.LanguageId)),
	)
}

var zeroRange = lsp.Range{lsp.Position{0, 0}, lsp.Position{0, 0}}

func (s *Server) didChange(params lsp.DidChangeTextDocumentParams) {
	td := params.TextDocument
	path := StripScheme(td.Uri)
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
		s.fs.WriteFile(path, []byte(text))
	}
	if file := s.fs.Files[path]; file.IsKlar() {
		// s.loadPackageFor(path)
		s.compileKlarModule(file.Klar.ModulePath)
	} else {
		// TODO: Handle Klon changes
	}
	// It would be terrible to log each change
}
