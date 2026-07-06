package build

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ProCode-Software/klar/internal/build/cache"
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

	Root                 bool
	EnforceTargetSupport bool
	// TODO: should we add codegen options
}

func NewPackageCompiler(c *Compiler, i *Input) *PackageCompiler {
	return &PackageCompiler{
		Compiler: c,
		Input:    i,
		Root:     true,
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
	ld.Root = pkc.Root
	loaded, err := ld.Load()
	if err != nil {
		return nil, err
	}
	// Load their stdlib dependencies (will be put in pkc.Deps)
	if err := pkc.LoadStdlibDeps(loaded.stdlibDeps); err != nil {
		return nil, err
	}

	// Typecheck
	if modules, err = pkc.TypeCheckModules(loaded); err != nil {
		return
	}
	// TODO: Codegen?

	// Save succeeded modules to cache
	// TODO: Cache is unimplemented
	/* if err = pkc.WriteToCache(loaded.sortedDeps); err != nil {
		return modules, err
	} */
	return modules, nil
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
		if len(importPath) > 1 && importPath[1] == "js" {
			// 'klar.js' is a project-specific virtual module. It's based on the
			// package's targets and the input's klar.build JS options.
			pkc.setImportError(importPath.String(), errors.New(
				"klar.js can't be imported yet",
			))
			continue
		}
		modulePath := module.ModuleDirOf(importPath, stdDir, stdDir)
		inp, err := pkc.ResolveInput(modulePath, 0)
		if err != nil {
			// Not in the standard library. An error will be reported by the typechecker.
			return nil
		}
		compiler := NewPackageCompiler(pkc.Compiler, inp)
		compiler.Root = false
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

func (pkc *PackageCompiler) TypeCheckModules(loaded *Loaded) (
	succeededModules []*Module, err error,
) {
	succeededModules = loaded.cached // I don't care about loaded.cache being mutated
	// If the build mode is parse-only, we don't need to typecheck. Just return
	// the modules without syntax errors.
	if pkc.Mode == ModeParse || os.Getenv("NO_TYPECHECK") == "1" {
		for _, importPathStr := range loaded.sortedDeps {
			if mod, ok := pkc.Deps.TryGet(importPathStr); ok && !mod.Failed {
				succeededModules = append(succeededModules, mod)
			}
		}
		return succeededModules, nil
	}

	skippedModules := make(map[*Module]struct{})
typeCheckModules:
	for i, importPathStr := range loaded.sortedDeps {
		mod, ok := pkc.Deps.TryGet(importPathStr)
		if !ok {
			// Unknown dependency. Will be reported when dependents try to import this
			continue
		}
		if mod.Failed {
			// This module has syntax errors
			skippedModules[mod] = struct{}{}
			continue
		}
		// Ensure we can actually typecheck this module. If any of the
		// module's dependencies are failed or skipped, this one is skipped
		// and we can't typecheck
		for importPath := range mod.Deps {
			if _, ok := skippedModules[pkc.Deps.Get(importPath.String())]; ok {
				skippedModules[mod] = struct{}{}
				pkc.Info(
					"Skipping typecheck of module due to errors in dependencies",
					slog.String("module", mod.Path),
					slog.String("dependency", importPath.String()),
				)
				continue typeCheckModules
			}
		}
		// Now we can actually typecheck
		pkc.Progress.CheckingModule(mod.Path, i+1, len(loaded.sortedDeps))
		errs := pkc.TypeCheckModule(mod, importPathStr)
		if hasErrs, isMax := pkc.sendErrors(errs); hasErrs {
			// Module has type errors
			mod.Failed = true
			skippedModules[mod] = struct{}{}
			if isMax {
				return succeededModules, errMaxErrors
			}
			continue
		}
		succeededModules = append(succeededModules, mod)
	}
	return succeededModules, nil
}

func (pkc *PackageCompiler) WriteToCache(importPaths []string) error {
	for _, importPath := range importPaths {
		m, ok := pkc.Deps.TryGet(importPath)
		if !ok || m.Stdin || m.Failed {
			// Dependency doesn't exist. Error already reported
			// Stdin inputs shouldn't be cached
			// Module has errors (not warnings)
			continue
		}
		cacheMod := &cache.Module{
			Path:       m.Path,
			Programs:   m.Programs,
			ModTimes:   m.ModTimes,
			SingleFile: m.SingleFile,
			Checked:    m.Checked,
		}
		// Only add compile warnings from inside this module
		for _, warn := range pkc.Warnings {
			if (m.SingleFile && warn.File == m.Path) ||
				(!m.SingleFile && filepath.Dir(warn.File) == m.Path) {
				cacheMod.Warnings = append(cacheMod.Warnings, warn)
			}
		}
		if err := cache.Save(pkc.PkgInfo.CacheDir(), cacheMod); err != nil {
			return err
		}
		pkc.Debug("Saved module to cache", slog.String("module", pkc.Path))
	}
	return nil
}
