package analysis

import "github.com/ProCode-Software/klar/internal/ast"

type Info struct {
	Expressions map[ast.Expression]*Expr
	Tracked     map[*Object]TrackedInfo
	// Keys may be [*ast.Program], [*ast.Block] or [ast.Statement]
	Blocks    map[ast.Node]*stmtContext // Node where each context begins
	InitOrder []*Object                 // For top-level objects
}

func (c *Checker) recordBlock(node ast.Node, sctx *stmtContext) {
	c.Module.Info.Blocks[node] = sctx
}
