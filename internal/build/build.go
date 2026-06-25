package build

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"
)

type (
	WarnLevel uint8
	InputKind int
)

const (
	KindFile InputKind = iota
	KindPackage
	KindModule
	KindStdin
)

const (
	_ WarnLevel = iota
	SuppressWarning
	WarningAsError
)

func CompileString(s, fileName string) (pc *ProjectCompiler, res *Result, err error) {
	cwd, err := Cwd()
	if err != nil {
		return
	}
	pc = NewProjectCompiler(NewCompiler(ModeBuild, cwd))
	pc.Inputs = append(pc.Inputs, &Input{Path: fileName, Kind: KindFile})
	pc.Parser = NewStaticParser(cwd, fileName, &StaticParserFile{Reader: strings.NewReader(s)})
	pc.StartTime = time.Now()
	res, err = pc.Compile()
	return
}

func Cwd() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", &FilesystemError{"determine", "working directory", err}
	}
	return cwd, nil
}

// Parser parses files into untyped ASTs.
type Parser interface {
	// Parse reads and parses the file at the given path and returns the short
	// file path, a [ParseResult] object, and a fatal error if one occurs, such
	// as during reading. If path == "", Parse should read from standard input.
	// l may be used to log status. Parse may be called concurrently.
	Parse(path string, l *slog.Logger, stdin bool) (
		shortPath string, res *ParseResult, err error,
	)
}

type ParseResult struct {
	Tokens  []lexer.Token
	Program *ast.Program
	Errors  []*klarerrs.Error
	ModTime time.Time
}
