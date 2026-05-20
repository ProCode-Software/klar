package analysis

import (
	"fmt"
	"maps"
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
)

type methodInfo struct {
	decl  *ast.FunctionDeclaration
	alias *ast.FuncAliasDeclaration // If method alias and decl == nil
	obj   *Object
}

// collectTopLevelObjects collects all top-level objects from
// each program and declares placeholder objects in the module's context.
// Contents of each objects are not checked yet.
func (c *Checker) collectTopLevelObjects(
	files []string, fileContexts map[string]*Context,
) (methods map[string][]methodInfo, inits map[string][]*Object) {
	declareMethod := func(s *ast.Identifier, info methodInfo) {
		if methods == nil {
			methods = make(map[string][]methodInfo)
		}
		methods[s.Name] = append(methods[s.Name], info)
	}
	// Sort the files for reproducible output
	for _, fileName := range files {
		var (
			program = c.Programs[fileName]
			fctx    = fileContexts[fileName]
			fid     = fctx.getAttribute(ContextFile).(FileID)
			// First statement after imports. Any import after this is misplaced.
			// An import will never be at program.Body[firstStmt].
			firstStmt, _ = fctx.getAttribute(firstStmtIndex).(int)
			attrs        []*ast.Attribute // All items are *ast.Attribute
		)
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
				name := stmt.Identifier.Name
				ov := NewObject(name, fid, stmt.GetRange(), c.module, &Overload{})
				ov.typ.(*Overload).Object = ov
				ov.public = public

				if stmt.SelfType != nil {
					// Method
					declareMethod(stmt.SelfType, methodInfo{decl: stmt, obj: ov})
				} else if par, isInit := c.getOverloadParent(stmt, fid, fctx); par != nil {
					// Normal function. Declare the overload as a top-level object. par
					// will be nil if an error is reported for par not being a function.

					// We manually set c.moduleDecls instead of using
					// declareTopLevelObject() so that the overload itself isn't
					// declared into the context.
					ov.order = uint32(len(c.moduleDecls))
					c.moduleDecls[ov] = &DeclarationInfo{
						file:       fctx,
						node:       stmt,
						Attributes: c.parseAttributes(attrs, funcAttribute),
					}
					attrs = nil
				} else if isInit {
					// If `name` refers to a type, this overload is an initializer.
					if inits == nil {
						inits = make(map[string][]*Object)
					}
					inits[name] = append(inits[name], ov) // TODO
				}
				continue
			case *ast.FuncAliasDeclaration:
				// Will be resolved later
				obj := NewObject(stmt.Identifier.Name,
					fid, stmt.Range, c.module, &FunctionAlias{},
				)
				obj.public = true
				if stmt.Struct != nil {
					// Method alias
					declareMethod(stmt.Struct, methodInfo{alias: stmt, obj: obj})
				} else {
					// Normal alias
					c.declareTopLevelObject(obj, &attrs, &DeclarationInfo{
						file: fctx,
						node: stmt,
					})
				}
				continue
			case ast.TypeDeclaration:
				name := stmt.Name()
				obj := NewObject(name, fid, stmt.GetRange(), c.module, &TypeName{nil, name})
				c.declareTopLevelObject(obj, &attrs, &DeclarationInfo{
					node: stmt,
					file: fctx,
				})
				obj.public = public
				continue
			case *ast.VariableDeclaration:
				// TODO: some but not all var declarations are top level (can contain
				// function calls in values).
				// Make rules for declarations in main.klar files
				if fileName == "main.klar" {
					break
				}
				c.createVarPlaceholders(stmt, fid, fctx, &attrs, public)
				continue

			// Top-level statements
			// ========
			case *ast.BadExpression:
				// Shouldn't be here. Invalid ASTs should't be typechecked.
				panic("typechecking invalid AST")
			case *ast.ExpressionStatement:
				if c.module.Flags.Has(REPLModule) {
					break // Allow unused values in REPL
				}
				// Only 'when' and call expressions are allowed as statements.
				// TODO: move this to statement checking, not top-level
				switch stmt.Expression.(type) {
				case *ast.WhenExpression, *ast.CallExpression:
				case *ast.BadExpression:
					panic("typechecking invalid AST")
				default:
					c.fileError(klarerrs.Node(klarerrs.ErrUnusedValue, stmt), fid)
					continue
				}
			}
			if len(attrs) > 0 {
				// If we're here, the attributes weren't applied to a declaration.
				attrs = nil
			}
			// Top-level statement: only allowed in main.klar or single-file modules
			if fileName == "main.klar" || c.module.Flags.Has(SingleFileModule) {
				c.module.TopLevel = append(c.module.TopLevel, stmt)
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
		}
	}
	// Ensure no top-level objects were shadowed by imports
	for _, fileCtx := range fileContexts {
		for name, impObj := range fileCtx.Declarations {
			modObj := c.module.Context.Lookup(name)
			if modObj == nil {
				continue
			}
			// Only imports are in the file scope. One of these could possibly share a name:
			// - Namespace of normal import
			// - Alias of import
			// - An unqualified import object
			var namespace string
			if impObj.Kind() == KindModule {
				// Provide the import path the namespace is from
				namespace = impObj.module.ImportPathString()
			}
			err := klarerrs.Range(klarerrs.ErrImportShadow, impObj.rang)
			err.Label = klarerrs.Quote(name) + " was already declared in the module"
			err.Params = klarerrs.ErrorParams{"name": name, "import": namespace}
			// Provide a detail from where the module object was declared
			err.Details = append(err.Details, klarerrs.Detail{
				File:  modObj.FilePath(),
				Range: modObj.rang, Message: "It was already declared here",
			})
			c.fileError(err, impObj.file)
		}
	}
	return methods, inits
}

