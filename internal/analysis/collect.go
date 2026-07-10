package analysis

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/ranges"
)

type methodInfo struct {
	decl  *ast.FunctionDeclaration
	alias *ast.FuncAliasDeclaration // If method alias and decl == nil
	obj   *Object                   // [*Overload] if function, or [*FunctionAlias] if alias
}

// collectTopLevelObjects collects all top-level objects from
// each program and declares placeholder objects in the module's context.
// Contents of each objects are not checked yet.
func (c *Checker) collectTopLevelObjects(
	files []string, fileContexts map[string]*Context,
) (methods map[string][]methodInfo, inits map[string][]*Object) {
	var (
		collector     = &stmtCollector{topLevel: true, ctx: c.module.Context}
		attrs         []*ast.Attribute
		topLevelFctx  *Context
		topLevelStmts []ast.Statement
	)
	for _, fileName := range files {
		var (
			program = c.Programs[fileName]
			fctx    = fileContexts[fileName]
			fid     = fctx.File
			// First statement after imports. Any import after this is misplaced.
			// An import will never be at program.Body[firstStmt].
			firstStmt, _  = fctx.getAttribute(firstStmtIndex).(int)
			allowTopLevel = fileName == "main.klar" || c.module.Flags.Has(SingleFileModule)
		)
		collector.fid = fid
		for _, stmt := range program.Body[firstStmt:] {
			var public bool
			if ps, ok := stmt.(*ast.PublicDeclaration); ok {
				public = true
				stmt = ps.Declaration
			}
			switch stmt := stmt.(type) {
			case *ast.ImportStatement:
				// Imports were already processed. Misplaced import
				err := klarerrs.Node(klarerrs.ErrImportsGoFirst, stmt)
				err.Label = "Put this import at the top of the file"
				c.fileError(err, fid)
				continue
			case *ast.Attribute:
				attrs = append(attrs, stmt)
				continue
			case *ast.FunctionDeclaration:
				c.declareFunc(stmt, collector, public, &attrs)
				continue
			case *ast.FuncAliasDeclaration:
				c.declareFuncAlias(stmt, collector, public, &attrs)
				continue
			case ast.TypeDeclaration:
				c.declareType(stmt, collector, public, &attrs)
				continue
			case *ast.VariableDeclaration:
				// TODO: some but not all var declarations are top level (can contain
				// function calls in values).
				// Make rules for declarations in main.klar files
				if fileName == "main.klar" {
					break
				}
				c.declareVars(stmt, collector, public, &attrs)
				continue

				// Top-level statements
				// ========
			case *ast.BadExpression:
				// Shouldn't be here. Invalid ASTs should't be typechecked.
				panic("typechecking invalid AST")
			case *ast.ExpressionStatement:
				// Allow unused values in REPL. Also, don't report an error here if top-level
				// statements are allowed. An error will be reported later while checking
				// top-level statements.
				if !allowTopLevel && !c.module.Flags.Has(REPLModule) &&
					!isAllowedAsStmt(stmt.Expression) {
					c.fileError(klarerrs.Node(klarerrs.ErrUnusedValue, stmt), fid)
					// Still have to check it
				}
			}
			if len(attrs) > 0 {
				// If we're here, the attributes weren't applied to a declaration.
				// For example, they were attached to a statement.
				err := klarerrs.Node(klarerrs.ErrInvalidAttributeTarget, stmt)
				err.Label = "Can't apply attributes to this statement"
				err.AddHighlight(
					"These attributes were applied to the statement",
					ranges.FromSlice(attrs),
				)
				c.fileError(err, fid)
				attrs = attrs[:0]
			}
			// Top-level statement: only allowed in main.klar or single-file modules
			if allowTopLevel {
				topLevelStmts = append(topLevelStmts, stmt)
				topLevelFctx = fctx
			} else {
				c.fileError(klarerrs.Node(klarerrs.ErrTopLevel, stmt), fid)
			}
		}
		if len(attrs) > 0 {
			// Attribute with no declaration after
			err := klarerrs.Slice(klarerrs.ErrNoDeclAfterAttr, attrs)
			err.Label = "Missing declaration after attribute"
			if len(attrs) > 1 {
				err.Label += "s"
			}
			c.fileError(err, fid)
			attrs = attrs[:0]
		}
	}
	// Ensure no top-level objects were shadowed by imports
	for _, fileName := range files {
		fctx := fileContexts[fileName]
		for _, imported := range fctx.SortedDecls() {
			name := imported.name
			topLevel := c.module.Context.Lookup(name)
			if topLevel == nil {
				continue // No error
			}
			// Only imports are in the file scope. One of these could possibly share a name:
			// - Namespace of normal import
			// - Alias of import
			// - An unqualified import object
			var namespace string
			if imported.Kind() == KindNamespace {
				// Provide the import path the namespace is from
				namespace = imported.module.ImportPathString()
			}
			err := klarerrs.Range(klarerrs.ErrImportShadow, imported.rang)
			err.Label = klarerrs.Quote(name) + " was already declared in the module"
			err.Params = klarerrs.ErrorParams{"name": name, "import": namespace}
			// Provide a detail from where the module object was declared
			err.Details = append(err.Details, klarerrs.Detail{
				File:    topLevel.FilePath(),
				Range:   topLevel.rang,
				Message: "It was already declared here",
			})
			c.fileError(err, imported.file)
		}
	}
	if len(topLevelStmts) > 0 {
		c.queue(func() { c.checkTopLevelStmts(topLevelStmts, topLevelFctx) }, true)
	}
	return collector.methods, collector.inits
}

