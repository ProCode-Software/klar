package build

import (
	"errors"
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/module/imports"
	"github.com/ProCode-Software/klar/internal/version"
)

type PackageCompiler struct {
	*Compiler
	*Input
	Deps *Deps
	// Errors that will appear when importing modules with these
	// import paths (keys).
	importErrs map[string]error

	EnforceTargetSupport bool
	// TODO: should we add codegen options
}

func NewPackageCompiler(c *Compiler, i *Input) *PackageCompiler {
	return &PackageCompiler{
		Compiler: c,
		Input:    i,
	}
}

func (pkc *PackageCompiler) Compile() (modules []*Module, err error) {
	// Check if the compiler supports compiling the package
	if pkc.Manifest != nil {
		if err := CheckCompilerCompatibility(pkc.Manifest.Klar); err != nil {
			return nil, err
		}
	}

	// Load modules from cache or parse their files
	ld := NewLoader(pkc.Compiler, pkc.Input, pkc.Deps)
	loaded, err := ld.Load()
	if err != nil {
		return nil, err
	}
	// Load their stdlib dependencies (will be put in pkc.Deps)
	if err := pkc.LoadStdlibDeps(loaded.stdlibDeps); err != nil {
		return nil, err
	}

	// Typecheck
	modules = pkc.TypecheckModules(loaded)

	// TODO: Codegen
	return
}

func CheckCompilerCompatibility(spec version.Specifier) error {
	if spec.IsZero() {
		return nil
	}
	if !spec.Matches(cli.ParsedKlarVersion) {
		return &InterfaceError{Code: ErrKlarVersion, Value: spec.String()}
	}
	return nil
}

func (pkc *PackageCompiler) LoadStdlibDeps(stdDeps []imports.ImportPath) error {
	stdDir := module.SystemDirs.Std
	for _, importPath := range stdDeps {
		if _, ok := pkc.Deps.TryGet(importPath.String()); ok {
			continue // Module already compiled
		}
		if importPath[1] == "js" {
			// klar.js is a project-specific "fake" module
			pkc.setImportError(importPath.String(), errors.New(
				"klar.js can't be imported yet",
			))
			continue
		}
		modulePath := stdDir + sep + module.SrcDir + sep + filepath.Join(importPath...)
		inp, err := pkc.ResolveInput(modulePath, 0, false)
		if err != nil {
			return err
		}
		compiler := NewPackageCompiler(pkc.Compiler, inp)
		compiler.Deps = pkc.Deps
		compiler.EnforceTargetSupport = false
		if _, err := compiler.Compile(); err != nil {
			return err
		}
	}
	return nil
}

func (pkc *PackageCompiler) setImportError(path string, err error) {
	if pkc.importErrs == nil {
		pkc.importErrs = make(map[string]error)
	}
	pkc.importErrs[path] = err
}

func (pkc *PackageCompiler) TypecheckModules(loaded *Loaded) (succeededModules []*Module) {
	succeededModules = loaded.cached // I don't care about loaded.cache being mutated
	skippedModules := make(map[*Module]struct{})
typeCheckModules:
	for _, importPathStr := range loaded.sortedDeps {
		mod, ok := pkc.Deps.TryGet(importPathStr)
		if !ok {
			// Unknown dependency. Will be reported when dependents try to import this
			continue
		}
		// The module is added to Deps even if it fails. When we compile another
		// module, it will see this module in Deps and know it failed.
		pkc.Deps.Set(mod, importPathStr)
		if mod.Failed {
			// This module has syntax errors
			skippedModules[mod] = struct{}{}
			continue
		}
		// Ensure we can actually typecheck this module. If any of the
		// module's dependencies are failed or skipped, this one is skipped
		// and we can't typecheck
		for _, prog := range mod.Programs {
			for importPath := range prog.Deps {
				if _, ok := skippedModules[pkc.Deps.Get(importPath.String())]; ok {
					skippedModules[mod] = struct{}{}
					continue typeCheckModules
				}
			}
		}
		// Now we can actually typecheck
		errs := pkc.TypeCheckModule(mod, importPathStr)
		if hasErrs := pkc.sendErrors(errs); hasErrs {
			// Module has type errors
			mod.Failed = true
			skippedModules[mod] = struct{}{}
			return
		}
		succeededModules = append(succeededModules, mod)
	}
	return succeededModules
}
