package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/runtime"
	typespkg "github.com/ProCode-Software/klar/internal/types"
)

var defaultPos lexer.Position

type (
	Type    = typespkg.Type
	Context = runtime.Context
)

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
	c.Errors = append(c.Errors, err)
	if c.OnError != nil {
		c.OnError(err)
	}
}

func (c *Checker) InferType(expr ast.Expression) Type {
	if expr == nil {
		return nil
	}
	return nil
}

func (c *Checker) ParseType(typ ast.Type, ctx *Context) Type {
	if typ == nil {
		return nil
	}
	return nil
}

func (c *Checker) CheckCompatible(t1, t2 Type) Type {
	if t1 == t2 {
		return t1
	} else {
		return typespkg.ErrorType
	}
}

func (c *Checker) CheckProgram() {
	rootCtx := runtime.NewContext(-1)
	c.Check(rootCtx, &c.Program.Body)
}

func (c *Checker) checkRedeclared(ok bool, ctx *Context, rang ranges.Range, name string) {
	if ok || ctx.TypeDeclarations[name] == nil {
		return
	}
	lastPos := ctx.TypeDeclarations[name].Position
	c.Error(errors.Redeclared(name, "Type", lastPos, rang))
}

func (c *Checker) Check(ctx *Context, body *[]ast.Statement) {
	var (
		foundDec bool
		// Sort each statement so normal statements can reference functions before
		// they are declared. Same thing for functions referencing types and for
		// structs/interfaces referencing type aliases
		types []ast.TypeDeclaration
		attrs []ast.Attribute
		funcs []ast.FunctionDeclaration
		stmts = make([]ast.Statement, 0, len(*body))
		// Only at top-level
		imports []ast.ImportStatement
		exports []ast.Publicizable
	)
	for _, dec := range *body {
		var (
			isImport, ok bool
			id           string
			pos          = dec.Base().Range
		)
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
		// Declare enums first since they don't depend on other types
		case ast.EnumDeclaration:
			ok = ctx.DeclareType(dec.Identifier, c.parseEnum(dec), pos)
			id = dec.Identifier
		// Types don't have to be declared before they can be used.
		// For structs and type aliases
		case ast.TypeAliasDeclaration:
			ok = ctx.DeclareType(dec.Identifier, nil, pos)
			id = dec.Identifier
			types = append(types, dec)
		case ast.StructDeclaration:
			ok = ctx.DeclareType(dec.Identifier, nil, pos)
			id = dec.Identifier
			types = append(types, dec)
		case ast.FunctionDeclaration:
			funcs = append(funcs, dec)
		case ast.Attribute:
			attrs = append(attrs, dec)
		default:
			stmts = append(stmts, dec)
		}
		if id != "" {
			c.checkRedeclared(ok, ctx, pos, id)
		}
		if !ctx.IsRoot() {
			continue
		}
		if dec, ok := dec.(ast.Publicizable); ok && dec.IsPublic() {
			exports = append(exports, dec)
		}
		if !isImport {
			foundDec = true
		}
	}

	// Types don't have to be declared before they can be used
	/* for _, t := range types {

	} */
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
