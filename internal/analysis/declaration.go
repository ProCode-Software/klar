package analysis

import "github.com/ProCode-Software/klar/internal/ast"

type DeclarationInfo struct {
	file *Context
	node ast.Statement
}

func (c *Checker) declareTopLevelObject(name string, obj *Object, info *DeclarationInfo) {
	c.moduleDecls[obj] = info
	obj.order = uint32(len(c.moduleDecls))
}

func (c *Checker) declare(ctx *Context, obj *Object, name string, file FileID) {
	if obj.name != "_" {
		
	}
}