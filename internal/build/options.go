package build

import (
	"io"
	"log"
	"os"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/module"
	"github.com/ProCode-Software/klar/internal/target"
)

type (
	InputKind int
	BuildMode int
	Flags     int16
)

const (
	ModeBuild   BuildMode = iota // Full compilation
	ModRun                       // Build to cache only
	ModeAnalyze                  // Typed AST only: test, typecheck, LSP
	ModeParse                    // Untyped + resolved AST: format
)

const (
	CreateJSDoc Flags = 1 << iota
	CreateDeclaration
	Minify
	CreateSourceMap
	CopyNodeModules
	BundleDeclaration
	UseESNext
)

const (
	KindDir InputKind = 1 << iota
	KindFile

	KindPackage = KindDir | (1 << iota)
	KindModule  = KindDir | (1 << iota)
	KindStdin   = KindFile | (1 << iota)
)

type Options struct {
	Inputs []Input
	BuildFile
	// ProjectDir   string
}

type Input struct {
	Kind InputKind
	Parent string // Parent module
	Path string
}

type Compiler struct {
	Mode                BuildMode
	Target              target.Target
	Verbose             bool
	Errors              []errors.CompileError
	Options             []*Options
	Project             *module.ProjectInfo
	PreBuild, PostBuild []any // TODO
	OpenFiles           []*os.File

	SuppressWarnings, WarningsAsErrors []string // TODO: better type
	*log.Logger
}

// Logging
// ==========

func (c *Compiler) InitLogger() (hasLogFile bool) {
	logFile := os.Getenv("KLAR_LOG_FILE")
	var out io.Writer
	switch {
	case logFile != "":
		file, err := os.Create(logFile)
		if err != nil {
			cli.Failure("Unable to open KLAR_LOG_FILE '"+logFile+"': ", err)
		}
		c.OpenFiles = append(c.OpenFiles, file)
		out = file
	case c.Verbose:
		out = os.Stderr
	default:
		out = io.Discard
	}
	c.Logger = log.New(out, "[compiler] ", log.Ltime)
	return
}

// Equivalent to c.Logger.Println
func (c *Compiler) Log(v ...any) {
	if c.Verbose {
		c.Println(v...)
	}
}

func (c *Compiler) Errorf(s string, v ...any) {
	if c.Verbose {
		c.Printf("[error] "+s, v...)
	}
}

func (c *Compiler) CloseAll() {
	for _, file := range c.OpenFiles {
		file.Close()
	}
}

func ResolveInputs(inputs []string) (res []Input, err error) {
	res = make([]Input, 0, len(inputs)*2)
	for _, input := range inputs {
		switch {
		case len(input) == 0:
			continue
		case input == "-":
			res = append(res, Input{Kind: KindStdin, Path: ""})
		case input[0] == '@':
			kind := KindPackage
		default:

		}
		kind := KindFile
		if input[len(input)-1] == '/' {
			kind = KindDir
		}
		res = append(res, Input{Kind: kind, Path: input})
	}
}