package build

import (
	"context"
	"time"

	"github.com/ProCode-Software/klar/internal/klarerrs"
)

// Compilation stops after exceeding this number of errors.
const MaxErrors = 10

type Result struct {
	Errors []*klarerrs.Error
	// Time from [Compiler.StartTime] to finish time.
	Elapsed time.Duration `json:"elapsedTime,format:units"`
	Modules []*Module
}

type processContext struct {
	ctx        context.Context // Cancellation context
	cancel     context.CancelFunc
	done       chan struct{}              // Step complete
	errorCh    chan []*klarerrs.Error // Diagnostics
	fatalErrCh chan error                 // Critical error
}

// The actual compilation process.

// Compile compiles c's Inputs, returing the result and any critical error
// that occurred. err == nil does not mean the build was successful; syntax
// and typecheck errors are stored in [*Result.Errors]
func (c *Compiler) Compile() (res *Result, err error) {
	res = &Result{}
	defer func() {
		res.Elapsed = time.Since(c.StartTime)
		res.Modules = c.Modules
		res.Errors = c.Errors
	}()
	// Resolve modules
	var totalFiles int
	if totalFiles, err = c.ResolveModules(); err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Stage 1: Parse and Type-check
	moduleCh := make(chan *Module)
	collectDone := make(chan struct{}, 1)
	pc := &processContext{
		ctx:        ctx,
		cancel:     cancel,
		done:       make(chan struct{}, 1),
		errorCh:    make(chan []*klarerrs.Error),
		fatalErrCh: make(chan error, 1),
	}
	// Global error collector
	go c.collectErrors(pc, collectDone)
	// Type-check modules as they are parsed into moduleCh
	go c.TypeCheckModules(pc, moduleCh)
	// Parse modules
	go c.ParseModules(pc, totalFiles, moduleCh)
	// Wait for type checking to finish
	if err = pc.wait(); err != nil {
		return
	}
	close(pc.errorCh)
	<-collectDone // Make sure errors are appended to c.Errors
	if len(pc.fatalErrCh) > 0 {
		err = <-pc.fatalErrCh
	}
	return
}

func (pc *processContext) wait() error {
	select {
	case <-pc.done:
		return nil
	case <-pc.ctx.Done():
		return pc.ctx.Err()
	case err := <-pc.fatalErrCh:
		return err
	}
}

// collectErrors collects errors from pc.errorCh and stores
// them in c.Errors. This function runs in a separate goroutine.
func (c *Compiler) collectErrors(pc *processContext, done chan struct{}) {
	defer close(done)
	for {
		select {
		case errs, ok := <-pc.errorCh:
			if !ok {
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
				pc.cancel()
				select {
				case pc.fatalErrCh <- &InterfaceError{Code: ErrTooManyErrors}:
				case <-pc.ctx.Done():
				}
				return
			}
		case <-pc.ctx.Done():
			return
		}
	}
}
