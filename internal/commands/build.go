package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ProCode-Software/klar/internal/args"
	lxr "github.com/ProCode-Software/klar/internal/lexer"
)

func BuildError(err error) {
	fmt.Fprintf(os.Stderr, "Build failed: %v", err)
	os.Exit(1)
}

func Build() {
	cli := args.ArgTable{}
	wg := sync.WaitGroup{}

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for _, input := range cli.Args {
		wg.Add(1)
		go func(input string) {
			defer wg.Done()
			file, err := os.Open(input)
			if os.IsNotExist(err) {
				file, err = os.Open(input + ".klar")
			}
			if os.IsNotExist(err) {
				BuildError(fmt.Errorf(
					"could not find file '%s'", filepath.Join(cwd, input),
				))
			}
			buildFile(file)
		}(input)
	}
}

func buildFile(file *os.File) {
	outputFile, _ := os.Create(file.Name() + "_build.txt")
	lexer := lxr.NewLexer(file)

	// Recover if the lexer panics
	defer func() {
		if err := recover(); err != nil {
			BuildError(fmt.Errorf("%v", err))
		}
	}()
	for {
		pos, token, src := lexer.Parse()
		if token == lxr.EOF { // EOF
			break
		}
		fmt.Fprintf(outputFile, "%")
	}
}
