package lsp

import (
	"cmp"
	"log/slog"
	"strings"

	"github.com/ProCode-Software/klar/internal/klarerrs"
	klonls "github.com/ProCode-Software/klar/internal/lsp/klon"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon"
	"github.com/ProCode-Software/klar/pkg/lsp"
	"github.com/ProCode-Software/klar/pkg/lsp/rpc"
)

func (s *Server) documentDiagnostic(id rpc.ID, params lsp.DocumentDiagnosticParams) {
	path := StripScheme(params.TextDocument.Uri)
	diags := s.diagnosticsForFile(path)
	if len(diags) > 0 && s.logEnabled() {
		var firstMsg string
		if first := diags[0]; first.Message.Curr() == 0 {
			firstMsg = first.Message.A()
		} else {
			firstMsg = first.Message.B().Value
		}
		s.Error(
			"File has errors", slog.String("path", path),
			slog.Int("count", len(diags)), slog.String("first", firstMsg),
		)
	}
	// Response: [lsp.DocumentDiagnosticReport]
	// type DocumentDiagnosticReport = [lsp.RelatedFullDocumentDiagnosticReport]
	//	| [lsp.RelatedUnchangedDocumentDiagnosticReport]
	// See https://microsoft.github.io/language-server-protocol/specifications/lsp/3.18/specification/#textDocument_diagnostic
	s.sendResponse(lsp.RelatedFullDocumentDiagnosticReport{
		Kind:  string(lsp.DocumentDiagnosticReportFull),
		Items: diags,
		// Diagnostics for dependencies and other files in the module.
		// Only applicable to Klar files.
		RelatedDocuments: s.getRelatedDiagnostics(path),
	}, id)
}

func (s *Server) diagnosticsForFile(path string) (diags []*lsp.Diagnostic) {
	file, ok := s.fs.Files[path]
	switch {
	case !ok:
		return nil
	case file.IsKlar():
		diags = make([]*lsp.Diagnostic, len(file.Klar.Diagnostics))
		for i, err := range file.Klar.Diagnostics {
			diags[i] = s.klarErrorToDiagnostic(err)
		}
	default:
		diags = make([]*lsp.Diagnostic, len(file.Klon.Diagnostics))
		for i, err := range file.Klon.Diagnostics {
			diags[i] = s.klonErrorToDiagnostic(err, path)
		}
	}
	return diags
}

type relatedDiagostic = rpc.Union2[
	lsp.FullDocumentDiagnosticReport, lsp.UnchangedDocumentDiagnosticReport,
]

func (s *Server) getRelatedDiagnostics(filePath string) *map[lsp.DocumentURI]relatedDiagostic {
	file := s.fs.Files[filePath]
	if !file.IsKlar() {
		return nil
	}
	// TODO: I think depsWithDiags may be needed later. Otherwise, I would like to unset it.
	relatedDiags := file.Klar.Module.depsWithDiags
	if len(relatedDiags) == 0 {
		return nil
	}
	lspRelatedDiags := make(map[lsp.DocumentURI]relatedDiagostic, len(relatedDiags)-1)
	for depPath := range relatedDiags {
		if depPath == filePath {
			continue
		}
		if _, ok := s.fs.Files[depPath]; !ok {
			// Module not loaded in workspace. Won't make a difference to the user
			continue
		}
		lspRelatedDiags[lsp.DocumentURI("file://"+depPath)] = relatedDiagostic{
			lsp.FullDocumentDiagnosticReport{
				Kind:  string(lsp.DocumentDiagnosticReportFull),
				Items: s.diagnosticsForFile(depPath),
			},
		}
	}
	return &lspRelatedDiags
}

