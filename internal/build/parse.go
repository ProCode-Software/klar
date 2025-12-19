package build

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	parse "github.com/ProCode-Software/klar/internal/parser"
	"github.com/ProCode-Software/klar/pkg/parser"
)

type parseContext struct {
	*processContext
	fileMu, printerMu sync.Mutex
	pool              *parsePool
}

// Step 3: Parse each file into an untyped AST
// =====

func (c *Compiler) ParseModules(
	procCtx *processContext, numFiles int, moduleCh chan *Module,
) {
	c.Logf("Begin parsing modules (%d modules, %d files)", len(c.modules), numFiles)
	pctx := &parseContext{
		processContext: procCtx,
		pool: newParsePool(lexer.IncludeComments, &parse.ParseOptions{
			MaxErrors: MaxErrors,
		}),
	}
	c.flatFiles = make(map[string]*ast.Program, numFiles)
	for _, mod := range c.modules {
		select {
		case <-procCtx.ctx.Done():
			return
		default:
		}
		var moduleWg sync.WaitGroup
		mod.Programs = make(map[string]*ast.Program, len(mod.Files))
		for _, filePath := range mod.Files {
			// Skip if already parsed
			if _, ok := c.flatFiles[filePath]; ok {
				c.Log("Skipping already parsed file:", filePath)
				continue
			}
			c.Log("Processing file:", filePath)
			moduleWg.Go(func() { c.parseFile(pctx, filePath, mod) })
		}
		moduleWg.Wait()
		moduleCh <- mod
	}
	// All modules are done parsing at this point
	c.flatFiles = nil
	close(moduleCh)
}

// parseFile parses a single file and sends results/errors to channels
func (c *Compiler) parseFile(pctx *parseContext, filePath string, mod *Module) {
	// Check if we should stop due to critical failure
	select {
	case <-pctx.ctx.Done():
		return
	default:
	}
	sendCriticalError := func(err error) {
		select {
		case pctx.fatalErrCh <- err:
		case <-pctx.ctx.Done():
		}
	}
	// Open file
	// ==========
	var (
		fr        io.ReadCloser
		fileSize  int64
		shortPath string
	)
	if filePath == "" {
		const stdinName = "standardInput"
		fr = os.Stdin
		filePath, shortPath = stdinName, stdinName
		c.Log("Reading file from stdin")
	} else {
		f, err := c.Opener.Open(filePath)
		if err != nil {
			c.LogErrorf("Error while opening file %s: %v", filePath, err)
			sendCriticalError(&FilesystemError{"open", filePath, err})
			return
		}
		fr, fileSize, shortPath = f.ReadCloser, f.Size, f.ShortPath
		c.Logf("Opened %s", filePath)
		defer f.Close()
	}

	// Tokenize
	// =========

	// Estimate file size
	lex := pctx.pool.GetLexer(fr)
	defer pctx.pool.PutLexer(lex)
	c.Log("Tokenizing file:", filePath)

	var sizeEstimate int64
	if fileSize > 0 {
		sizeEstimate = fileSize / 10
	}
	toks, err := parser.TokenizeLexer(lex, sizeEstimate)
	if err != nil {
		c.LogErrorf("Error while tokenizing %s: %v", filePath, err)
		sendCriticalError(&InterfaceError{Code: ErrLexer, Err: err})
		return
	}
	// Load tokens into printer for diagnostics
	pctx.printerMu.Lock()
	c.errorPrinter.LoadTokens(filePath, shortPath, toks)
	pctx.printerMu.Unlock()

	// Parse
	// ========

	pa := pctx.pool.GetParser(toks, filePath)
	defer pctx.pool.PutParser(pa)

	c.Log("Parsing file:", filePath)
	ast := pa.Parse()
	errs := pa.Errors

	// Send syntax errors
	if len(errs) > 0 {
		c.LogErrorf("%d errors found while parsing %s", len(errs), filePath)
		select {
		case pctx.errorCh <- convertParseErrors(errs):
		case <-pctx.ctx.Done():
			return
		}
	}
	// Store result to avoid reparsing the same file
	pctx.fileMu.Lock()
	c.flatFiles[filePath] = ast
	mod.Programs[filepath.Base(filePath)] = ast
	pctx.fileMu.Unlock()
}

func convertParseErrors(errs []*errors.ParseError) []errors.CompileError {
	compileErrs := make([]errors.CompileError, len(errs))
	for i, err := range errs {
		compileErrs[i] = err
	}
	return compileErrs
}

// Lexer/parser pool
type parsePool struct{ parser, lexer sync.Pool }

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
	l.Reset()
	p.lexer.Put(l)
}

func (p *parsePool) GetParser(tokens []lexer.Token, file string) *parse.Parser {
	pa := p.parser.Get().(*parse.Parser)
	pa.Tokens = tokens
	pa.Options.File = file
	return pa
}

func (p *parsePool) PutParser(pa *parse.Parser) {
	pa.Reset()
	p.parser.Put(pa)
}
