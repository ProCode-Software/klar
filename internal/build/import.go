package build

import (
	"fmt"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/module/imports"
	"github.com/ProCode-Software/klar/internal/target"
)

// CompileFunc is [module.BaseImporter.Compile].
type CompileFunc = func(p imports.ImportPath, dir string, t target.Target) (
	*analysis.Module, error,
)

// makeImportCompiler returns a CompileFunc that compiles dependencies for
// the given host module. This is used by [module.BaseImporter].
func (c *Compiler) makeImportCompiler(hostMod *Module) CompileFunc {
	return func(p imports.ImportPath, dir string, t target.Target) (*analysis.Module, error) {
		fmt.Println("import", p)
		// If the compiler already typechecked the module, look for it
		// TODO: add a module map to [Compiler]?
		for _, mod := range c.Modules {
			if mod.Path == dir && mod.Checked != nil {
				return mod.Checked, nil
			}
		}
		// Compile from scratch
		return c.CompileImport(hostMod, p, dir, t)
	}
}

// TODO: Store a map of [Compiler] to avoid recreating compilers for the same
// host module and target package.

func (c1 *Compiler) CompileImport(hostMod *Module,
	p imports.ImportPath, dir string, t target.Target,
) (*analysis.Module, error) {
	newError := func(err error) *klarerrs.Error {
		return &klarerrs.Error{
			Code: klarerrs.ErrModuleCompileError,
			Info: klarerrs.ModuleErrorInfo{
				ModulePath: dir,
				ImportPath: p.String(),
				Err:        err,
			},
		}
	}
	c, err := NewCompiler(c1.Mode)
	if err != nil {
		return nil, err
	}
	c.UseStdParser()
	input := Input{Path: dir, Name: p.Namespace(), Kind: KindModule}
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
		klarBuild.Target = t
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
		println("multiple modules", fmt.Sprintf("%#v", res.Modules))
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

func (c *Compiler) GetImporter(m *Module) analysis.Importer {
	i := module.NewBaseImporter(c.moduleInputs[m].PkgInfo, c.makeImportCompiler(m))
	return i
}
