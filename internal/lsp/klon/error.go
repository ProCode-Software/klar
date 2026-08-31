package klon

import (
	"regexp"

	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/klon/klonerrs"
	"github.com/ProCode-Software/klar/pkg/lsp"
)

var varAlreadyDeclaredRegex = regexp.MustCompile(`^Variable '(.+)' was already declared`)

// For errors like [klonerrs.ErrVarAlreadyDeclared] and [klonerrs.ErrVarCycle]
// to display a clickable range in the editor.
func GetRelatedErrorInfo(
	e *klon.Error, doc *ast.Document, toLocation func(ranges.Range) lsp.Location,
) []lsp.DiagnosticRelatedInformation {
	// TODO: Extract any ranges from the message. It's not a great method,
	// but Klon errors don't provide that much context (e.g. a param map).
	switch e.Code {
	case klonerrs.ErrVarAlreadyDeclared:
		// Extract the name from the error message
		groups := varAlreadyDeclaredRegex.FindStringSubmatch(e.Text)
		if groups == nil {
			return nil
		}
		return []lsp.DiagnosticRelatedInformation{{
			// Use the range of the original declaration in the AST
			// Limitation: Pos() is the range of the value, not where the name is.
			// That range is also used in the error message
			Location: toLocation(doc.Variables[groups[1]].Pos()),
			Message:  "It was originally declared here",
		}}
	case klonerrs.ErrVarCycle: // Not a parse error
	case klonerrs.ErrDuplicateField: // Not a parse error
	case klonerrs.ErrExpectedEOF:
	// Provide the location of the first value
	default:
		return nil
	}
	return nil
}
