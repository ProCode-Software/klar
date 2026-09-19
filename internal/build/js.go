package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ProCode-Software/klar/internal/codegen"
	"github.com/ProCode-Software/klar/internal/codegen/jswriter"
	"github.com/sanity-io/litter"
	"golang.org/x/sync/errgroup"
)

type jsCompiler struct {
	*PackageCompiler
	Modules []*Module
}

// Number of spaces used to indent generated JavaScript/TypeScript code.
const JavaScriptIndentSize = 2

type loweredModule struct {
	*Module
	ir []*codegen.File
}

func (pkc *PackageCompiler) CodegenJS(mods []*Module) error {
	jsc := &jsCompiler{pkc, mods}
	// What can and can't be parallelized:
	// - Can: Lowering individual files to JavaScript IR
	// - Not: Class association and reorganization for a single module
	// - Not: Bundling unless the bundle mode is per-module
	// - Can: Writing bundled or unbundled JavaScript IR to disk
	//
	// I can't think of reasons why lowering can fail. Bundling can fail
	// due to conflicting names in different modules; only if the mode is
	// [klarbuild.BundleSource] or [klarbuild.BundleStd]. Bundling the stdlib
	// can't fail because it's not exposed as an importable library and can
	// safely be name-mangled.

	// 1. Lower each module and reorganize JS IR
	var wg sync.WaitGroup
	loweredModCh := make(chan loweredModule, len(mods))
	for _, mod := range mods {
		wg.Go(func() {
			files := jsc.lowerModule(mod)
			loweredModCh <- loweredModule{Module: mod, ir: files}
		})
	}
	wg.Wait()
	close(loweredModCh)

	// 2. Bundle
	var eg errgroup.Group
	bundledModCh := make(chan loweredModule, len(mods))
	for mod := range loweredModCh {
		for _, file := range mod.ir {
			fmt.Printf("%s:\n", mod.FilePath(file.Name))
			litter.Dump(file.IR)
		}
		fmt.Println()

		eg.Go(func() (err error) {
			if mod.ir, err = jsc.bundleModule(mod); err != nil {
				return err
			}
			bundledModCh <- mod
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	close(bundledModCh)

	// 3. Write to disk
	indent := JavaScriptIndentSize
	if pkc.KlarBuild != nil && pkc.KlarBuild.JS != nil && pkc.KlarBuild.JS.Minify {
		indent = 0
	}
	for mod := range bundledModCh {
		for _, jsFile := range mod.ir {
			eg.Go(func() error { return jsc.writeFile(jsFile, mod.Module, indent) })
		}
	}
	return eg.Wait()
}

func (jsc *jsCompiler) lowerModule(mod *Module) []*codegen.File {
	gen := codegen.NewGenerator(mod.Programs, mod.Checked)
	gen.Run()
	return gen.Files
}

func (jsc *jsCompiler) bundleModule(mod loweredModule) (bundled []*codegen.File, err error) {
	return mod.ir, nil
}

func (jsc *jsCompiler) writeFile(jsFile *codegen.File, mod *Module, indent int) error {
	outPath := filepath.Join(mod.Path, strings.TrimSuffix(jsFile.Name, ".klar")+".js")
	// TODO: Use the configured build output. If none and bundled,
	// use the name of the module.
	f, err := os.Create(outPath)
	if err != nil {
		return &FilesystemError{"create", outPath, err}
	}
	defer f.Close()
	if err := jswriter.WriteModule(jsFile.IR, f, indent); err != nil {
		return &FilesystemError{"write to", outPath, err}
	}
	return nil
}
