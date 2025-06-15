package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/runtime"
	"github.com/ProCode-Software/klar/internal/types"
)

var defaultPos lexer.Position

type (
	Type    = types.Type
	Context = runtime.Context
)

// A Checker type-checks a [ast.Program].
type Checker struct {
	Errors          []errors.KlarError
	Exports         map[string]runtime.Exportable
	Program         ast.Program
	OnError         func(err errors.KlarError)
	ContinueOnError bool

	typeDepMode int
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

func (c *Checker) CheckCompatible(t1, t2 Type) Type {
	if t1 == t2 {
		return t1
	} else {
		return types.InvalidType
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
		// they are declared. Same thing for functions referencing alias and for
		// structs/interfaces referencing type alias
		alias []ast.TypeAliasDeclaration
		attrs []ast.Attribute
		funcs []ast.FunctionDeclaration
		intfs []ast.TypeDeclaration // Structs and interfaces
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
		// Resolve all modules before type-checking so they can be referenced
		// by the current module.
		case ast.ImportStatement:
			imports = append(imports, dec)
			isImport = true
			if foundDec {
				c.Error(errors.Node(errors.ErrImportsGoFirst, dec))
			}
			continue
		// Declare enums first since they don't depend on other types
		case ast.EnumDeclaration:
			id = dec.Identifier
			ok = ctx.DeclareType(id, c.parseEnum(dec), pos)
		// Types don't have to be declared before they can be used.
		// in structs and type aliases. No recursive types in aliases
		case ast.TypeAliasDeclaration:
			ok = ctx.DeclareType(dec.Identifier, nil, pos)
			id = dec.Identifier
			alias = append(alias, dec)
		// Structs and interfaces may recursively reference themselves with
		// limitations.
		case ast.StructDeclaration, ast.InterfaceDeclaration:
			d := dec.(ast.TypeDeclaration)
			id = d.Name()
			ok = ctx.DeclareType(id, nil, pos)
			intfs = append(intfs, d)
		// Functions may redeclare themselves with different parameters/overloads
		case ast.FunctionDeclaration:
			funcs = append(funcs, dec)
		// Attributes attach to declarations
		// @target - sets the target runtime for a declaration
		// @deprecated - warn when referenced
		// @added - version when added
		// @external - external implementation
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
	deps := c.getTypeAliasDeps(alias, ctx) // Deps of aliases
	c.mergeStructDeps(deps, intfs, ctx)
	types, names, undef := sortTypeDecls(deps, alias, intfs)
	for i, t := range types {
		if t == nil {
			// Not defined
			name := names[i]
			in := undef[name]
			c.Error(errors.Undefined(
				errors.ErrTypeUndefined, name, traceUndefined(name, in),
			))
			continue
		}
		name := t.Name()
		// Skip type cycles, would already be set to error type
		if ctx.TypeDeclarations[name] != nil {
			continue
		}
		switch t := t.(type) {
		case ast.StructDeclaration:
		case ast.InterfaceDeclaration:
		case ast.TypeAliasDeclaration:
			ctx.SetType(name, c.ParseType(t.Type, ctx))
		}
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
