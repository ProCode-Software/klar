package build

import (
	"fmt"
	"sync"

	"github.com/ProCode-Software/klar/internal/codegen"
	"github.com/sanity-io/litter"
)

type jsCompiler struct {
	*PackageCompiler
	Modules []*Module
}

func (pkc *PackageCompiler) CodegenJS(mods []*Module) {
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
	type loweredModule struct {
		*Module
		ir []*codegen.File
	}
	loweredModCh := make(chan loweredModule, len(mods))
	for _, mod := range mods {
		wg.Go(func() {
			files := jsc.lowerModule(mod)
			loweredModCh <- loweredModule{Module: mod, ir: files}
		})
	}
	wg.Wait()
	close(loweredModCh)

	for mod := range loweredModCh {
		for _, file := range mod.ir {
			fmt.Printf("%s:\n", mod.FilePath(file.Name))
			litter.Dump(file.IR)
		}
		fmt.Println()
	}
}

func (jsc *jsCompiler) lowerModule(mod *Module) []*codegen.File {
	gen := codegen.NewGenerator(mod.Programs, mod.Checked)
	gen.Run()
	return gen.Files
}
