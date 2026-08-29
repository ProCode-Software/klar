package lsp

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/lexer"
	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/lsp"
)

func (s *Server) compileKlarModule(path string) {
	mod := s.modules[path]
	deps := &s.pkgs[mod.PkgPath].deps

	defer s.compiler.ResetState()
	s.compiler.StartTime = time.Now()
	// TODO: Remove the compilation error limit
	var fatalErr error
	if *deps == nil {
		// First time compiling the package. Compile it to get the
		// dependency packages. This only occurs once per *package*.
		// For that reason, I don't think that a pool is needed.
		pc := build.NewProjectCompiler(s.compiler)
		pc.Inputs = []*build.Input{mod.compilerInput}
		var res *build.Result
		res, fatalErr = pc.Compile()
		if inputDeps := res.AllModules[mod.compilerInput]; inputDeps != nil {
			*deps = *inputDeps
		}
	} else {
		// TODO: Should each module get its own permanent PackageCompiler?
		// It could possibly be deleted when the module is closed.
		pkc := compilerPool.Get()
		defer compilerPool.Put(pkc)
		pkc.Compiler, pkc.Input, pkc.Deps = s.compiler, mod.compilerInput, deps
		_, fatalErr = pkc.Compile()
	}
	if fatalErr != nil {
		s.Error("Fatal error while compiling modules", slog.Any("error", fatalErr))
		// TODO: Have a timeout so this doesn't show every time the user types
		s.showMessageToUser(lsp.MessageTypeError, fmt.Sprintf(
			"A critical error occured while compiling %q:\n\n%v",
			path, fatalErr,
		))
		// TODO: Send error notification to user
	}

	// The module's dependencies have been compiled, so we can apply them.
	// As a result, they don't have to be compiled when they're first opened.
	//
	// The deps map from the compiler includes the input, so this also applies
	// the typechecked module.
	for _, mod := range *deps {
		if _, ok := s.modules[mod.Path]; !ok {
			// [Module.compilerInput] will be set when the module is opened
			s.modules[mod.Path] = &Module{PkgPath: mod.Path}
		}
		s.modules[mod.Path].Module = mod.Checked

		// Set the AST for each file in the target module
		if mod.Path == path {
			for base, prog := range mod.Programs {
				path := mod.FilePath(base)
				if file, ok := s.fs.Files[path]; ok {
					file.Klar.AST = prog
				}
			}
		}
	}

	// Report diagnostics
	s.reportDiagnostics(s.compiler.Errors, s.compiler.Warnings)
}

// initCompiler initializes the shared compiler instance.
func (s *Server) initCompiler() {
	c := build.NewCompiler(build.ModeAnalyze, "")
	c.FS = s.fs
	c.Logger = s.Logger.With(slog.String("source", "compiler"))
	c.UseStdParser()
	s.compiler = c
}

// Pool for module compilers
// ======

var compilerPool = _compilerPool{New: func() any {
	return build.NewPackageCompiler(nil, nil)
}}

type _compilerPool struct {
	sync.Pool
}

func (pool *_compilerPool) Get() *build.PackageCompiler {
	return pool.Pool.Get().(*build.PackageCompiler)
}

func (pool *_compilerPool) Put(pkc *build.PackageCompiler) {
	pkc.Reset()
	pool.Pool.Put(pkc)
}

// Position mapping
// =======

// A PositionMapper is used for converting UTF-32 [lexer.Position]s from the
// compiler to positions compatible with the LSP client.
type PositionMapper struct {
	// The Unicode code points in the file that take up more than 1 byte
	nonASCII []struct {
		Pos  lexer.Position
		Size uint8 // Size of the character in bytes. Must be 2-4
	}
}

// Map does not handle converting 1-based positions to the protocol's 0-based positions.
func (pm *PositionMapper) Map(pos lexer.Position, enc lsp.PositionEncodingKind) lexer.Position {
	if pm == nil {
		return pos
	}
	var addedCols uint32
	for _, c := range pm.nonASCII {
		if ranges.ComparePos(c.Pos, pos) >= 1 {
			break
		}
		switch enc {
		case lsp.PositionEncodingUTF8:
			addedCols += uint32(c.Size) - 1
		case lsp.PositionEncodingUTF16:
			// In UTF-32, only 4-byte characters are 2 UTF-16 code units long.
			// Also remember we're always subtracting 1
			if c.Size >= 4 {
				addedCols += 1
			}
		}
	}
	return pos.Add(0, addedCols)
}
