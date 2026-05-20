package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/klarerrs"
)

// If stopParsing is passed to panic, the checker will immediately stop parsing.
type stopChecker struct{}

func (c *Checker) error(err *klarerrs.Error) *klarerrs.Error {
	c.Errors = append(c.Errors, err)
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
	switch err := err.(type) {
	case *klarerrs.Error:
		err.File = file
	case *klarerrs.TypeError:
		err.File = file
	case *klarerrs.ModuleError:
		err.File = file
	case *klarerrs.ReferenceError:
		err.File = file
	case *klarerrs.Warning:
		err.File = file
	default:
		panic(fmt.Sprintf("unhandled error type %T", err))
	}
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
	err.Label = klarerrs.Quote(old.name) + "already exists"
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

func objectError[T *klarerrs.Error](code klarerrs.Code, obj *Object) T {
	var x T
	switch *klarerrs.Error(x).(type) {
	case *klarerrs.Error:
		err := &klarerrs.Error{}
		err.Range = obj.rang
		err.File = obj.FilePath()
		err.Code = code
		return *klarerrs.Error(err).(T)
	case *klarerrs.TypeError:
		err := &klarerrs.TypeError{}
		err.Range = obj.rang
		err.File = obj.FilePath()
		err.Code = code
		return *klarerrs.Error(err).(T)
	default:
		panic("unhandled error type")
	}
}
