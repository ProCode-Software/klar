package build

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ProCode-Software/klar/internal/codegen"
	"github.com/ProCode-Software/klar/internal/codegen/jswriter"
	"github.com/ProCode-Software/klar/internal/module"
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
	var (
		wg           sync.WaitGroup
		loweredModCh = make(chan loweredModule, len(mods))
		progressCh   chan struct{}
	)
	if !pkc.ProgressHidden() {
		// Show progress to the user
		progressCh = make(chan struct{}, len(mods))
		go jsc.loweringProgress(progressCh)
	}
	for _, mod := range mods {
		wg.Go(func() {
			files := jsc.lowerModule(mod)
			loweredModCh <- loweredModule{Module: mod, ir: files}
			if progressCh != nil {
				progressCh <- struct{}{}
			}
		})
	}
	wg.Wait()
	close(loweredModCh)
	if progressCh != nil {
		close(progressCh)
	}

	// 2. Bundle
	var eg errgroup.Group
	bundledModCh := make(chan loweredModule, len(mods))
	for mod := range loweredModCh {
		/* for _, file := range mod.ir {
			fmt.Printf("%s:\n", mod.FilePath(file.Name))
			litter.Dump(file.IR)
		}
		fmt.Println() */

		eg.Go(func() (err error) {
			if mod.ir, err = jsc.bundleModule(mod); err != nil {
				jsc.Logger.Error(
					"Failed to bundle module",
					slog.String("module", mod.Path), slog.Any("error", err),
				)
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
	pkc.Progress.WritingModules()
	indent := JavaScriptIndentSize
	if pkc.KlarBuild != nil && pkc.KlarBuild.JS != nil && pkc.KlarBuild.JS.Minify {
		indent = 0
	}
	for mod := range bundledModCh {
		// TODO: This has to be set earlier. Ensure this is correct,
		// and whether the output is a file or directory is validated.
		var outDir string
		if false && pkc.KlarBuild != nil &&
			len(pkc.KlarBuild.Output) > 0 && pkc.KlarBuild.Output[0] != "" {
			// TODO: Make the output path relative to the location of the klar.build
			outDir = pkc.KlarBuild.Output[0]
		} else if pkc.PkgInfo != nil {
			outDir = filepath.Join(append(
				[]string{pkc.PkgInfo.Dir, module.DistDir},
				mod.Checked.ImportPath...,
			)...)
		}
		if outDir != "" {
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				jsc.Logger.Error(
					"Failed to create module output directory",
					slog.String("module", mod.Path), slog.String("output", outDir),
					slog.Any("error", err),
				)
				eg.Go(func() error { return err })
				continue
			}
		}
		jsc.Logger.Debug(
			"Writing JavaScript for module",
			slog.String("module", mod.Path), slog.Int("numFiles", len(mod.ir)),
			slog.String("to", outDir),
		)
		for _, jsFile := range mod.ir {
			eg.Go(func() error {
				return jsc.writeFile(jsFile, mod.Module, outDir, indent)
			})
		}
	}
	return eg.Wait()
}

func (jsc *jsCompiler) lowerModule(mod *Module) []*codegen.File {
	jsc.Logger.Debug("Lowering module to JavaScript", slog.String("module", mod.Path))
	gen := codegen.NewGenerator(mod.Programs, mod.SortedFiles(), mod.Checked)
	gen.Run()
	return gen.Files
}

func (jsc *jsCompiler) bundleModule(mod loweredModule) (bundled []*codegen.File, err error) {
	jsc.Logger.Debug("Bundling module", slog.String("module", mod.Path))
	return mod.ir, nil
}

func (jsc *jsCompiler) writeFile(
	jsFile *codegen.File, mod *Module, outDir string, indent int,
) error {
	var outPath string
	if outDir != "" {
		outPath = filepath.Join(
			outDir,
			strings.TrimSuffix(jsFile.Name, ".klar")+".js",
		)
	} else {
		outPath = strings.TrimSuffix(mod.Path, ".klar") + ".js"
	}
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
	// TODO: Banner
	return nil
}

func (jsc *jsCompiler) loweringProgress(ch chan struct{}) {
	total := cap(ch)
	curr := 1
	jsc.Progress.GeneratingJS(curr, total)
	for range ch {
		curr++
		if curr <= cap(ch) {
			jsc.Progress.GeneratingJS(curr, total)
		}
	}
}
