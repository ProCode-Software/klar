package analysis

import (
	"fmt"
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/internal/version"
)

type FileID int

type Options struct {
	// Imports modules. If set to nil, [StdImporter] is used. TODO
	Importer module.Importer
	// The target platform for which the program is being compiled.
	// This is needed for resolving platform-specific implementations
	// and modules.
	Target *target.Target
	// The minimum version of Klar required to compile the program.
	KlarVersion *version.Version
	// If Error != nil, it is called when an error is reported.
	Error func(errors.CompileError)
	// If MaxErrors > 0, the type checker will terminate when this limit is reached.
	MaxErrors int
	// Type checker options from klar.build
	*klarbuild.CheckerOptions
}

type Checker struct {
	Programs     map[string]*ast.Program // Files in the module that is being checked.
	Errors       []errors.CompileError   // Errors reported while type checking.
	Options      *Options                // Options for type checking.
	rootContext  *Context                // Context where top-level objects are defined.
	module       *Module
	fileIds      map[FileID]string // File IDs to file base names
	usedImports  map[*Module]struct{}
	importMap    map[string]*Module
	nodeContexts map[ast.Node]*Context
	moduleDecls  map[*Object]*DeclarationInfo // Declaration info for top-level objects
}

func NewChecker(
	name, path string, importPath []string,
	files map[string]*ast.Program, opts *Options,
) *Checker {
	if opts == nil {
		opts = &Options{Importer: nil} // TODO: use default importer
	}
	mod := NewModule(name, path, importPath, files, opts.KlarVersion, opts.Target)
	return NewCheckerFromModule(mod, opts)
}

func NewCheckerFromModule(mod *Module, opts *Options) *Checker {
	if opts == nil {
		opts = &Options{Importer: nil} // TODO: use default importer
	}
	return &Checker{
		rootContext: mod.Context,
		module:      mod,
		Programs:    mod.Programs,
		Options:     opts,
	}
}

func NewEmptyChecker() *Checker {
	return &Checker{}
}

func (c *Checker) Init(mod *Module, opts *Options) {
	c.rootContext = mod.Context
	c.module = mod
	c.Programs = mod.Programs
	c.Options = opts
}

func (c *Checker) Reset() {
	c.module = nil
	c.Errors = nil
	c.rootContext = nil
	c.usedImports = nil
	c.Programs = nil
	c.Options = nil
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
	if c.nodeContexts == nil {
		c.nodeContexts = make(map[ast.Node]*Context, len(c.Programs))
	}
	var i int
	for name := range c.Programs {
		fileContexts[name] = NewContext(c.rootContext, 0).setAttribute(ContextFile, i)
		i++
	}
	return fileContexts
}

func (c *Checker) collectTopLevelObjects(fileContexts map[string]*Context) {
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
				c.FileError(errors.Node(errors.ErrImportsGoFirst, stmt), fid)
			case *ast.FunctionDeclaration:

			case *ast.FuncAliasDeclaration:
			case ast.TypeDeclaration:
				obj := NewObject(stmt.Name(), fid, stmt.GetRange(), c.module, TypeName{nil})
				
			case *ast.PublicDeclaration:
			case *ast.OpaqueDeclaration:
				// Opaque declarations should be inside [ast.PublicDeclaration].
				// This declaration is not public
				c.FileError(errors.Node(errors.ErrPrivateOpaque, stmt), fid)
			case *ast.ExpressionStatement:
				// Only 'when' and call expressions are allowed as statements.
				// TODO: move this to statement checking, not top-level
				// TODO: allow 'when' and function call
				switch stmt.Expression.(type) {
				case *ast.WhenExpression, *ast.CallExpression:
				default:
					c.FileError(errors.Node(errors.ErrUnusedValue, stmt), fid)
				}
			default:
				panic(fmt.Sprintf("unhandled top-level statement %T", stmt))
			}
		}
		_ = fileContext
	}
}

