package build

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/parser"
)

// StaticParser is a [Parser] implementation that parses only a set of files.
// A reader, tokens, or an [ast.Program] may be provided for each file.
type StaticParser struct {
	Files map[string]*StaticParserFile
}

type StaticParserFile struct {
	Tokens    []lexer.Token
	Reader    io.Reader
	ShortPath string
	Program   *ast.Program
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
		return path, nil, fmt.Errorf("load file %s: %w", path, os.ErrNotExist)
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
		f.Tokens = lexer.NewLexer(f.Reader).TokenizeAll(size / 10)
		fallthrough
	default:
		// Need to parse
		pa := parser.New(f.Tokens, nil)
		pa.Options.File = path
		prog := pa.Parse()
		return f.ShortPath, &ParseResult{
			Tokens:  f.Tokens,
			Program: prog,
			Errors:  pa.Errors,
		}, nil
	}
}
