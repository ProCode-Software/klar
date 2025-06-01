package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/runtime"
	"github.com/ProCode-Software/klar/internal/types"
)

var defaultPos lexer.Position

type Type = types.Type

// A Checker type-checks a [ast.Program].
type Checker struct {
	Errors          []errors.KlarError
	Exports         map[string]runtime.Exportable
	Program         ast.Program
	OnError         func(err errors.KlarError)
	ContinueOnError bool
}

// NewChecker returns a new Checker for program.
func NewChecker(program ast.Program) *Checker {
	runtime.RuntimeContexts = make(runtime.ContextMap, 1)
	return &Checker{
		Program: program,
	}
}

func (c *Checker) Error(err errors.KlarError) {
	c.Errors = append([]errors.KlarError{err}, c.Errors...)
	if c.OnError != nil {
		c.OnError(err)
	}
}

func (c *Checker) InferType(expr ast.Expression) Type {
	return nil
}

func (c *Checker) CheckCompatible(t1, t2 Type) Type {
	if t1 == t2 {
		return t1
	} else {
		return types.ErrorType
	}
}

func (c *Checker) CheckProgram() {
	rootCtx := runtime.NewContext(-1)
	c.Check(rootCtx)
}

func (c *Checker) Check(ctx *runtime.Context) {
	var (
		foundDec bool
		// Sort each statement so normal statements can reference functions before
		// they are declared. Same thing for functions referencing types.
		types   []ast.TypeDeclaration
		funcs   []ast.FunctionDeclaration
		stmts   []ast.Statement
		imports []ast.ImportStatement
	)
	for _, dec := range c.Program.Body {
		var isImport bool
		switch dec := dec.(type) {
		// Imports are only parsed at the top-level, but they must go
		// before other declarations.
		case ast.ImportStatement:
			imports = append(imports, dec)
			isImport = true
			if foundDec {
				c.Error(errors.Node(errors.ErrImportsGoFirst, dec))
			}
			continue
		case ast.TypeDeclaration:
			types = append(types, dec)
		case ast.FunctionDeclaration:
			funcs = append(funcs, dec)
		default:
			stmts = append(stmts, dec)
		}
		if !isImport {
			foundDec = true
		}
	}
	/* switch dec := dec.(type) {
	case ast.VariableDeclaration:
		var typ Type
		if dec.ExplicitType == nil {
			typ = c.InferType(dec.Value)
		} else {
			typ = c.CheckCompatible(dec.ExplicitType, c.InferType(dec.Value))
		}
		ctx.Declare(dec.Identifier, dec.Constant, typ, dec.Base().Start)
	case ast.EnumDeclaration:
		// rootCtx.DeclareType(dec.)
	} */
}