// checkContextDecls typechecks all declarations in the
// given context, but not function bodies.
func (c *Checker) checkContextDecls(
	ctx *Context, methods map[string][]methodInfo, inits map[string][]*Object,
) {
	var (
		typeAliases []*Object // [*TypeName]
		nonTypes    []*Object // Variable/function declaration
		funcAliases []*Object // [*FunctionAlias]
	)
	// 1. Check new type declarations (not aliases)
	for _, obj := range ctx.SortedDecls() {
		switch obj.typ.(type) {
		case *TypeName:
			if _, ok := obj.info.node.(*ast.TypeAliasDeclaration); ok {
				typeAliases = append(typeAliases, obj)
			} else {
				c.checkDeclaration(obj)
			}
		case *FunctionAlias:
			funcAliases = append(funcAliases, obj)
		default:
			nonTypes = append(nonTypes, obj)
		}
	}
	// 2. Type aliases
	for _, obj := range typeAliases {
		c.checkDeclaration(obj)
	}
	// 3. Non-types (no methods)
	for _, obj := range nonTypes {
		c.checkDeclaration(obj)
	}
	// 4. Function aliases (no methods)
	for _, obj := range funcAliases {
		c.checkDeclaration(obj)
	}
	// 5. Methods and initializers: Associate methods/initializers with receiver types
	for _, typeName := range slices.Sorted(maps.Keys(methods)) {
		c.collectMethods(ctx, typeName, methods[typeName])
	}
	for _, typeName := range slices.Sorted(maps.Keys(inits)) {
		c.collectInitializers(ctx, typeName, inits[typeName])
	}
}

type stmtCollector struct {
	ctx       *Context // Where the objects will be declared to
	fid       FileID
	inits     map[string][]*Object
	methods   map[string][]methodInfo
	topLevel  bool
	currOrder uint32
}

func (sc *stmtCollector) declareMethod(s *ast.Identifier, info methodInfo) {
	if sc.methods == nil {
		sc.methods = make(map[string][]methodInfo)
	}
	sc.methods[s.Name] = append(sc.methods[s.Name], info)
}

func (sc *stmtCollector) declareInitializer(ov *Object) {
	if sc.inits == nil {
		sc.inits = make(map[string][]*Object)
	}
	sc.inits[ov.name] = append(sc.inits[ov.name], ov)
}

func (sc *stmtCollector) nextOrder() uint32 {
	sc.currOrder++
	return sc.currOrder - 1
}

