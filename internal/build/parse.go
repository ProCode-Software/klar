package build

import (
	"bufio"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/lexer"
	parse "github.com/ProCode-Software/klar/internal/parser"
	"github.com/ProCode-Software/klar/pkg/parser"
)

// Parser parses files into untyped ASTs.
type Parser interface {
	// Parse reads and parses the file at the given path and retuns the tokens,
	// program, parse errors, and a fatal error if one occurs, such as during
	// reading. If path == "", Parse should read from standard input. l may be
	// used to log status. Parse may be called concurrently.
	Parse(path string, l *slog.Logger) ([]lexer.Token, *ast.Program, []*errors.ParseError, error)
}

type Logger interface {
	LogInfo(msg string, attrs ...any)
	LogError(msg string, attrs ...any)
}

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
	c.LogInfo("Begin parsing modules", slog.Int("modules", len(c.Modules)),
		slog.Int("files", numFiles),
	)
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
				c.LogInfo("Skipping already parsed file", slog.String("path", filePath))
				mod.Programs[filepath.Base(filePath)] = prog
				continue
			}
			c.LogInfo("Processing file", slog.String("path", filePath))
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
	if c.Parser == nil {
		panic("Parser not set up")
	}
	tokens, program, errs, fatalErr := c.Parser.Parse(filePath, c.Logger)
	if fatalErr != nil {
		sendCriticalError(fatalErr)
	}
	// Send syntax errors
	if len(errs) > 0 {
		c.LogError("Errors found while parsing file",
			slog.Int("errors", len(errs)), slog.String("file", filePath),
		)
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

// StdParser is the default [Parser] implementation for Klar.
type StdParser struct {
	*parsePool
}

func (p *StdParser) Parse(filePath string) (
	tokens []lexer.Token, program *ast.Program, errs []errors.ParseError, fatalErr error,
) {
	// Open file
	// ==========
	var (
		fr        io.ReadCloser
		fileSize  int64
		shortPath string
		err       error
		toks      []lexer.Token
		f         *OpenFile
	)
	
	if filePath == "" {
		// Read from standard input
		const stdinName = "standardInput"
		fr = os.Stdin
		filePath, shortPath = stdinName, stdinName
		c.LogInfo("Reading file from stdin")
	} else {
		f, err := c.Opener.Open(filePath)
		if err != nil {
			c.LogError("Error while opening file",
				slog.String("file", filePath), slog.Any("error", err),
			)
			sendCriticalError(&FilesystemError{"open", filePath, err})
			return
		}
		fr, fileSize, shortPath = f.ReadCloser, f.Size, f.ShortPath
		c.LogInfo("Successfully opened file", slog.String("file", filePath))
		defer f.Close()
	}
	// Tokenize
	// =========
	if !skipLexer { // Tokenize unless the opener already provided tokens
		lex := pc.pool.GetLexer(fr)
		defer pc.pool.PutLexer(lex)
		c.LogInfo("Tokenizing file", slog.String("path", filePath))
		// Estimate file size
		var sizeEstimate int64
		if fileSize > 0 {
			sizeEstimate = fileSize / 10
		}
		if toks, err = parser.TokenizeLexer(lex, sizeEstimate); err != nil {
			c.LogError("Error while tokenizing file",
				slog.String("file", filePath), slog.Any("error", err),
			)
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

	c.LogInfo("Parsing file", slog.String("path", filePath))
	ast := pa.Parse()
	errs := pa.Errors
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
