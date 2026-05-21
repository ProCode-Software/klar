package klarbuild

import (
	"github.com/ProCode-Software/klar/internal/config"
	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/pkg/klon"
)

func Parse(path string) (conf *File, err error) {
	conf = &File{Configuration: Configuration{}}
	if err = config.ReadFromFile(path, &conf, Context); err != nil {
		return conf, err
	}
	return conf, nil
}

// Default returns the default build configuration.
func Default() *File { return &File{Configuration: Configuration{}} }

var Context = &klon.Context{
	Enums: map[string]map[string]any{
		"ExhaustivenessOption": {
			"none":         NoExhaustiveness,
			"all":          AllExhaustiveness,
			"exceptResult": AllExhaustivenessExceptResult,
			"enumsOnly":    EnumExhaustiveness,
		},
		"CheckedAssertionOption": {
			"true":         AllowAssertions,
			"false":        DisallowAssertions,
			"withComments": AllowAssertionsWithComments,
		},
		"BundleMode": {
			"off":       BundleOff,
			"source":    BundleSource,
			"perModule": BundlePerModule,
			"all":       BundleStd,
		},
		"GlobalType": {
			"object":   GlobalObject,
			"string":   GlobalString,
			"number":   GlobalNumber,
			"function": GlobalFunction,
			"Array":    GlobalArray,
			"boolean":  GlobalBoolean,
			"Error":    GlobalError,
			"null":     GlobalNull,
			"const":    GlobalConst,
		},
		"SourceMapMode": {
			"false":  SourceMapDisabled,
			"true":   SourceMapEnabled,
			"inline": SourceMapInline,
		},
		"Target": {
			"js":      target.JavaScript,
			"klarvm":  target.KlarVM,
			"browser": target.Browser,
			"node":    target.Node,
			"deno":    target.Deno,
			"bun":     target.Bun,
		},
	},
}
