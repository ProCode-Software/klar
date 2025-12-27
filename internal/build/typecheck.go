package build

import (
	"sync"

	"github.com/ProCode-Software/klar/internal/analysis"
	"github.com/ProCode-Software/klar/internal/errors"
)

// Step 4: Type check the modules. This is done in a separate goroutine,
// where the parsed ASTs are in a channel (see [Compiler.ParseModules]).

// TypeCheckModules type checks the modules in moduleCh, sending any critical
// error to errCh and a signal to done when finished.
func (c *Compiler) TypeCheckModules(procCtx *processContext, moduleCh chan *Module) {
	checkerPool := newCheckerPool()
	for {
		select {
		case <-procCtx.ctx.Done():
			return
		case parsedMod, more := <-moduleCh:
			if !more {
				procCtx.done <- struct{}{}
				return
			}
			// Typecheck the module
			errs := c.typeCheckModule(parsedMod, checkerPool)
			if len(errs) > 0 {
				select {
				case procCtx.errorCh <- errs:
				case <-procCtx.ctx.Done():
					return
				}
			}
		}
	}
}

// typeCheckModule type checks a single module, returning any errors.
func (c *Compiler) typeCheckModule(
	parsedMod *Module, pool *checkerPool,
) []errors.CompileError {
	opts := c.getCheckerOptions(parsedMod)
	mod := analysis.NewModule(
		parsedMod.Name, parsedMod.Path,
		nil, // TODO: import path
		parsedMod.Programs,
		opts.KlarVersion,
		opts.Target,
	)
	ch := pool.Get(mod, opts)
	defer pool.Put(ch)
	ch.Check()
	return ch.Errors
}

// getCheckerOptions returns an [analysis.Options] object for a module. It
// contains options from klar.build and the parent package's manifest.
func (c *Compiler) getCheckerOptions(mod *Module) *analysis.Options {
	conf := c.moduleInputs[mod]
	opts := &analysis.Options{
		CheckerOptions: conf.Options.Checker,
		Target:         &conf.Options.Target,
		// TODO
		KlarVersion: nil,
		Importer:    nil,
	}
	return opts
}

// Pool of [analysis.Checker] objects (similar to parsePool)
// =========

type checkerPool struct{ sync.Pool }

func newCheckerPool() *checkerPool {
	return &checkerPool{sync.Pool{
		New: func() any { return new(analysis.Checker) },
	}}
}

// Get returns an [analysis.Checker] from the pool, initializing it with mod and opts.
func (p *checkerPool) Get(
	mod *analysis.Module, opts *analysis.Options,
) *analysis.Checker {
	ch := p.Pool.Get().(*analysis.Checker)
	ch.Init(mod, opts)
	return ch
}

// Put resets ch and puts it back into the pool.
func (p *checkerPool) Put(ch *analysis.Checker) {
	ch.Reset()
	p.Pool.Put(ch)
}
