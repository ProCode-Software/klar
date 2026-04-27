package analysis

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/errors"
)

// If stopParsing is passed to panic, the checker will immediately stop parsing.
type stopChecker struct{}

func (c *Checker) error(err errors.CompileError) errors.CompileError {
	c.Errors = append(c.Errors, err)
	if c.Options.Error != nil {
		c.Options.Error(err)
	}
	if mx := c.Options.MaxErrors; mx > 0 && len(c.Errors) >= mx {
		c.Errors = append(c.Errors, errors.TooManyErrors())
		panic(stopChecker{})
	}
	return err
}

func (c *Checker) fileError(err errors.CompileError, fid FileID) {
	file := c.module.JoinFilePath(c.module.fileID[fid])
	switch err := err.(type) {
	case *errors.ParseError:
		err.File = file
	case *errors.TypeError:
		err.File = file
	case *errors.ModuleError:
		err.File = file
	case *errors.ReferenceError:
		err.File = file
	case *errors.Warning:
		err.File = file
	default:
		panic(fmt.Sprintf("unhandled error type %T", err))
	}
	c.error(err)
}

func (c *Checker) redeclaredError() {
	// TODO
}

func objectError[T errors.CompileError](code errors.ErrorCode, obj *Object) T {
	var x T
	switch errors.CompileError(x).(type) {
	case *errors.ParseError:
		err := &errors.ParseError{}
		err.Range = obj.rang
		err.File = obj.FilePath()
		err.ErrorCode = code
		return errors.CompileError(err).(T)
	case *errors.TypeError:
		err := &errors.TypeError{}
		err.Range = obj.rang
		err.File = obj.FilePath()
		err.ErrorCode = code
		return errors.CompileError(err).(T)
	default:
		panic("unhandled error type")
	}
	// return nil
}
