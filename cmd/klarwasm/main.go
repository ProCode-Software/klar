//go:build js && wasm

// Klarwasm provides a WASM-compatible interface for the Klar compiler.
package main

import (
	"encoding/json/v2"
	"strings"
	"syscall/js"
	"time"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/klarerrs/jsonerrors"
	"github.com/ProCode-Software/klar/internal/target"
)

func main() {
	ansi.DisableColor = true
	pc := makeCompiler()
	var compile, freeCompiler js.Func
	compile = js.FuncOf(func(this js.Value, args []js.Value) any {
		s, report := Compile(pc, args[0].String(), args[1].String())
		var getErrors js.Func
		getErrors = js.FuncOf(func(this js.Value, args []js.Value) any {
			getErrors.Release()
			return report()
		})
		return js.ValueOf(map[string]any{
			"output":    s,
			"getErrors": getErrors, // Returns CLI-style diagnostics as a string
		})
	})
	// Function to free the compiler when the page is unloaded
	freeCompiler = js.FuncOf(func(js.Value, []js.Value) any {
		compile.Release()
		freeCompiler.Release()
		return nil
	})

	js.Global().Set("Klar", js.ValueOf(map[string]any{
		"compile":      compile,
		"freeCompiler": freeCompiler,
	}))

	select {} // Keep running
}

func makeCompiler() *build.ProjectCompiler {
	pc := build.NewProjectCompiler(build.NewCompiler(build.ModeBuild, ""))
	pc.Parser = build.NewStaticParser(pc.FS, "", "", nil)
	return pc
}

// Compile compiles the given source string and returns the result as a JSON string.
// The returned function report can be called to return the CLI-style diagnostics.
func Compile(
	pc *build.ProjectCompiler, s, fileName string,
) (data string, report func() string) {
	// Load the file into the compiler's FS
	sp := pc.Parser.(*build.StaticParser)
	sp.LoadFile(fileName, &build.StaticParserFile{Reader: strings.NewReader(s)})

	pc.Inputs = append(pc.Inputs, &build.Input{
		Kind:    build.KindFile,
		Path:    fileName,
		Targets: []target.Target{target.JavaScript},
	})

	pc.StartTime = time.Now()
	var resultJSONBuf strings.Builder
	res, err := pc.Compile()
	if err != nil || len(res.Errors) > 0 {
		err = jsonerrors.WriteTo(&resultJSONBuf, res, err)
	} else {
		err = json.MarshalWrite(&resultJSONBuf, res)
	}
	// Error while writing JSON
	if err != nil {
		// TODO: Maybe we should return an error to JS instead
		panic(err)
	}
	return resultJSONBuf.String(), func() string {
		return ReportErrors(pc, res, err)
	}
}

func ReportErrors(pc *build.ProjectCompiler, res *build.Result, err error) string {
	var buf strings.Builder
	pc.Reporter.UseColor = true
	pc.Reporter.Output = &buf

	// Actual errors
	pc.PrintAllErrors(res.Errors)

	// Critical error
	if err != nil {
		const prefix = "<**><r!>Error</r!><dim>:</dim> "
		if ie, ok := err.(*build.InterfaceError); ok {
			main, det := ie.PrettyError()
			ansi.TagFprintfln(&buf, prefix+"%s</**>%s", main, det)
		} else {
			ansi.TagFprintfln(&buf, prefix+"%s</**>", err)
		}
	}
	return buf.String()
}
