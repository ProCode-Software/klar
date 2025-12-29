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

// ParseModules parses all files in all modules, storing each AST in the module.
func (c *Compiler) ParseModules(
	pc *processContext, numFiles int, moduleCh chan *Module,
) {
	defer close(moduleCh)
	c.LogInfof("Begin parsing modules (%d modules, %d files)", len(c.Modules), numFiles)
	pctx := &parseContext{
		processContext: pc,
		pool: newParsePool(lexer.IncludeComments, &parse.ParseOptions{
			MaxErrors: MaxErrors,
		}),
	}
	c.flatFiles = make(map[string]*ast.Program, numFiles)
	for _, mod := range c.Modules {
		select {
		case <-pc.ctx.Done():
			return
		default:
		}
		var moduleWg sync.WaitGroup
		mod.Programs = make(map[string]*ast.Program, len(mod.Files))
		for _, filePath := range mod.Files {
			// Skip if already parsed
			if prog, ok := c.flatFiles[filePath]; ok {
				c.LogInfo("Skipping already parsed file:", filePath)
				mod.Programs[filepath.Base(filePath)] = prog
				continue
			}
			c.LogInfo("Processing file:", filePath)
			moduleWg.Go(func() { c.parseFile(pctx, filePath, mod) })
		}
		moduleWg.Wait()
		// If mod.Programs is less, a parse error occurred in one of the files.
		// In that case, avoid typechecking the entire module.
		if len(mod.Programs) == len(mod.Files) {
			moduleCh <- mod
		}
	}
	// All modules are done parsing at this point
	// TODO: will flatFiles be needed for other build steps?
	c.flatFiles = nil
}

// parseFile parses a single file and sends results/errors to channels
func (c *Compiler) parseFile(pc *parseContext, filePath string, mod *Module) {
	// Check if we should stop due to critical failure
	select {
	case <-pc.ctx.Done():
		return
	default:
	}
	sendCriticalError := func(err error) {
		select {
		case pc.fatalErrCh <- err:
		case <-pc.ctx.Done():
		}
	}
	// Open file
	// ==========
	var (
		fr        io.ReadCloser
		fileSize  int64
		shortPath string
		err       error
		toks      []lexer.Token
		skipLexer bool

		op, canOpenTokens = c.Opener.(TokenOpener)
	)
	switch {
	case filePath == "":
		// Read from standard input
		const stdinName = "standardInput"
		fr = os.Stdin
		filePath, shortPath = stdinName, stdinName
		c.LogInfo("Reading file from stdin")
	case canOpenTokens:
		if toks, shortPath, err = op.OpenTokens(filePath); err == nil {
			skipLexer = true
			break
		}
		fallthrough
	default:
		f, err := c.Opener.Open(filePath)
		if err != nil {
			c.LogErrorf("Error while opening file %s: %v", filePath, err)
			sendCriticalError(&FilesystemError{"open", filePath, err})
			return
		}
		fr, fileSize, shortPath = f.ReadCloser, f.Size, f.ShortPath
		c.LogInfof("Opened %s", filePath)
		defer f.Close()
	}
	// Tokenize
	// =========
	if !skipLexer { // Tokenize unless the opener already provided tokens
		lex := pc.pool.GetLexer(fr)
		defer pc.pool.PutLexer(lex)
		c.LogInfo("Tokenizing file:", filePath)
		// Estimate file size
		var sizeEstimate int64
		if fileSize > 0 {
			sizeEstimate = fileSize / 10
		}
		if toks, err = parser.TokenizeLexer(lex, sizeEstimate); err != nil {
			c.LogErrorf("Error while tokenizing %s: %v", filePath, err)
			sendCriticalError(&InterfaceError{Code: ErrLexer, Err: err})
			return
		}
	}
	// Load tokens into printer for diagnostics
	pc.printerMu.Lock()
	c.errorPrinter.LoadTokens(filePath, shortPath, toks)
	pc.printerMu.Unlock()

	// Parse
	// ========
	pa := pc.pool.GetParser(toks, filePath)
	defer pc.pool.PutParser(pa)

	c.LogInfo("Parsing file:", filePath)
	ast := pa.Parse()
	errs := pa.Errors

	// Send syntax errors
	if len(errs) > 0 {
		c.LogErrorf("%d errors found while parsing %s", len(errs), filePath)
		select {
		case pc.errorCh <- convertParseErrors(errs):
		case <-pc.ctx.Done():
		}
		return
	}
	// Store result to avoid reparsing the same file
	pc.fileMu.Lock()
	c.flatFiles[filePath] = ast
	mod.Programs[filepath.Base(filePath)] = ast
	pc.fileMu.Unlock()
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
