package analysis

import (
	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/ast/typed"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/internal/runtime"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/internal/types"
)

var defaultRange ranges.Range

type (
	Type    = types.Type
	context = *runtime.Context
)

// A Checker type-checks a [ast.Program].
type Checker struct {
	Errors  []errors.KlarError
	Exports map[string]runtime.Exportable
	Program *ast.Program
	OnError func(err errors.KlarError)

	FilePath string
	Target   target.Double

	typeDepMode int
}

// NewChecker returns a new Checker for program.
func NewChecker(program *ast.Program) *Checker {
	runtime.RuntimeContexts = make(runtime.ContextMap, 1)
	return &Checker{
		Program: program,
	}
}

func (c *Checker) Error(err errors.KlarError) {
	switch newErr := err.(type) {
	case errors.ParseError:
		newErr.File = c.FilePath
		err = newErr
	case errors.TypeError:
		newErr.File = c.FilePath
		err = newErr
	case errors.ReferenceError:
		newErr.File = c.FilePath
		err = newErr
	case errors.Warning:
		newErr.File = c.FilePath
		err = newErr
	}
	c.Errors = append(c.Errors, err)
	if c.OnError != nil {
		c.OnError(err)
	}
}

func (c *Checker) toTypedExports(
	exports []*ast.PublicDeclaration, asStmt []ast.Statement, ctx context,
) []typed.Declaration {
	items := make([]typed.Declaration, len(exports))
	typedCtx, returns := c.CheckContext(ctx, asStmt)
	// Error if there are return statements (because top level)
	for _, ret := range returns {
		c.Error(errors.Node(errors.ErrReturnOutsideFunc, ret.Node))
	}

	for _, funct := range typedCtx.Functions {
		items = append(items, funct)
	}
	for _, typ := range typedCtx.Types {
		items = append(items, typ)
	}
	for _, vari := range typedCtx.Statements {
		items = append(items, vari.(*typed.VariableDecl))
	}
	return items
}

func (c *Checker) CheckProgram() *typed.Program {
	rootCtx := runtime.NewContext(-1)
	var (
		foundDec    bool
		imports     []*ast.ImportStatement
		exports     []*ast.PublicDeclaration
		attrs       []*ast.Attribute
		exportStmts []ast.Statement
		others      []ast.Statement
	)
	for _, stmt := range c.Program.Body {
		switch stmt := stmt.(type) {
		// Resolve all modules before type-checking so they can be referenced
		// by the current module.
		case *ast.ImportStatement:
			imports = append(imports, stmt)
			if foundDec {
				c.Error(errors.Node(errors.ErrImportsGoFirst, stmt))
			}
		case *ast.PublicDeclaration:
			exports = append(exports, stmt)
			exportStmts = append(exportStmts, stmt.Declaration)
			foundDec = true
		case *ast.Attribute:
			attrs = append(attrs, stmt)
		default:
			others = append(others, stmt)
			foundDec = true
		}
	}
	typedExports := c.toTypedExports(exports, exportStmts, rootCtx)

	typedCtx, returns := c.CheckContext(rootCtx, c.Program.Body)
	// No returns outside top level
	for _, ret := range returns {
		c.Error(errors.Node(errors.ErrReturnOutsideFunc, ret.Node))
	}
	return &typed.Program{
		Context:  *typedCtx,
		Imports:  imports,
		Exports:  typedExports,
		BaseNode: c.Program.BaseNode,
	}
}

func (c *Checker) checkRedeclared(ok bool, ctx context, rang ranges.Range, name string) {
	if ok || ctx.TypeDeclarations[name] == nil {
		return
	}
	lastPos := ctx.TypeDeclarations[name].Position
	c.Error(errors.Redeclared(name, "Type", lastPos, rang))
}

func (c *Checker) CheckContext(ctx context, body []ast.Statement) (*typed.Context, []Return) {
	var (
		// Sort in a specific order so normal statements can reference types, etc.
		// The order is:
		//	1. types (each sorted by references)
		//	2. functions (each sorted by references)
		//  3. normal statements
		typs      []ast.TypeDeclaration
		funcs     []*ast.FunctionDeclaration
		funcAlias []*ast.FuncAliasDeclaration
		stmts     = make([]ast.Statement, 0, len(body))
		typeNames = make(map[string]ast.TypeDeclaration)
	)
	// Group each statement
	for _, dec := range body {
		switch dec := dec.(type) {
		case ast.TypeDeclaration:
			name := dec.Name()
			if last, exists := typeNames[name]; exists {
				c.Error(errors.Redeclared(name, "Type", last.GetRange(), dec.GetRange()))
				continue
			}
			typeNames[name] = dec
			typs = append(typs, dec)
		case *ast.FunctionDeclaration:
			funcs = append(funcs, dec)
		case *ast.FuncAliasDeclaration:
			funcAlias = append(funcAlias, dec)
		default:
			stmts = append(stmts, dec)
		}
	}
	// Sort the type declarations in dependency order. Types that reference other
	// types are declared last.
	sortedTypeNames := c.SortTypes(typeNames, ctx)
	for _, name := range sortedTypeNames {
		decl, ok := typeNames[name]
		if !ok {
			// The type will be resolved when it is parsed. It may be in another context
			continue
		}
		var val types.Type
		switch decl := decl.(type) {
		case *ast.StructDeclaration:
			val = c.ParseStruct(decl, ctx)
		case *ast.InterfaceDeclaration:
			val = c.ParseInterface(decl, ctx)
		case *ast.EnumDeclaration:
			val = c.ParseEnum(decl, ctx)
		case *ast.TypeAliasDeclaration:
			val = c.ParseType(decl.Type, ctx)
		}
		ctx.DeclareType(name, val, decl.GetRange())
	}
	// Declare functions and methods next
	typedFuncs := make([]*typed.FunctionDecl, len(funcs))
	for i, decl := range funcs {
		typedFuncs[i] = c.checkFuncDecl(decl, ctx)
	}
	// Normal statements
	typedStmts, returns := c.CheckStatements(stmts, ctx)
	return &typed.Context{
		Types:      nil,
		Functions:  typedFuncs,
		Statements: typedStmts,
	}, returns
}

/* 		case *ast.EnumDeclaration:
   			id = dec.Identifier
   			ok = ctx.DeclareType(id, c.parseEnum(dec), pos)
   		// Types don't have to be declared before they can be used.
   		// in structs and type aliases. No recursive types in aliases
   		case *ast.TypeAliasDeclaration:
   			ok = ctx.DeclareType(dec.Identifier, nil, pos)
   			id = dec.Identifier
   			alias = append(alias, dec)
   		// Structs and interfaces may recursively reference themselves with
   		// limitations.
   		case *ast.StructDeclaration, *ast.InterfaceDeclaration:
   			d := dec.(ast.TypeDeclaration)
   			id = d.Name()
   			ok = ctx.DeclareType(id, nil, pos)
   			intfs = append(intfs, d) */
