package klarbuild

import (
	"os"

	"github.com/ProCode-Software/klar/internal/target"
	"github.com/ProCode-Software/klar/pkg/klon"
)

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

func Parse(path string) (config *File, err error) {
	var fr *os.File
	fr, err = os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fr.Close()

	config = &File{Configuration: Configuration{}}
	if err = klon.UnmarshallReadContext(fr, &config, Context); err != nil {
		return nil, err
	}
	return config, nil
}

func Default() *File {
	return &File{Configuration: Configuration{}}
}
