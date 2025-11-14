package main

import (
	"fmt"
	"os"
)

const template = `
func (err *%[1]s) GetMessage() string         { return err.Message }
func (err *%[1]s) GetRange() ranges.Range     { return err.Range }
func (err *%[1]s) GetCode() ErrorCode         { return err.ErrorCode }
func (err *%[1]s) GetHints() []Hint           { return err.Hints }
func (err *%[1]s) GetFile() string            { return err.File }
func (err *%[1]s) GetDetails() []Detail       { return err.Details }
func (err *%[1]s) GetHighlights() []Highlight { return err.Highlights }
func (err *%[1]s) Hint(hint Hint)             { err.Hints = append(err.Hints, hint) }
func (err *%[1]s) Hintf(hint string, a ...any) {
	err.Hints = append(err.Hints, fmt.Sprintf(hint, a...))
}`

func main() {
	outputFile := os.Args[1]
	errorTypes := os.Args[2:]

	f, err := os.Create(outputFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	fmt.Fprintln(f, "package errors")
	fmt.Fprintln(f, `import (
	"fmt"
	"github.com/ProCode-Software/klar/internal/ranges"` + "\n)")
	for _, typ := range errorTypes {
		fmt.Fprintf(f, template, typ)
	}
}
