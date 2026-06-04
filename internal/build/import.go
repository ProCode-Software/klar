package build

import (
	"sync"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/module"
)

// makeImportCompiler returns a [module.CompileFunc] that compiles
// dependencies for hostMod, the module that is importing the dependency.
// This is used by [module.BaseImporter].
func (c *Compiler) makeImportCompiler(hostMod *Module) module.CompileFunc {
	return func(ctx module.ImportContext, dir string) (*analysis.Module, error) {
		// When a module is requested by Importer, there are 3 possibilities:
		//
		// 1. The module is a Compiler input, and is already typechecked
		// 2. The module is an input, but is awaiting typechecking
		// 3. The module is a dependency that needs to be loaded from cache
		// or compiled from scratch

		// The module is an input, typechecked or not
		if mod := c.moduleMap.Get(dir); mod != nil {
			if mod.Checked == nil {
				<-mod.Ready // Wait for typechecking to complete
			}
			// Check if the module has errors
			if mod.Checked.Flags.Has(analysis.ModuleWithErrors) {
				return nil, klarerrs.ImportError(
					klarerrs.ErrModuleCompileError,
					mod.Checked.ImportPath, dir, nil,
				)
			}
			return mod.Checked, nil
		}
		// The module isn't an input -- a dependency.
		// Module cache hasn't been implemented yet (TODO), so modules will
		// always be compiled from scratch each compile session.
		return c.CompileImport(hostMod, dir, ctx)
	}
}

func (c1 *Compiler) CompileImport(
	hostMod *Module, dir string, ctx module.ImportContext,
) (*analysis.Module, error) {
	newError := func(err error) *klarerrs.Error {
		return &klarerrs.Error{
			Code: klarerrs.ErrModuleCompileError,
			Info: klarerrs.ModuleErrorInfo{
				ModulePath: dir,
				ImportPath: ctx.ImportPath().String(),
				Err:        err,
			},
		}
	}
	c, err := NewCompiler(c1.Mode)
	if err != nil {
		return nil, err
	}
	c.UseStdParser()
	input := Input{Path: dir, Name: ctx.ImportPath().Namespace(), Kind: KindModule}
	input.ResolveKlarBuild()

	// Parse the dependency's klar.build
	klarBuild := &c1.moduleInputs[hostMod].Options.File // Host klar.build
	if input.KlarBuild != "" {
		// Ignore the dependency's warnings
		config, _, err := klarbuild.Parse(input.KlarBuild)
		if err != nil {
			return nil, newError(err)
		}
		// Merge it with the host's klar.build
		klarBuild = mergeDependencyKlarBuild(klarBuild, config)
		klarBuild.Target = ctx.Target()
	}
	c.Options = append(c.Options, &Options{Inputs: []Input{input}, File: *klarBuild})

	// Compile!
	res, err := c.Compile()
	if err != nil {
		return nil, newError(err)
	}
	if len(res.Errors) > 0 {
		return nil, newError(res.Errors[0])
	}
	if len(res.Modules) > 1 { // For debugging only
		for _, mod := range res.Modules {
			println("module", mod.Name, mod.Path)
		}
	}
	// Hopefully, this should be the correct (and only) module.
	return res.Modules[0].Checked, nil
}

// mergeDependencyKlarBuild merges the dependency's klar.build with the host's.
// The dependency's [klarbuild.CheckerOptions], [klarbuild.AssetOptions], and
// some of [klarbuild.JSOptions] are used. All other fields are copied from the host.
func mergeDependencyKlarBuild(host, dep *klarbuild.File) *klarbuild.File {
	if dep == nil || host == dep {
		return host
	}
	if len(dep.Configurations) > 0 {
		// TODO: is this correct?
		dep.Configuration = *dep.Configurations[0]
	}
	copy := *host
	kb := &copy

	kb.WarningsAsErrors = dep.WarningsAsErrors
	kb.SuppressWarnings = dep.SuppressWarnings
	kb.Checker = dep.Checker
	kb.Assets = dep.Assets
	if dep.JS != nil {
		kb.JS.Banner = dep.JS.Banner
		kb.JS.Globals = dep.JS.Globals
		kb.JS.ESNext = dep.JS.ESNext
		kb.JS.TypeScriptLibs = dep.JS.TypeScriptLibs
	}
	return kb
}

// Avoid recreating importers for the same package.
// string = Package directory (with glas.pack)
// TODO: not target-aware
var (
	importerMu    sync.Mutex
	importerCache = make(map[string]analysis.Importer)
)

func (c *Compiler) GetImporter(m *Module) analysis.Importer {
	mi := c.moduleInputs[m]
	importerMu.Lock()
	defer importerMu.Unlock()
	if importerCache == nil {
		importerCache = make(map[string]analysis.Importer)
	} else if cached, ok := importerCache[mi.PkgInfo.Dir]; ok {
		return cached
	}
	i := module.NewBaseImporter(mi.PkgInfo, c.makeImportCompiler(m))
	importerCache[mi.PkgInfo.Dir] = i
	return i
}
