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
	Target *target.Target
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

func (c *Checker) collectTopLevelObjects(fileContexts map[string]*Context) {
	var methods []*ast.FunctionDeclaration
	for name, program := range c.Programs {
		fileContext := fileContexts[name]
		fid := fileContext.getAttribute(ContextFile).(FileID)
		// First statement after imports. Any import after this is misplaced.
		// An import will never be at program.Body[firstStmt].
		firstStmt, _ := fileContext.getAttribute(firstStmtIndex).(int)
		for _, stmt := range program.Body[firstStmt:] {
			switch stmt := stmt.(type) {
			case *ast.ImportStatement:
				// Imports were already processed. Misplaced import
				c.fileError(errors.Node(errors.ErrImportsGoFirst, stmt), fid)
				continue
			case *ast.FunctionDeclaration:
				_ = methods
				continue
			case *ast.FuncAliasDeclaration:
			case ast.TypeDeclaration:
				obj := NewObject(stmt.Name(), fid, stmt.GetRange(), c.module, TypeName{nil})
				c.declareTopLevelObject(obj, &DeclarationInfo{node: stmt, file: fileContext})
				continue
			case *ast.PublicDeclaration:
				continue
			case *ast.OpaqueDeclaration:
				// Opaque declarations should be inside [ast.PublicDeclaration].
				// This declaration is not public
				c.fileError(errors.Node(errors.ErrPrivateOpaque, stmt), fid)
				continue

			case *ast.VariableDeclaration:
				// TODO: some but not all var declarations are top level (contain function calls in
				// values). Call a function that should also determine if the declaration is top level.
				continue
			case *ast.BadExpression:
				continue // Shouldn't be here. Invalid ASTs should't be typechecked.
			case *ast.ExpressionStatement:
				// Only 'when' and call expressions are allowed as statements.
				// TODO: move this to statement checking, not top-level
				switch stmt.Expression.(type) {
				case *ast.WhenExpression, *ast.CallExpression, *ast.BadExpression:
				default:
					c.fileError(errors.Node(errors.ErrUnusedValue, stmt), fid)
					continue
				}
			}
			// Top-level statement
			if c.module.TopLevel != -1 && c.module.TopLevel != fid {
				c.fileError(errors.Node(errors.ErrTopLevel, stmt).
					SetParam("other", c.module.ResolveFile(c.module.TopLevel)), fid,
				)
			} else {
				c.module.TopLevel = fid
			}
		}
		_ = fileContext
	}
}
