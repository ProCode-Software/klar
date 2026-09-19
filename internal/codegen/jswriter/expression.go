package jswriter

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

func (w *Writer) writeExpression(expr jsir.Expression) {
	switch expr := expr.(type) {
	default:
		panic(fmt.Sprintf("unhandled expression: %T", expr))
	}
}
