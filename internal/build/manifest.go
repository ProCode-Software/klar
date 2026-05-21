package build

import "github.com/ProCode-Software/klar/internal/config/klarbuild"

// ReadKlarBuild reads a 'klar.build' file at path and returns the configurations
// defined in it. No [Options] will contain more than one configuration. An error
// is returned if the file cannot be read or parsed.
func ReadKlarBuild(path string) ([]*Options, error) {
	f, err := klarbuild.Parse(path)
	if err != nil {
		return nil, &InterfaceError{
			Code:  ErrInvalidConfig,
			Value: path,
			Err:   err,
		}
	}
	// Single configuration
	if len(f.Configurations) == 0 {
		cfg, err := MergeKlarBuild(f, nil)
		if err != nil {
			return nil, err
		}
		return []*Options{cfg}, nil
	}
	// Multiple configurations
	opts := make([]*Options, len(f.Configurations))
	for i, cfg := range f.Configurations {
		if opts[i], err = MergeKlarBuild(f, cfg); err != nil {
			return nil, err
		}
	}
	return opts, nil
}

// MergeKlarBuild converts f and c into an [Options] object by converting
// c's inputs into [Input]. If c != nil, f and c are merged into a single
// configuration. The resulting Configuration will have a single top-level config.
func MergeKlarBuild(f *klarbuild.File, c *klarbuild.Configuration) (*Options, error) {
	// Single top-level configuration
	if c == nil {
		c := f.Configuration
		inputs, err := ResolveInputs(c.Input)
		if err != nil {
			return nil, err
		}
		return &Options{File: *f, Inputs: inputs}, nil
	}
	// c is an individual config. Duplicate the file and replace the top-level config.
	f2 := *f
	f2.Configurations = nil
	f2.Configuration = *c
	inputs, err := ResolveInputs(f2.Input)
	if err != nil {
		return nil, err
	}
	return &Options{File: f2, Inputs: inputs}, nil
}

// DefaultKlarBuild returns an [Options] object with the default klar.build
// configuration. The configuration has no inputs.
func DefaultKlarBuild() *Options {
	def := klarbuild.Default()
	// There are no inputs in the default config for there to be an error
	opts, _ := MergeKlarBuild(def, nil)
	return opts
}
