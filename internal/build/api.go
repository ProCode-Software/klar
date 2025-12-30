package build

import (
	"log/slog"
	"os"

	"github.com/ProCode-Software/klar/internal/config/klarbuild"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/errors/printer"
)

func NewCompiler(mode BuildMode) *Compiler {
	return &Compiler{
		Mode:         mode,
		errorPrinter: &printer.Printer{MaxLines: 3, Color: true},
		Logger:       slog.New(slog.DiscardHandler),
	}
}

// UseStdOpener sets c's Opener to the standard opener. UseStdOpener
// returns an error if it fails to get the working directory.
func (c *Compiler) UseStdOpener() error {
	cwd, err := os.Getwd()
	if err != nil {
		return &FilesystemError{"get", "working directory", err}
	}
	c.Opener = StdOpener{cwd: cwd}
	return nil
}

// PrintError prints an error to the error printer.
func (c *Compiler) PrintError(err errors.CompileError) {
	c.errorPrinter.PrintError(err)
}

// AddInputs adds the given inputs to a new [Options] inside c.
func (c *Compiler) AddInputs(inputs ...Input) {
	c.Options = append(c.Options, &Options{Inputs: inputs})
}

// ReadKlarBuild reads a 'klar.build' file at path and returns the configurations
// defined in it. No [Options] will contain more than one configuration. An error
// is returned if the file cannot be read or parsed.
func ReadKlarBuild(path string) ([]*Options, error) {
	f, err := klarbuild.Parse(path)
	if err != nil {
		return nil, err
	}
	if len(f.Configurations) > 0 {
		opts := make([]*Options, len(f.Configurations))
		for i, cfg := range f.Configurations {
			opts[i] = ParseKlarBuild(f, cfg)
		}
	}
	return []*Options{ParseKlarBuild(f, nil)}, nil
}

// ParseKlarBuild converts f and c into an [Options] object by converting
// c's inputs into [Input]. If c != nil, f and c are merged into a single
// configuration.
func ParseKlarBuild(f *klarbuild.File, c *klarbuild.Configuration) *Options {
	// TODO: input resolution
	if c == nil {
		_ = f.Configuration
		return &Options{
			File: *f,
			Inputs: nil,
		} 
	}
	newFile := *f
	newFile.Configurations = nil
	newFile.Configuration = *c
	return &Options{
		File: newFile,
		Inputs: nil,
	}
}
