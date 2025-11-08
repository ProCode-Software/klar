package build

import (
	"context"
	"os"
	"sync"

	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/pkg/parser"
)

func (c *Compiler) ParseModules() (syntaxErrors []*errors.ParseError, criticalErr error) {
	if c.FlatFiles == nil {
		c.FlatFiles = make(map[string]*File)
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	// Context for cancellation on critical failure
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	criticalError := func(err error) {
		mu.Lock()
		if criticalErr == nil {
			criticalErr = err
			cancel()
		}
		mu.Unlock()
	}

	// Iterate over all modules
	for _, module := range c.Modules {
		// Process each file in the module
		for _, filePath := range module.Files {
			wg.Go(func() {
				// Check if we should stop due to critical failure
				select {
				case <-ctx.Done():
					return
				default:
				}
				var (
					fr   *os.File
					err  error
					file = &File{Path: filePath}
				)
				if filePath == "" {
					fr = os.Stdin
				} else {
					fr, err = os.Open(filePath)
					if err != nil {
						criticalError(err)
						return
					}
				}
				c.OpenFiles = append(c.OpenFiles, fr)

				// Create lexer tokens
				toks, err := parser.TokenizeFile(fr, parser.IncludeComments)
				if err != nil {
					criticalError(err)
					return
				}
				// Load into error printer
				c.ErrorPrinter.LoadTokens(filePath, toks)

				// Parse
				ast, errs := parser.Parse(toks, &parser.Options{
					MaxErrors: 10,
					File: filePath,
				})

				// Example: Collect syntax errors
				if len (errs) > 0 {
				    mu.Lock()
				    syntaxErrors = append(syntaxErrors, errs...)
				    mu.Unlock()
				}

				// Add to FlatFiles map (thread-safe)
				mu.Lock()
				c.FlatFiles[filePath] = file
				mu.Unlock()
			})
		}
	}

	// Wait for all parsing to complete
	wg.Wait()

	return syntaxErrors, criticalErr
}
