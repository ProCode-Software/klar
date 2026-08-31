package lsp

import (
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/klarerrs"
	"github.com/ProCode-Software/klar/internal/lexer"

	"github.com/ProCode-Software/klar/internal/ranges"
	"github.com/ProCode-Software/klar/pkg/klon"
	"github.com/ProCode-Software/klar/pkg/lsp"
)

func (s *Server) compileKlarModule(path string) {
	mod := s.modules[path]
	deps := &s.pkgs[mod.PkgPath].deps

	defer s.compiler.ResetState()
	s.compiler.StartTime = time.Now()
	// TODO: Remove the compilation error limit (or set it to 500)
	var fatalErr error
	if *deps == nil {
		// First time compiling the package. Compile it to get the
		// dependency packages. This only occurs once per *package*.
		// For that reason, I don't think that a pool is needed.
		s.Info("Compiling package for the first time", slog.String("module", path))
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
		pkc := projCompilerPool.Get().(*build.PackageCompiler)
		defer func() {
			pkc.Reset()
			projCompilerPool.Put(pkc)
		}()
		pkc.Compiler, pkc.Input, pkc.Deps = s.compiler, mod.compilerInput, deps
		_, fatalErr = pkc.Compile()
	}
	if fatalErr != nil {
		s.Error("Fatal error while compiling modules", slog.Any("error", fatalErr))
		// TODO: Have a timeout so this doesn't show every time the user types
		s.showMessageToUser(lsp.MessageTypeError, fmt.Sprintf(
			"A critical error occurred while compiling %q:\n\n%v",
			path, fatalErr,
		))
		// TODO: Send error notification to user
	}

	// 2. The module's dependencies have been compiled, so we can apply them.
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

		// 3. Clear previous diagnostics
		isInput := mod.Path == path
		for base, prog := range mod.Programs {
			path := mod.FilePath(base)
			if file, ok := s.fs.Files[path]; ok {
				file.Klar.Diagnostics = file.Klar.Diagnostics[:0]
				file.Klar.Module.depsWithDiags = nil
				// 4. Set the AST for each file in the target module
				if isInput {
					file.Klar.AST = prog
				}
			}
		}
	}

	// 5. Attach errors and warnings to their files
	if len(s.compiler.Errors) == 0 && len(s.compiler.Warnings) == 0 {
		return
	}
	filesWithDiags := make(map[string]struct{})
	for _, group := range [...][]*klarerrs.Error{s.compiler.Errors, s.compiler.Warnings} {
		for _, err := range group {
			if file, ok := s.fs.Files[err.File]; ok {
				file.Klar.Diagnostics = append(file.Klar.Diagnostics, err)
				file.Klar.Module.depsWithDiags = filesWithDiags
			}
		}
	}
}

var projCompilerPool = sync.Pool{New: func() any {
	return build.NewPackageCompiler(nil, nil)
}}

// initCompiler initializes the shared compiler instance.
func (s *Server) initCompiler() {
	c := build.NewCompiler(build.ModeAnalyze, "")
	c.FS = s.fs
	c.Logger = s.Logger.With(slog.String("source", "compiler"))
	c.Reporter.Output = io.Discard
	c.UseStdParser()
	s.compiler = c
}

// Klon parsing
// =======

func (s *Server) compileKlonFile(path string) {
	file := s.fs.Files[path]
	parsed, errs := klon.Parse(file.Content)
	file.Klon.AST = parsed
	file.Klon.Diagnostics = errs
	// TODO: Check for undeclared var references and unused vars (hint)
}

// Position mapping
// =======

// A PositionMapper is used for converting UTF-32 [lexer.Position]s from the
// compiler to positions compatible with the LSP client.
type PositionMapper struct {
	// The Unicode code points in the file that take up more than 1 byte
	nonASCII []unicodeChar
}

type unicodeChar struct {
	Pos  lexer.Position
	Size uint8 // Size of the character in bytes. Must be 2-4
}

func (f *File) makePositionMap() {
	var (
		nonASCII []unicodeChar
		i        int
		pos      = lexer.Position{1, 1}
	)
	for i < len(f.Content) {
		r, n := utf8.DecodeRune(f.Content[i:])
		if n > 1 {
			nonASCII = append(nonASCII, unicodeChar{Pos: pos, Size: uint8(n)})
		}
		i += n
		if r == '\n' {
			pos.Line++
			pos.Col = 1
		} else {
			pos.Col++
		}
	}
	if len(nonASCII) > 0 {
		f.PosMapper = &PositionMapper{nonASCII: nonASCII}
	} else {
		f.PosMapper = nil
	}
}

// ToLSPEncoding does not handle converting 1-based positions to the protocol's 0-based positions.
func (pm *PositionMapper) ToLSPEncoding(
	pos lexer.Position, enc lsp.PositionEncodingKind,
) lexer.Position {
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

func (s *Server) toLSPPosition(pos lexer.Position, file string) lsp.Position {
	if pos.IsZero() {
		return lsp.Position{0, 0}
	}
	pos = s.fs.Files[file].PosMapper.ToLSPEncoding(pos, s.caps.posEncoding)
	// LSP positions are 0-based
	return lsp.Position{pos.Line - 1, pos.Col - 1}
}

func (s *Server) toLSPRange(r ranges.Range, file string) lsp.Range {
	return lsp.Range{
		Start: s.toLSPPosition(r.Start, file),
		End:   s.toLSPPosition(r.End, file),
	}
}

func (pm *PositionMapper) ToASTEncoding(
	pos lexer.Position, enc lsp.PositionEncodingKind,
) lexer.Position {
	if pm == nil {
		return pos
	}
	// This does the opposite of [PositionMapper.ToLSPEncoding]
	return pos
}
