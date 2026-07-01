package analysis

import (
	"fmt"

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
	err.File = c.module.ResolveFilePath(fid)
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
	// TODO: finish implementing
	switch typ := typ.(type) {
	case *Variable:
		return "variable"
	case *Constant:
		return "constant"
	case *Function, *Overload:
		return "function"
	case *TypeName:
		if typ.Type == nil {
			return "type"
		}
		return typ.Kind().String()
	case nil:
		return ""
	default:
		_ = typ
	}
	return fmt.Sprintf("%T", typ)
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
	err.Label = "This has type " + klarerrs.Quote(got.String())
	err.Info = klarerrs.TypeErrorInfo{
		ExpectedType: exp.String(),
		GotType:      got.String(),
	}
	return err
}

func handlePanic() {
	r := recover()
	if _, ok := r.(stopChecker); !ok && r != nil {
		panic(r)
	}
}

// The returned Error's File is already set.
func cycleError(cycle []*Object) *klarerrs.Error {
	// Find the object with the earliest position in the file
	firstInSrcI, firstPos := 0, cycle[0].rang.Start
	for i, o := range cycle {
		pos := o.rang.Start
		if ranges.ComparePos(pos, firstPos) < 0 {
			firstInSrcI, firstPos = i, pos
		}
	}
	o := cycle[firstInSrcI]
	// If the object is an alias, mark it as valid to avoid later errors.
	tn, isTypeDecl := o.typ.(*TypeName)
	if isTypeDecl {
		if alias, ok := tn.Type.(*TypeAlias); ok {
			alias.resolved = InvalidType
			alias.Type = InvalidType
		}
	}

	err := klarerrs.Range(klarerrs.ErrDepCycle, o.rang)
	err.File = o.FilePath()
	err.Name = o.name
	if isTypeDecl {
		err.SetParam("type", true)
	}
	if len(cycle) == 1 {
		// Self-cycle. Report a better error message
		err.SetParam("self", true)
		err.Label = "This type depends on itself"
		return err
	}
	for i := range cycle {
		nextI := (firstInSrcI + i + 1) % len(cycle)
		nextObj := cycle[nextI]
		var lastInCycleMsg string
		if nextI == 0 {
			lastInCycleMsg = " in a cycle"
		}
		err.AddDetailf(
			o.FilePath(), o.rang, "%s depends on %s%s",
			klarerrs.Quote(o.name), klarerrs.Quote(nextObj.name), lastInCycleMsg,
		)
		o = nextObj
	}
	return err
}

func fieldNotFound(name string) *klarerrs.Error {
	return &klarerrs.Error{
		Code:  klarerrs.ErrFieldNotFound,
		Label: "Field " + quote(name) + " doesn't exist",
		Name:  name,
	}
	// A range will be added later by the caller of [Indexer.IndexDot]
}
