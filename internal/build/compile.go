package build

import (
	"context"
	"time"

	"github.com/ProCode-Software/klar/internal/errors"
)

// Compilation stops after exceeding this number of errors.
const MaxErrors = 10

type BuildResult struct {
	Errors []errors.CompileError
	// Time from [Compiler.StartTime] to finish time.
	Elapsed time.Duration
	// Whether the build stopped early due to too many errors
	EarlyExit bool
	Modules   []*Module
}

type processContext struct {
	ctx        context.Context // Cancellation context
	cancel     context.CancelFunc
	done       chan struct{}              // Step complete
	errorCh    chan []errors.CompileError // Diagnostics
	fatalErrCh chan error                 // Critical error
}

// The actual compilation process.

// Compile compiles c's Inputs, returing the result and any critical error
// that occured. err == nil does not mean the build was successful; syntax
// and typecheck errors are stored in [*BuildResult.Errors]
func (c *Compiler) Compile() (res *BuildResult, err error) {
	res = &BuildResult{EarlyExit: true}
	defer func() {
		res.Elapsed = time.Since(c.StartTime)
		res.Modules = c.modules
		res.Errors = c.Errors
	}()
	// Resolve modules
	var totalFiles int
	if totalFiles, err = c.ResolveModules(); err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	moduleCh := make(chan *Module)
	procCtx := &processContext{
		ctx:        ctx,
		cancel:     cancel,
		done:       make(chan struct{}),
		errorCh:    make(chan []errors.CompileError),
		fatalErrCh: make(chan error, 1),
	}
	// Global error collector
	go c.collectErrors(procCtx)
	// Type-check modules (after parsing them)
	go c.TypeCheckModules(procCtx, moduleCh)
	// Parse modules
	go c.ParseModules(procCtx, totalFiles, moduleCh)
	// Wait for type checking to finish
	if err = procCtx.wait(); err != nil {
		return
	}
	res.EarlyExit = false
	println(len(c.Errors))
	return
}

func (procCtx *processContext) wait() error {
	select {
	case <-procCtx.done:
		return nil
	case err := <-procCtx.fatalErrCh:
		return err
	}
}

// collectErrors collects errors from procCtx.errorCh and stores
// them in c.Errors. This function runs in a separate goroutine.
func (c *Compiler) collectErrors(procCtx *processContext) {
	for {
		select {
		case errs, ok := <-procCtx.errorCh:
			if !ok {
				procCtx.done <- struct{}{}
				return
			}
			// If there are too many errors, show only the first [MaxErrors]
			var tooManyErrors bool
			if len(c.Errors)+len(errs) > MaxErrors {
				errs = errs[:MaxErrors-len(c.Errors)]
				tooManyErrors = true
			}
			c.Errors = append(c.Errors, errs...)
			// Stop compilation if there are too many errors
			if tooManyErrors {
				select {
				case procCtx.fatalErrCh <- &InterfaceError{Code: ErrTooManyErrors}:
				case <-procCtx.ctx.Done():
				}
				return
			}
		case <-procCtx.ctx.Done():
			return
		}
	}
}