func (s *Server) klarErrorToDiagnostic(e *klarerrs.Error) *lsp.Diagnostic {
	sev := lsp.DiagnosticSeverityError
	if e.IsWarning() {
		sev = lsp.DiagnosticSeverityWarning
	}
	// The compiler only gives errors and warnings (LSP also has hint and info).
	// Future linters can give hints and info.

	// Message will include any label in parenthesis, as well as any hints
	var msg strings.Builder
	msg.WriteString(e.Message())
	// TODO: Reconsider if the label should be shown
	// It is needed for [klarerrs.ErrOperandTypeMismatch]
	if false && e.Label != "" {
		msg.WriteString(" (")
		msg.WriteString(e.Label)
		msg.WriteByte(')')
	}
	for _, hint := range e.Hints {
		msg.WriteString("\n\nHint: ")
		msg.WriteString(hint.Message)
	}

	diag := &lsp.Diagnostic{
		Source: "klar",
		Range:  s.toLSPRange(e.Range, e.File),
		// TODO: Consider using Markdown for messages
		Message:  rpc.Union2[string, lsp.MarkupContent]{msg.String()},
		Severity: &sev,
		Code:     &rpc.Union2[int, string]{e.Code.Format()},
	}
	// More specific source for the type of error
	switch e.Prefix() {
	case klarerrs.TypeErrorPrefix, klarerrs.ReferenceErrorPrefix:
		diag.Source = "klar/analysis"
	case klarerrs.SyntaxErrorPrefix:
		diag.Source = "klar/syntax"
	case klarerrs.ModuleErrorPrefix:
		diag.Source = "klar/module"
	}

	// Show a documentation link for the error code. In most editors, this makes
	// the code clickable.
	// TODO: Replace with documentation link. For now it displays the file where
	// it is defined in the Klar compiler.
	if basename, ok := prefixLinks[e.Prefix()]; ok {
		diag.CodeDescription = &lsp.CodeDescription{
			Href: "https://github.com/ProCode-Software/klar/tree/main/internal/klarerrs/" +
				basename + ".go",
		}
	}

	// Use diagnostic tags for deprecated/unused errors. This dims or
	// strikethroughs the range in the editor
	switch e.Code {
	case klarerrs.WarnUnused:
		diag.Tags = []lsp.DiagnosticTag{lsp.DiagnosticTagUnnecessary}
	case -182234: // TODO: Change once a deprecated code is added
		diag.Tags = []lsp.DiagnosticTag{lsp.DiagnosticTagDeprecated}
	}

	if len(e.Details) > 0 {
		diag.RelatedInformation = make([]lsp.DiagnosticRelatedInformation, len(e.Details))
		for i, det := range e.Details {
			// The file in the detail may be outside the project, and not loaded by the LSP
			var rang lsp.Range
			if _, ok := s.fs.Files[det.File]; ok || det.File == "" {
				rang = s.toLSPRange(det.Range, det.File)
			}
			diag.RelatedInformation[i] = lsp.DiagnosticRelatedInformation{
				Location: lsp.Location{
					// TODO: URI may be untitled (check if absolute)
					Uri:   lsp.DocumentURI("file://" + cmp.Or(det.File, "")),
					Range: rang,
				},
				Message: det.Message,
			}
		}
	}

	// rust-analyzer in VSCode provides links to the compiler message (with ANSI coloring)
	// as a client-side decoration
	// https://github.com/rust-lang/rust-analyzer/blob/master/editors/code/src/diagnostics.ts
	return diag
}

var prefixLinks = map[klarerrs.Code]lsp.URI{
	klarerrs.SyntaxErrorPrefix:         "syntax_error",
	klarerrs.NoPrefix:                  "unprefixed",
	klarerrs.WarningPrefix:             "warning",
	klarerrs.TypeErrorPrefix:           "type_error",
	klarerrs.ModuleErrorPrefix:         "module_error",
	klarerrs.ReferenceErrorPrefix:      "reference_error",
	klarerrs.ImplementationErrorPrefix: "implementation_error",
}

func (s *Server) klonErrorToDiagnostic(e *klon.Error, file string) *lsp.Diagnostic {
	doc := s.fs.Files[file].Klon.AST
	diag := &lsp.Diagnostic{
		Source:  "klon",
		Range:   s.toLSPRange(e.Range, file),
		Message: rpc.Union2[string, lsp.MarkupContent]{e.Text},
		// Codes from Klon are just numbers (not so useful)
		Code:     &rpc.Union2[int, string]{int(e.Code)},
		Severity: new(lsp.DiagnosticSeverityError),
		RelatedInformation: klonls.GetRelatedErrorInfo(
			e, doc, func(r ranges.Range) lsp.Location {
				return lsp.Location{
					Uri:   lsp.DocumentURI("file://" + file), // TODO: file may be untitled
					Range: s.toLSPRange(r, file),
				}
			},
		),
	}
	if e.Warning {
		diag.Severity = new(lsp.DiagnosticSeverityWarning)
	}
	return diag
}

func (s *Server) klonErrorRelatedInfo(e *klon.Error, file string) []lsp.DiagnosticRelatedInformation {
	return nil
}
