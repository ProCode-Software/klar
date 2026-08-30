package klon

import (
	"github.com/ProCode-Software/klar/pkg/klon"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
)

func Parse(b []byte) (*ast.Document, []*klon.Error) {
	return klon.Parse(b)
}

type Analyzer struct {
	doc         *ast.Document
	Diagnostics []*klon.Error
}

func (a *Analyzer) Analyze() {
}
