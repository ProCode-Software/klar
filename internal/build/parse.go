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
	"github.com/ProCode-Software/klar/internal/parser"
	pkgparser "github.com/ProCode-Software/klar/pkg/parser"
)

// Step 3: Parse each file into an untyped AST
// =====

// ParseModules parses all files in all modules, storing each AST in the module.
func (c *Compiler) ParseModules(
	pc *processContext, numFiles int, moduleCh chan *Module,
) {
	defer close(moduleCh)
	c.Info("Begin parsing modules", slog.Int("modules", len(c.Modules)),
		slog.Int("files", numFiles),
	)
	var fileMu sync.Mutex
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
			// Cache "file" attribute
			fl := c.Logger.With(slog.String("file", filePath))
			// Skip if already parsed
			if prog, ok := c.flatFiles[filePath]; ok {
				fl.Info("Skipping already parsed file")
				mod.Programs[filepath.Base(filePath)] = prog
				continue
			}
			fl.Info("Processing file")
			moduleWg.Go(func() { c.parseFile(pc, filePath, mod, fl, &fileMu) })
		}
		moduleWg.Wait()
		// If mod.Programs is less, a parse error occurred in one of the files.
		// In that case, avoid typechecking the entire module.
		if len(mod.Programs) == len(mod.Files) {
			moduleCh <- mod
		}
	}
	c.Info("Finished parsing files")
	// All modules are done parsing at this point
	// TODO: will flatFiles be needed for other build steps?
	c.flatFiles = nil
}

// parseFile parses a single file and sends results/errors to channels
func (c *Compiler) parseFile(pc *processContext,
	filePath string, mod *Module, l *slog.Logger, mu *sync.Mutex,
) {
	// Check if we should stop due to critical failure
	select {
	case <-pc.ctx.Done():
		return
	default:
	}
	if c.Parser == nil {
		panic("Parser not set up")
	}
	shortPath, res, err := c.Parser.Parse(filePath, l)
	if err != nil {
		select {
		case pc.fatalErrCh <- err:
		case <-pc.ctx.Done():
		}
		return
	}

	// Load the tokens for diagnostics first
	mu.Lock()
	c.Reporter.LoadFile(filePath, shortPath, res.Tokens)
	// Store result to avoid reparsing the same file
	if len(res.Errors) == 0 {
		c.flatFiles[filePath] = res.Program
		mod.Programs[filepath.Base(filePath)] = res.Program
	}
	mu.Unlock()

	// Send syntax errors
	if len(res.Errors) > 0 {
		c.Error("Errors found while parsing file",
			slog.Int("errors", len(res.Errors)), slog.String("file", filePath),
		)
		select {
		case pc.errorCh <- convertParseErrors(res.Errors):
		case <-pc.ctx.Done():
		}
		return
	}
}

// Standard parser implementation
// ==============================

// StdParser is the default [Parser] implementation for Klar.
type StdParser struct {
	*parsePool
	cwd string
}

func NewStdParser(cwd string, parseOpts *parser.Options) *StdParser {
	return &StdParser{parsePool: newParsePool(parseOpts), cwd: cwd}
}

func (p *StdParser) Reset() {
	p.parsePool = nil
	p.cwd = ""
}

func (p *StdParser) Parse(filePath string, l *slog.Logger) (
	shortPath string, res *ParseResult, err error,
) {
	// Open file
	// ==========
	var f *os.File
	var sizeEst int64
	if filePath == "" {
		// Read from standard input
		const stdinName = "standardInput"
		f = os.Stdin
		filePath, shortPath = stdinName, stdinName
		l.Info("Reading file from stdin")
	} else {
		f, err = os.Open(filePath)
		if err != nil {
			l.Error("Error while opening file", slog.Any("error", err))
			return "", nil, &FilesystemError{"open", filePath, err}
		}
		defer f.Close()
		// Get file size
		stat, err := f.Stat()
		if err != nil {
			l.Error("Error while getting file info", slog.Any("error", err))
			return shortPath, nil, &FilesystemError{"stat", filePath, err}
		}
		sizeEst = stat.Size() / 10
		// Get relative path
		if shortPath, err = filepath.Rel(p.cwd, filePath); err != nil {
			l.Warn("Could not get short path for file", slog.Any("error", err))
			shortPath = filePath
		}
		l.Info("Successfully opened file")
	}
	res = &ParseResult{}
	// Tokenize
	// =========
	lex := p.GetLexer(f)
	defer p.PutLexer(lex)
	l.Info("Tokenizing file")

	pkgparser.TokenizeLexer(lex, sizeEst)

	// Parse
	// ========
	pa := p.GetParser(res.Tokens, filePath)
	defer p.PutParser(pa)

	l.Info("Parsing file")
	res.Program = pa.Parse()
	res.Errors = pa.Errors
	return shortPath, res, nil
}

func convertParseErrors(errs []*errors.ParseError) []errors.CompileError {
	compileErrs := make([]errors.CompileError, len(errs))
	for i, err := range errs {
		compileErrs[i] = err
	}
	return compileErrs
}

// Lexer/parser pool
// =================

// parsePool provides a pool of [lexer.Lexer] and [parser.Parser].
type parsePool struct{ parser, lexer sync.Pool }

// newParsePool creates a new [parsePool] with the provided
// [lexer.Flags] and [pkgparser.Options] as defaults.
func newParsePool(parseOpts *parser.Options) *parsePool {
	return &parsePool{
		lexer: sync.Pool{New: func() any {
			return lexer.NewLexer(nil)
		}},
		parser: sync.Pool{New: func() any {
			return parser.New(nil, parseOpts)
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

func (p *parsePool) GetParser(tokens []lexer.Token, file string) *parser.Parser {
	pa := p.parser.Get().(*parser.Parser)
	pa.Tokens = tokens
	pa.Options.File = file
	return pa
}

func (p *parsePool) PutParser(pa *parser.Parser) {
	pa.Reset()
	p.parser.Put(pa)
}
