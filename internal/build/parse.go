package build

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	parse "github.com/ProCode-Software/klar/internal/parser"
	"github.com/ProCode-Software/klar/pkg/parser"
)

const (
	maxErrors = 10
	stdinName = "standardInput"
)

type parseContext struct {
	ctx           context.Context
	cancel        context.CancelFunc
	syntaxErrCh   chan []*errors.ParseError
	criticalErrCh chan error
	collectorDone chan struct{}
	fileMu        sync.Mutex
	printerMu     sync.RWMutex
	wg            sync.WaitGroup
	pool          *parsePool
	cwd           string
}

// Step 3: Parse each file into an untyped AST
// =====

func (c *Compiler) ParseModules(numFiles int) (
	syntaxErrors []*errors.ParseError, criticalErr error,
) {
	// Initialize parse context
	c.Logf("Begin parsing modules (%d modules, %d files)", len(c.Modules), numFiles)
	pctx := &parseContext{
		syntaxErrCh:   make(chan []*errors.ParseError),
		criticalErrCh: make(chan error, 1),
		collectorDone: make(chan struct{}),
		pool: newParsePool(lexer.IncludeComments, &parse.ParseOptions{
			MaxErrors: maxErrors,
		}),
	}
	pctx.ctx, pctx.cancel = context.WithCancel(context.Background())
	defer pctx.cancel()

	// Get working directory to shorten file paths to relative paths
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	pctx.cwd = cwd

	// Init files
	c.FlatFiles = make(map[string]*File, numFiles)

	// Start error collector
	go pctx.collectErrs(&syntaxErrors, &criticalErr)

	// Parse all module files
	for _, module := range c.Modules {
		for _, filePath := range module.Files {
			// Skip if already parsed
			if _, ok := c.FlatFiles[filePath]; ok {
				c.Log("Skipping already parsed file:", filePath)
				continue
			}
			c.Log("Processing file:", filePath)
			pctx.wg.Go(func() { c.parseFile(pctx, filePath) })
		}
	}

	// Wait and cleanup
	pctx.wg.Wait()
	close(pctx.syntaxErrCh)
	close(pctx.criticalErrCh)
	<-pctx.collectorDone

	return syntaxErrors, criticalErr
}

// collectErrs runs in a separate goroutine to aggregate errors from channels
func (pctx *parseContext) collectErrs(syntaxErrors *[]*errors.ParseError, criticalErr *error) {
	defer close(pctx.collectorDone)
	for {
		select {
		case errs, ok := <-pctx.syntaxErrCh:
			if !ok {
				// Channel closed, drain criticalErrCh and exit
				select {
				case err := <-pctx.criticalErrCh:
					if *criticalErr == nil {
						*criticalErr = err
					}
				default:
				}
				return
			}
			// Too many errors (single file)
			if l := len(errs); l > 0 &&
				errs[l-1].GetCode() == errors.ErrTooManyErrors {
				errs = errs[:l-1]
			}
			*syntaxErrors = append(*syntaxErrors, errs...)
			// Too many errors (global)
			if len(*syntaxErrors) >= maxErrors {
				if *criticalErr == nil {
					*criticalErr = &InterfaceError{Code: ErrTooManyErrors}
				}
				pctx.cancel()
				return
			}
		case err := <-pctx.criticalErrCh:
			if *criticalErr == nil {
				*criticalErr = err
				pctx.cancel()
			}
		}
	}
}

// parseFile parses a single file and sends results/errors to channels
func (c *Compiler) parseFile(pctx *parseContext, filePath string) {
	// Check if we should stop due to critical failure
	select {
	case <-pctx.ctx.Done():
		return
	default:
	}

	sendCriticalError := func(err error) {
		select {
		case pctx.criticalErrCh <- err:
		case <-pctx.ctx.Done():
		}
	}

	// Open file
	var fr *os.File
	var err error
	if filePath == "" {
		fr = os.Stdin
		filePath = stdinName
		c.Log("Reading file from stdin")
	} else {
		fr, err = os.Open(filePath)
		if err != nil {
			c.LogErrorf("Error while opening file %s: %v", filePath, err)
			sendCriticalError(err)
			return
		}
		c.Logf("Opened %s", filePath)
		defer fr.Close()
	}

	// === Tokenize ===

	// Estimate file size
	stat, err := fr.Stat()
	if err != nil {
		sendCriticalError(err)
		return
	}
	lex := pctx.pool.GetLexer(fr)
	defer pctx.pool.PutLexer(lex)
	c.Log("Tokenizing file:", filePath)

	toks, err := parser.TokenizeLexer(lex, stat.Size()/10)
	if err != nil {
		c.LogErrorf("Error while tokenizing %s: %v", filePath, err)
		sendCriticalError(err)
		return
	}

	var relPath string
	if filePath != stdinName {
		relPath, err = filepath.Rel(pctx.cwd, filePath)
		if err != nil {
			c.LogErrorf("Unable to get short path of %s: %v", filePath, err)
			sendCriticalError(err)
			return
		}
	} else {
		relPath = stdinName
	}
	pctx.printerMu.Lock()
	c.ErrorPrinter.LoadTokens(filePath, relPath, toks)
	pctx.printerMu.Unlock()

	// === Parse ===
	pa := pctx.pool.GetParser(toks, filePath)
	defer pctx.pool.PutParser(pa)

	c.Log("Parsing file:", filePath)
	ast := pa.Parse()
	errs := pa.Errors

	// Send syntax errors
	if len(errs) > 0 {
		c.LogErrorf("%d errors found while parsing %s", len(errs), filePath)
		select {
		case pctx.syntaxErrCh <- errs:
		case <-pctx.ctx.Done():
			return
		}
	}

	// Store result
	pctx.fileMu.Lock()
	c.FlatFiles[filePath] = &File{
		Path:   filePath,
		AST:    ast,
		Tokens: toks,
	}
	pctx.fileMu.Unlock()
}

// Lexer/parser pool
type parsePool struct {
	parser, lexer sync.Pool
}

func newParsePool(lexFlags lexer.Flags, parseOpts *parser.Options) *parsePool {
	return &parsePool{
		lexer: sync.Pool{New: func() any {
			return lexer.NewLexer(nil, lexFlags)
		}},
		parser: sync.Pool{New: func() any {
			return parse.New(nil, parseOpts)
		}},
	}
}

func (p *parsePool) GetLexer(r io.Reader) *lexer.Lexer {
	l := p.lexer.Get().(*lexer.Lexer)
	l.Reader = bufio.NewReader(r)
	return l
}

func (p *parsePool) PutLexer(l *lexer.Lexer) {
	l.Free()
	p.lexer.Put(l)
}

func (p *parsePool) GetParser(tokens []lexer.Token, file string) *parse.Parser {
	pa := p.parser.Get().(*parse.Parser)
	pa.Tokens = tokens
	pa.Options.File = file
	return pa
}

func (p *parsePool) PutParser(pa *parse.Parser) {
	pa.Free()
	p.parser.Put(pa)
}