// declareFunc declares an [Overload] object for the given declaration. If the
// declaration is a method, it is inserted into sc.methods. If the declaration
// is an initializer, it is inserted into sc.inits. The overload is
// associated with its parent [Function]. Signatures aren't checked yet.
func (c *Checker) declareFunc(stmt *ast.FunctionDeclaration, sc *stmtCollector,
	public bool, attrs *[]*ast.Attribute,
) {
	name := stmt.Identifier.Name
	ov := NewObject(name, sc.fid, stmt.GetRange(), c.module, &Overload{})
	ov.typ.(*Overload).Object = ov
	ov.public = public

	var par *Object
	var isInit bool
	if stmt.SelfType != nil {
		// Method
		sc.declareMethod(stmt.SelfType, methodInfo{decl: stmt, obj: ov})
	} else {
		par, isInit = c.getOverloadParent(name, stmt, sc)
	}
	switch {
	case isInit:
		// If `name` refers to a type, this overload is an initializer.
		sc.declareInitializer(ov)
	default:
		// New overload (may be the first)
		parFn := par.typ.(*Function)
		parFn.Overloads = append(parFn.Overloads, ov.typ.(*Overload))
		// If at least 1 overload is public, the entire function is public
		if public {
			par.public = true
		}
	case par == nil:
		// This is a method, or the object with same name isn't a function.
		// For the latter case, an error was already reported
	}
	// No kind is declared into the context. For overloads, their parent has
	// already been declared.
	ov.info = &DeclarationInfo{node: stmt}
	ov.order = sc.nextOrder()
	c.declareWithInfo(ov, sc.ctx, attrs, false)
}

// declareFuncAlias declares a function alias object for the given declaration. If
// the declaration is a method, it is inserted into sc.methods. Alias targets
// aren't checked yet.
func (c *Checker) declareFuncAlias(stmt *ast.FuncAliasDeclaration, sc *stmtCollector,
	public bool, attrs *[]*ast.Attribute,
) {
	// TODO: Currently, function aliases can't be used as overloads.
	// Example that is not allowed:
	//
	// 	type Person
	// 	func Person = newPerson // newPerson may have overloads, which may get confusing

	// Will be resolved later
	obj := NewObject(
		stmt.Identifier.Name,
		sc.fid, stmt.Range, c.module, &FunctionAlias{},
	)
	obj.public = public
	if stmt.Struct != nil {
		// Method alias
		sc.declareMethod(stmt.Struct, methodInfo{alias: stmt, obj: obj})
	}
	// Both methods and normal aliases have their info recorded.
	obj.info = &DeclarationInfo{node: stmt}
	obj.order = sc.nextOrder()
	c.declareWithInfo(obj, sc.ctx, attrs, stmt.Struct == nil)
}

// declareType declares a [TypeName] object for the given type declaration.
// Underlying types are not checked yet.
func (c *Checker) declareType(stmt ast.TypeDeclaration, sc *stmtCollector,
	public bool, attrs *[]*ast.Attribute,
) {
	name := stmt.Name()
	obj := NewObject(name, sc.fid, stmt.GetRange(), c.module, &TypeName{nil, name})

	var hadInit bool
	if maybeFnObj := sc.ctx.Lookup(name); maybeFnObj != nil {
		var fn *Function
		if fn, hadInit = maybeFnObj.typ.(*Function); hadInit {
			// The initializer was declared earlier than this type.
			// Change the object in the context to this type.
			sc.ctx.Declarations[name] = obj
			// Move the existing initializers to sc.inits
			for _, ov := range fn.Overloads {
				sc.declareInitializer(ov.Object)
			}
			fn.Overloads = fn.Overloads[:0]
		}
	}

	obj.info = &DeclarationInfo{node: stmt}
	obj.order = sc.nextOrder()
	c.declareWithInfo(obj, sc.ctx, attrs, !hadInit)
	obj.public = public
}

// getOverloadParent finds the [*Function] associated with f's name, which
// overloads can be added to. The function is declared if it doesn't exist.
// An error is reported if the object with f's name exists and is neither
// a [*Function] nor a [*TypeName]. If it is a [*TypeName], isInit will
// be true.
func (c *Checker) getOverloadParent(
	name string, node ast.Statement, sc *stmtCollector,
) (par *Object, isInit bool) {
	// par is the function we're adding overloads to
	// TODO: Implement adding module-scoped initializers for builtins
	par = sc.ctx.Lookup(name)
	if par == nil {
		// If this is the first overload, declare a new parent function
		par = NewObject(name, sc.fid, node.GetRange(), c.module, &Function{})
		// The parent's node and range are the first overload
		par.info = &DeclarationInfo{node: node}
		par.order = sc.nextOrder()
		c.declareWithInfo(par, sc.ctx, nil, true)
		return par, false
	}
	if _, ok := par.typ.(*Function); !ok {
		if par.IsTypeName() {
			// If the parent is a type, this is an initializer for it
			return nil, true
		}
		// If the parent isn't a function, it's redeclared
		err := redeclaredError(&Object{rang: node.GetRange()}, par, false)
		c.fileError(err, sc.fid)
		return nil, false
	}
	return par, false
}

