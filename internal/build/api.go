package build

import (
	"bytes"
	"io"
	"log/slog"
	"os"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/errors/printer"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/parser"
	pkgparse "github.com/ProCode-Software/klar/pkg/parser"
)

func NewCompiler(mode BuildMode) (*Compiler, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, &FilesystemError{"get", "working directory", err}
	}
	return &Compiler{
		Mode:         mode,
		errorPrinter: &printer.Printer{MaxLines: 3, Color: true},
		Logger:       slog.New(slog.DiscardHandler),
		WorkDir:      cwd,
	}, nil
}

// PrintError prints an error to the error printer.
func (c *Compiler) PrintError(err errors.CompileError) {
	c.errorPrinter.PrintError(err)
}

// AddInputs adds the given inputs to a new [Options] inside c.
func (c *Compiler) AddInputs(inputs ...Input) {
	c.Options = append(c.Options, &Options{Inputs: inputs})
}

// Parser parses files into untyped ASTs.
type Parser interface {
	// Parse reads and parses the file at the given path and returns the short
	// file path, a [ParseResult] object, and a fatal error if one occurs, such
	// as during reading. If path == "", Parse should read from standard input.
	// l may be used to log status. Parse may be called concurrently.
	Parse(path string, l *slog.Logger) (shortPath string, res *ParseResult, err error)
}

type ParseResult struct {
	Tokens  []lexer.Token
	Program *ast.Program
	Errors  []*errors.ParseError
}

// UseStdParser sets c's Parser to the standard parser [StdParser].
func (c *Compiler) UseStdParser() {
	c.Parser = NewStdParser(c.WorkDir, lexer.IncludeComments,
		&parser.Options{MaxErrors: MaxErrors + 1},
	)
}

type StaticParserFile struct {
	Tokens    []lexer.Token
	Reader    io.Reader
	ShortPath string
	Program   *ast.Program
}

// StaticParser is a [Parser] implementation that parses only a set of files.
// A reader, tokens, or an [ast.Program] may be provided for each file.
type StaticParser struct {
	Files map[string]*StaticParserFile
}

// NewStaticParser creates a new [StaticParser] that parses one file.
func NewStaticParser(path string, f *StaticParserFile) *StaticParser {
	return &StaticParser{map[string]*StaticParserFile{path: f}}
}

func (p *StaticParser) LoadFile(path string, f *StaticParserFile) {
	if p.Files == nil {
		p.Files = make(map[string]*StaticParserFile)
	}
	p.Files[path] = f
}

// Parse implements [Parser]. It returns [os.ErrNotExist] if path
// is not found in the StaticParser's file map.
func (p *StaticParser) Parse(path string, l *slog.Logger) (
	shortPath string, res *ParseResult, err error,
) {
	f, ok := p.Files[path]
	if !ok {
		return path, nil, os.ErrNotExist
	}
	if f.ShortPath == "" {
		f.ShortPath = path
	}
	switch {
	case f.Program != nil:
		// Program already provided
		if f.Tokens == nil {
			panic("both Program and Tokens must be provided")
		}
		return f.ShortPath, &ParseResult{Tokens: f.Tokens, Program: f.Program}, nil
	case f.Tokens == nil:
		var size int64
		switch r := f.Reader.(type) {
		case nil:
			panic("Reader must be provided if Tokens == nil")
		case *os.File:
			if stat, err := r.Stat(); err == nil {
				size = stat.Size()
			}
		case *bytes.Buffer:
			size = int64(r.Len())
		}
		f.Tokens, err = pkgparse.Tokenize(f.Reader, lexer.IncludeComments, size/10)
		if err != nil {
			return path, nil, err
		}
		fallthrough
	default:
		// Need to parse
		pa := parser.New(f.Tokens, nil)
		pa.Options.File = path
		prog := pa.Parse()
		return f.ShortPath, &ParseResult{
			Tokens: f.Tokens,
			Program: prog,
			Errors: pa.Errors,
		}, nil
	}
}
