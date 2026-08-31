package klon

import (
	"github.com/ProCode-Software/klar/internal/lsp/klarast"
	"github.com/ProCode-Software/klar/pkg/klon"
	"github.com/ProCode-Software/klar/pkg/klon/ast"
	"github.com/ProCode-Software/klar/pkg/lsp"
)

type Analyzer struct {
	doc         *ast.Document
	Diagnostics []*klon.Error
}

func (a *Analyzer) Analyze() {
}

func GetColors(doc *ast.Document) (colors []lsp.Color) {
	ast.Walk(doc, func(node ast.Node, _ int) ast.StopCode {
		str, ok := node.(*ast.String)
		if !ok || str.Raw == "" || !klarast.HexColorRegex.MatchString(str.Raw) {
			return ast.ContinueWalk
		}
		if color := klarast.ParseHex(str.Raw); color != nil {
			colors = append(colors, *color)
		}
		return ast.ContinueWalk
	})
	return colors
}