// declareVars declares placeholder [*Object]s for each variable
// declared in d. Values aren't checked yet.
func (c *Checker) declareVars(d *ast.VariableDeclaration, sc *stmtCollector,
	public bool, attrs *[]*ast.Attribute,
) {
	var (
		lastDecl     *Object
		varKind      int // 1 = var, 2 = const
		explicitType Type
		singleExpr   **Expr
	)
	if d.ExplicitType != nil {
		fctx := sc.ctx
		if sc.ctx.File.TopLevel() {
			fctx = c.FileContextOf(sc.fid)
		}
		explicitType = c.parseType(d.ExplicitType, fctx)
	}
	// If the RHS is a single value, store it so we can infer the
	// expression once.
	if d.IsSingleRHS() {
		if len(d.Values) != 1 {
			panic(fmt.Sprintf(
				"invalid AST: expected 1 or %d values, but got %d",
				len(d.Variables), len(d.Values),
			))
		}
		singleExpr = new(*Expr)
	}
	for i, assg := range d.Variables {
		dest, ok := assg.(ast.Destructurable)
		if !ok {
			// Not a destructure
			c.fileError(klarerrs.Node(klarerrs.ErrNonNameDeclaration, dest), sc.fid)
		}
		// Undestructured value
		var value ast.Expression
		if d.IsSingleRHS() {
			value = d.Values[0]
		} else {
			value = d.Values[i]
		}
		// Pointer value will be set when `value` is first inferred. Other
		// variables that depend on the same type (or value) will reuse
		// the cached type/Expr.
		rhsExpr := cmp.Or(singleExpr, new(*Expr))

		// Find every variable name in the destructure pattern
		for name, err := range ast.DestructureNames(dest) {
			if err != nil {
				// Expression is not a variable
				c.fileError(klarerrs.Node(klarerrs.ErrNonNameDeclaration, err), sc.fid)
				continue
			}
			obj := NewObject(name.Identifier, sc.fid, name.Range, c.module, nil)
			obj.public = public

			// Check whether the variable is a const. Vars and consts can't
			// be mixed in the same declaration.
			oldVarKind := varKind
			if IsConst(name.Identifier) {
				obj.typ = &Constant{}
				varKind = 2
			} else {
				varObjKind := LocalVar
				if sc.topLevel {
					varObjKind = TopLevelVar
				}
				_ = NewVariable(obj, varObjKind, nil)
				varKind = 1
			}
			if name.Identifier == "_" {
				// If the name is a discard, don't set whether the decl is
				// for consts or vars.
				varKind = oldVarKind
			}
			if oldVarKind != 0 && oldVarKind != varKind {
				// Vars and consts declared in the same declaration
				kindString := [...]string{1: "variable", 2: "constant"}

				err := objectError(klarerrs.ErrVarConstMixInDecl, obj)
				err.Label = "This is a " + kindString[varKind]
				err.AddHighlight(
					"This was already declared as a "+kindString[oldVarKind],
					lastDecl.rang,
				)
				// TODO: hint with diff
				err.Hint("Declare the variables and constants in separate declarations.")
				c.fileError(err, sc.fid)
				varKind = oldVarKind
			}
			if name.Identifier != "_" {
				lastDecl = obj
			}

			obj.info = &DeclarationInfo{node: d, varInfo: &varInfo{
				lhs:     dest,
				rhs:     value,
				rhsExpr: rhsExpr,
				expType: explicitType,
			}}
			obj.order = sc.nextOrder()
			c.declareWithInfo(obj, sc.ctx, attrs, true)
		}
	}
}

func (c *Checker) checkTopLevelStmts(stmts []ast.Statement, fctx *Context) {
	sctx := newStmtContext(fctx, fctx.File, 0)
	c.checkBlock(stmts, sctx)
}
