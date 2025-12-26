package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type DeclarationInfo struct {
	file *Context
	node ast.Statement
}

func (c *Checker) declareTopLevelObject(obj *Object, info *DeclarationInfo) {
	name := obj.name
	c.declare(c.rootContext, obj, name, )
	c.moduleDecls[obj] = info
	obj.order = uint32(len(c.moduleDecls))
}

func (c *Checker) declare(ctx *Context, obj *Object, name string, file FileID) {
	if obj.name == "" {
		return
	}
}