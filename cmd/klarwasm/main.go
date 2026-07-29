//go:build js && wasm

// Klarwasm provides a WASM-compatible interface for the Klar compiler.
package main

import (
	"encoding/json/v2"
	"strings"
	"syscall/js"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
	"github.com/ProCode-Software/klar/internal/klarerrs/jsonerrors"
)

func main() {
	var compile, freeCompiler js.Func
	compile = js.FuncOf(func(this js.Value, args []js.Value) any {
		s, report := Parse(args[0].String(), args[1].String())
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

// Parse compiles the given source string and returns the result as a JSON string.
func Parse(s string, fileName string) (data string, report func() string) {
	var (
		buf         strings.Builder
		c, res, err = build.CompileString(s, fileName)
		isMaxErrors = build.IsMaxErrors(err)
	)
	if err != nil || len(res.Errors) > 0 {
		jsonerrors.WriteTo(&buf, res, err, isMaxErrors)
	} else {
		json.MarshalWrite(&buf, res)
	}
	return buf.String(), func() string {
		return ReportErrors(c, res, err, isMaxErrors)
	}
}

func ReportErrors(c *build.ProjectCompiler, res *build.Result, err error, isMaxErrors bool) string {
	var buf strings.Builder
	c.Reporter.UseColor = true
	c.Reporter.Output = &buf

	// Actual errors
	c.PrintAllErrors(res.Errors)

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
