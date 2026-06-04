package analysis

import (
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

// If stopParsing is passed to panic, the checker will immediately stop parsing.
type stopChecker struct{}

func (c *Checker) error(err *klarerrs.Error) *klarerrs.Error {
	c.Errors = append(c.Errors, err)
	c.module.Flags |= ModuleWithErrors
	if c.Options.Error != nil {
		c.Options.Error(err)
	}
	if mx := c.Options.MaxErrors; mx > 0 && len(c.Errors) >= mx {
		c.Errors = append(c.Errors, klarerrs.TooManyErrors())
		panic(stopChecker{})
	}
	return err
}

func (c *Checker) fileError(err *klarerrs.Error, fid FileID) {
	file := c.module.JoinFilePath(c.module.fileID[fid])
	err.File = file
	c.error(err)
}

func redeclaredError(new, old *Object, topLevel bool) *klarerrs.Error {
	// TODO
	err := klarerrs.Range(klarerrs.ErrRedeclared, new.rang)
	err.Details = append(err.Details, klarerrs.Detail{
		File:    old.FilePath(),
		Range:   old.Range(),
		Message: klarerrs.Quote(old.name) + " was originally declared here",
	})
	err.Label = klarerrs.Quote(old.name) + " already exists"
	err.SetParam("oldKind", kindOf(old.typ))
	err.SetParam("newKind", kindOf(new.typ))
	err.SetParam("name", old.name)
	return err
}

func kindOf(typ Type) string {
	switch typ := typ.(type) {
	case nil:
		return ""
	default:
		_ = typ
	}
	return ""
}

func objectError(code klarerrs.Code, obj *Object) *klarerrs.Error {
	err := &klarerrs.Error{
		Range: obj.rang,
		File:  obj.FilePath(),
		Code:  code,
	}
	return err
}

func typeMismatch(exp, got Type, gotRange ranges.Range) *klarerrs.Error {
	err := klarerrs.Range(klarerrs.ErrTypeMismatch, gotRange)
	err.Label = "This should have type " + klarerrs.Quote(TypeToString(exp))
	err.SetParam("expected", TypeToString(exp))
	err.SetParam("got", TypeToString(got))
	return err
}