// checkTopLevelObjects typechecks all top-level objects, but not function bodies.
// TODO: take a context
func (c *Checker) checkTopLevelObjects(
	methods map[string][]methodInfo, inits map[string][]*Object,
) {
	var (
		objs        = slices.SortedFunc(maps.Keys(c.moduleDecls), sortByOrder)
		typeAliases []*Object // Guaranteed to be TypeName
		nonTypes    []*Object // Variable/function declaration
		funcAliases []*Object // Guaranteed to be FunctionAlias
	)
	// 1. Check new type declarations (not aliases)
	for _, obj := range objs {
		switch obj.typ.(type) {
		case *TypeName:
			if _, ok := c.moduleDecls[obj].node.(*ast.TypeAliasDeclaration); ok {
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
		if _, ok := obj.typ.(*FunctionAlias); !ok {
			c.checkDeclaration(obj)
		}
	}
	// 4. Function aliases (no methods)
	for _, info := range funcAliases {
		c.resolveFuncAlias(info)
	}
	// 5. Methods and initializers: Associate methods/initializers with receiver types
	for typeName, methods := range methods {
		c.collectMethods(c.rootContext, typeName, methods)
	}
	for typeName, inits := range inits {
		c.collectInitializers(c.rootContext, typeName, inits)
	}
}

// getOverloadParent finds the [*Function] associated with f's name, which
// overloads can be added to. An error is reported if the object with f's
// name exists and is not a [*Function].
func (c *Checker) getOverloadParent(
	f *ast.FunctionDeclaration, fid FileID, fctx *Context,
) (_ *Object, isInit bool) {
	// The p function we're adding overloads to
	p := c.rootContext.Lookup(f.Identifier.Name)
	if p == nil {
		// If this is the first overload, declare a new parent function
		p = NewObject(f.Identifier.Name, fid, f.GetRange(), c.module, &Function{})
		// The parent's node and range are the first overload
		c.declareTopLevelObject(p, nil, &DeclarationInfo{
			file: fctx,
			node: f,
		})
	} else if _, ok := p.typ.(*Function); !ok {
		if _, ok := p.typ.(*TypeName); ok {
			return nil, true
		}
		// If the parent isn't a function, it's redeclared
		err := redeclaredError(&Object{rang: f.Range}, p, false)
		c.fileError(err, fid)
		return nil, false
	}
	return p, false
}

// createVarPlaceholders declares placeholder [*Object]s for each variable
// declared in d. Types and values aren't checked yet.
func (c *Checker) createVarPlaceholders(d *ast.VariableDeclaration,
	fid FileID, fctx *Context,
	attrs *[]*ast.Attribute, public bool,
) {
	var (
		lastDecl     *Object
		varKind      int // 1 = var, 2 = const
		explicitType Type
	)
	if d.ExplicitType != nil {
		explicitType = c.parseType(d.ExplicitType, fctx)
	}
	for i, assg := range d.Variables {
		dest, ok := assg.(ast.Destructurable)
		if !ok {
			// Not a destructure
			c.fileError(klarerrs.Node(klarerrs.ErrNonNameDeclaration, dest), fid)
		}
		// Undestructured value
		var value ast.Expression
		if len(d.Values) < len(d.Variables) {
			if len(d.Values) != 1 {
				panic(fmt.Sprintf("expected 1 or %d values, but got %d",
					len(d.Variables), len(d.Values),
				))
			}
			value = d.Values[0]
		} else {
			value = d.Values[i]
		}
		// Pointer value will be set when `value` is first inferred
		rhsType := new(Type(nil))
		if explicitType != nil {
			*rhsType = explicitType
		}

		// Find every variable name in the destructure pattern
		for name, err := range dest.Names() {
			if err != nil {
				// Expression is not a variable
				c.fileError(klarerrs.Node(klarerrs.ErrNonNameDeclaration, err), fid)
				continue
			}
			obj := NewObject(name.Name, fid, name.Range(), c.module, nil)
			obj.public = public

			// Check whether the variable is a const. Vars and consts can't
			// be mixed in the same declaration.
			oldVarKind := varKind
			if IsConst(name.Name) {
				obj.typ = &Constant{}
				varKind = 2
			} else {
				typ := &Variable{VarKind: TopLevelVar}
				typ.Object = obj
				obj.typ = typ
				varKind = 1
			}
			if name.IsDiscard() {
				// If the name is a discard, don't set whether the decl is
				// for consts or vars.
				varKind = oldVarKind
			}
			if oldVarKind != 0 && oldVarKind != varKind {
				// Vars and consts declared in the same declaration
				kindString := [...]string{1: "variable", 2: "constant"}

				err := objectError[*klarerrs.Error](klarerrs.ErrVarConstMixInDecl, obj)
				err.Label = "This is a " + kindString[varKind]
				err.Highlights = append(err.Highlights, klarerrs.Highlight{
					Range:   lastDecl.rang,
					Message: "This was already declared as a " + kindString[oldVarKind],
				})
				// TODO: hint with diff
				err.Hint("Declare the variables and constants in separate declarations.")
				c.fileError(err, fid)
				varKind = oldVarKind
			}
			if !name.IsDiscard() {
				lastDecl = obj
			}

			c.declareTopLevelObject(obj, attrs, &DeclarationInfo{
				node:    d,
				file:    fctx,
				rhs:     value,
				rhsType: rhsType,
			})
		}
	}
}
