package analysis

import (
	"maps"
	"slices"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
)

type methodInfo struct {
	self *ast.Identifier
	decl *ast.FunctionDeclaration
	obj  *Object
}

type funcAliasInfo struct {
	decl   *ast.FuncAliasDeclaration
	fid    FileID
	public bool
}

// collectTopLevelObjects collects all top-level objects from
// each program and declares objects in the module's context.
func (c *Checker) collectTopLevelObjects(fileContexts map[string]*Context) (
	methods map[string][]methodInfo, funcAliases []funcAliasInfo,
) {
	var overloads map[string][]*Object
	for fileName, program := range c.Programs {
		var (
			fileContext = fileContexts[fileName]
			fid         = fileContext.getAttribute(ContextFile).(FileID)
			// First statement after imports. Any import after this is misplaced.
			// An import will never be at program.Body[firstStmt].
			firstStmt, _ = fileContext.getAttribute(firstStmtIndex).(int)
		)
		for i, stmt := range program.Body[firstStmt:] {
			var public bool
			if s, ok := stmt.(*ast.PublicDeclaration); ok {
				public = true
				stmt = s.Declaration
			}
			switch stmt := stmt.(type) {
			case *ast.ImportStatement:
				// Imports were already processed. Misplaced import
				err := errors.Node(errors.ErrImportsGoFirst, stmt)
				err.Label = "Put this import at the top of the file"
				c.fileError(err, fid)
				continue
			case *ast.Attribute:
				for _, attr := range program.Body[i:] {
					var nextStmt ast.Statement
					attr, ok := attr.(*ast.Attribute)
					if !ok {
						nextStmt = attr
						break
					}
					// TODO: store a slice of Attr for the node after
					_ = nextStmt
				}
				continue
			case *ast.FunctionDeclaration:
				name := stmt.Identifier.Name
				fn := NewObject(name, fid, stmt.GetRange(), c.module, &Function{})
				if stmt.Struct == nil {
					// Normal function
					if overloads == nil {
						overloads = make(map[string][]*Object)
					}
					overloads[name] = append(overloads[name], fn)
					// c.declare(fileContext, fn)
				} else if !stmt.Struct.IsDiscard() {
					// Method - ignore discarded names, though they will still be typechecked
					if methods == nil {
						methods = make(map[string][]methodInfo)
					}
					methods[stmt.Struct.Name] = append(methods[stmt.Struct.Name],
						methodInfo{stmt.Struct, stmt, fn},
					)
				}
				info := &DeclarationInfo{file: fileContext, node: stmt}
				c.moduleDecls[fn] = info
				fn.order = uint32(len(c.moduleDecls))
				fn.public = public
				continue
			case *ast.FuncAliasDeclaration:
				// Will be resolved later
				funcAliases = append(funcAliases, funcAliasInfo{stmt, fid, public})
				continue
			case ast.TypeDeclaration:
				name := stmt.Name()
				obj := NewObject(name, fid, stmt.GetRange(), c.module, &TypeName{nil, name})
				c.declareTopLevelObject(obj, &DeclarationInfo{node: stmt, file: fileContext})
				obj.public = public
				continue

			case *ast.VariableDeclaration:
				// TODO: some but not all var declarations are top level (contain function calls in
				// values). Call a function that should also determine if the declaration is top level.
				if fileName == "main.klar" {
					break
				}
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
					c.fileError(errors.Node(errors.ErrUnusedValue, stmt), fid)
					continue
				}
			}
			// Top-level statement: only allowed in main.klar or single-file modules
			if fileName == "main.klar" || c.module.Flags.Has(SingleFileModule) {
				c.module.TopLevel = append(c.module.TopLevel, stmt)
			} else {
				c.fileError(errors.Node(errors.ErrTopLevel, stmt), fid)
			}
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
			err := errors.Range(errors.ErrImportShadow, impObj.rang)
			err.Label = errors.Quote(name) + " was already declared in the module"
			err.Params = errors.ErrorParams{"name": name, "import": namespace}
			// Provide a detail from where the module object was declared
			err.Details = append(err.Details, errors.Detail{
				File:      modObj.FilePath(),
				Highlight: errors.Highlight{modObj.rang, "It was already declared here"},
			})
			c.fileError(err, impObj.file)
		}
	}
	return
}

// checkTopLevelObjects typechecks all top-level objects, but not function bodies.
// TODO: take a context
func (c *Checker) checkTopLevelObjects(
	methods map[string][]methodInfo, funcAliases []funcAliasInfo,
) {
	var (
		objs        = slices.SortedFunc(maps.Keys(c.moduleDecls), sortByOrder)
		typeAliases []*Object
		nonTypes    []*Object // Guaranteed to be TypeName
	)
	// 1. Associate methods with receiver types
	for typeName, methods := range methods {
		c.collectMethods(c.rootContext, typeName, methods)
	}
	// 2. Check new type declarations (not aliases)
	for _, obj := range objs {
		if _, ok := obj.typ.(*TypeName); ok {
			if _, ok := c.moduleDecls[obj].node.(*ast.TypeAliasDeclaration); ok {
				typeAliases = append(typeAliases, obj)
			} else {
				c.checkDeclaration(obj)
			}
		} else {
			nonTypes = append(nonTypes, obj)
		}
	}
	// 3. Type aliases
	for _, obj := range typeAliases {
		c.checkDeclaration(obj)
	}
	// 4. Non-types
	for _, obj := range nonTypes {
		if _, ok := obj.typ.(*FunctionAlias); !ok {
			c.checkDeclaration(obj)
		}
	}
	// 5. Function aliases
	for _, info := range funcAliases {
		c.resolveFuncAlias(info)
	}
}
