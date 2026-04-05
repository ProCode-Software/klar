package main

import (
	"encoding/json/v2"
	"strings"
	"syscall/js"
	"time"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/errors"
	"github.com/ProCode-Software/klar/internal/errors/jsonerrors"
	"github.com/ProCode-Software/klar/pkg/parser"
)

func main() {
	js.Global().Set("compileKlar", js.FuncOf(func(this js.Value, args []js.Value) any {
		return Parse(args[0].String())
	}))
	select {}
}

func Parse(s string) string {
	prog, parseErrs, err := parser.ParseString(s)
	var buf strings.Builder
	compileErrs := make([]errors.CompileError, len(parseErrs))
	for i, e := range parseErrs {
		compileErrs[i] = e
	}
	if err == nil && len(parseErrs) == 0 {
		if err := json.MarshalWrite(&buf, prog); err != nil {
			panic(err)
		}
		return buf.String()
	}
	jsonerrors.WriteTo(&buf, &build.BuildResult{
		Errors:  compileErrs,
		Modules: nil,
		Elapsed: time.Duration(0),
	}, err, false)
	return buf.String()
}
