package build

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"

	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/parser"
	"github.com/ProCode-Software/klar/internal/util"
)

// Compilation stops after exceeding this number of errors.
const MaxErrors = 10

var errMaxErrors = errors.New("max errors reached")

func (c *Compiler) parseFile(m *Module, file string,
	reporterMu, moduleMu *sync.Mutex,
) error {
	// Use [StdParser] if [Compiler.Parser] isn't set
	if c.Parser == nil {
		c.UseStdParser()
	}

	path := m.FilePath(file)
	c.Debug("Parsing file", slog.String("file", path))
	shortPath, res, err := c.Parser.Parse(path, c.Logger)
	if err != nil {
		return err
	}
	moduleMu.Lock()
	hasErrors, maxErrors := c.sendErrors(res.Errors)
	if hasErrors {
		m.Failed = true
		c.Error("File has syntax errors", slog.String("file", path))
	}
	m.Programs[file] = res.Program
	m.ModTimes[file] = res.ModTime
	moduleMu.Unlock()

	// Load tokens into error reporter
	reporterMu.Lock()
	if m.IsStdin() {
		path = stdinName
	}
	c.Reporter.LoadFile(path, shortPath, res.Tokens)
	reporterMu.Unlock()

	if maxErrors {
		return errMaxErrors
	}
	return nil
}

// Standard parser implementation
// ========

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

const stdinName = "standardInput"

func (p *StdParser) Parse(filePath string, l *slog.Logger) (
	shortPath string, res *ParseResult, err error,
) {
	// Open file
	// ==========
	var f *os.File
	var sizeEst int64
	res = &ParseResult{}
	if filePath == "" {
		// Read from standard input
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
		// Get file size and last modified time
		stat, err := f.Stat()
		if err != nil {
			l.Error("Error while getting file info", slog.Any("error", err))
			return shortPath, nil, &FilesystemError{"stat", filePath, err}
		}
		res.ModTime = stat.ModTime()
		sizeEst = stat.Size() / 10
		shortPath = util.RelPath(p.cwd, filePath) // Get relative path
	}

	// Tokenize
	// =========
	lex := p.GetLexer(f)
	defer p.PutLexer(lex)
	res.Tokens = lex.TokenizeAll(sizeEst)

	// Parse
	// ========
	pa := p.GetParser(res.Tokens, filePath)
	defer p.PutParser(pa)
	res.Program = pa.Parse()
	res.Errors = pa.Errors
	return shortPath, res, nil
}

// Lexer/parser pool
// ========

// parsePool provides a pool of [lexer.Lexer] and [parser.Parser].
type parsePool struct{ parser, lexer sync.Pool }

// newParsePool creates a new [parsePool] with the provided
// [lexer.Flags] and [parser2.Options] as defaults.
func newParsePool(parseOpts *parser.Options) *parsePool {
	return &parsePool{
		lexer:  sync.Pool{New: func() any { return lexer.NewLexer(nil) }},
		parser: sync.Pool{New: func() any { return parser.New(nil, parseOpts) }},
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
