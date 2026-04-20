package analysis

import (
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/internal/version"
)

type FileID int

type Options struct {
	// The [Importer] to use for importing modules. If set to nil, an error is
	// raised when attempting to import a module.
	Importer Importer
	// The target platform for which the program is being compiled.
	// This is needed for resolving platform-specific implementations
	// and modules.
	Target target.Target
	// The minimum version of Klar required to compile the program.
	KlarVersion *version.Version
	// If Error != nil, it is called when an error is reported.
	Error func(errors.CompileError)
	// Whether the program is being typechecked for testing purposes.
	IsTest bool
	// If MaxErrors > 0, the type checker will terminate when this limit is reached.
	MaxErrors int
	// Type checker options from klar.build
	*klarbuild.CheckerOptions
}

type Checker struct {
	Programs    map[string]*ast.Program // Files in the module that is being checked.
	Errors      []errors.CompileError   // Errors reported while type checking.
	Options     *Options                // Options for type checking.
	rootContext *Context                // Context where top-level objects are defined.
	module      *Module

	importMap    map[string]*Module
	nodeContexts map[ast.Node]*Context
	moduleDecls  map[*Object]*DeclarationInfo // Declaration info for top-level objects
}

// NewChecker returns an initialized Checker that checks the programs in mod.
// If opts == nil, default options are used.
func NewChecker(mod *Module, opts *Options) *Checker {
	c := &Checker{}
	c.Init(mod, opts)
	return c
}

func (c *Checker) Init(mod *Module, opts *Options) {
	if opts == nil {
		opts = &Options{}
	}
	c.rootContext = mod.Context
	c.module = mod
	c.Programs = mod.Programs
	c.Options = opts
	c.moduleDecls = make(map[*Object]*DeclarationInfo)
}

func (c *Checker) Reset() {
	c.module = nil
	c.Errors = nil
	c.rootContext = nil
	c.Programs = nil
	c.Options = nil
	c.moduleDecls = nil
	c.nodeContexts = nil
	c.moduleDecls = nil
}

func (c *Checker) filePath(name string) string {
	if (c.module.Flags & SingleFileModule) != 0 {
		return c.module.Path
	}
	return filepath.Join(c.module.Path, name)
}

func (c *Checker) Check() {
	// Initialize contexts for each file
	fileContexts := c.initFileContexts()
	// Perform imports
	c.performFileImports(fileContexts)
	// Collect top-level objects in each file and put them in the module
	c.collectTopLevelObjects(fileContexts)
}

func (c *Checker) initFileContexts() map[string]*Context {
	fileContexts := make(map[string]*Context, len(c.Programs))
	c.module.fileID = make(map[FileID]string, len(c.Programs))
	if c.nodeContexts == nil {
		c.nodeContexts = make(map[ast.Node]*Context, len(c.Programs))
	}
	var i FileID
	for name := range c.Programs {
		c.module.fileID[i] = name
		fileContexts[name] = NewContext(c.rootContext, 0).setAttribute(ContextFile, i)
		i++
	}
	return fileContexts
}

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
			case *ast.FunctionDeclaration:
				name := stmt.Identifier.Name
				fn := NewObject(name, fid, stmt.GetRange(), c.module, &Function{})
				if stmt.Struct == nil {
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
				continue
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
			if fileName == "main" || c.module.Flags.Has(SingleFileModule) {
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
