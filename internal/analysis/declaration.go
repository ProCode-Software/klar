package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
)

type DeclarationInfo struct {
	file *Context
	node ast.Statement
}

func (c *Checker) declareTopLevelObject(obj *Object, info *DeclarationInfo) {
	c.declare(c.rootContext, obj)
	c.moduleDecls[obj] = info
	obj.order = uint32(len(c.moduleDecls))
}

func (c *Checker) declare(ctx *Context, obj *Object, flags ...Flag) {
	if obj.name == "_" {
		return
	}
	if existing := ctx.Declare(obj, flags...); existing != nil {
		// Declared already
		err := errors.Range(errors.ErrRedeclaredVar, obj.rang)
		err.SetParam("existing", existing.rang)
		c.FileError(err, obj.file)
	}
}
