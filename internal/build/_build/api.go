package build

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/pkg/klarerrors/reporter"
)

func NewCompiler(mode BuildMode, cwd string) *Compiler {
	return &Compiler{
		Mode:    mode,
		WorkDir: cwd,
		Reporter: &reporter.Reporter{
			MaxLines:     3,
			Output:       os.Stderr,
			ColorPalette: reporter.DefaultColorPalette(),
			CharacterSet: reporter.DefaultCharacterSet(),
			UseColor:     !ansi.DisableColor,
		},
		Logger: slog.New(slog.DiscardHandler),
	}
}

// Cwd is [os.Getwd], but returns a [*FilesystemError] if it fails.
func Cwd() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", &FilesystemError{"determine", "working directory", err}
	}
	return cwd, nil
}

// AddInputs adds the given inputs to a new [Options] inside c. Each input's
// klar.build is left empty.
func (c *Compiler) AddInputs(inputs ...Input) {
	c.Options = append(c.Options, &Options{Inputs: inputs})
}

func CompileString(s, fileName string) (c *Compiler, res *Result, err error) {
	cwd, err := Cwd()
	if err != nil {
		return nil, nil, err
	}
	c = NewCompiler(ModeBuild, cwd)
	c.Parser = NewStaticParser(fileName, &StaticParserFile{Reader: strings.NewReader(s)})
	c.AddInputs(Input{Kind: KindFile, Name: fileName, Path: fileName})

	c.StartTime = time.Now()
	res, err = c.Compile()
	return
}
