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
	cli.Parse()
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
	wg.Wait()
	fmt.Println("Build complete!")
}

func buildFile(file *os.File) {
	lexer := lxr.NewLexer(file)

	// Recover if the lexer panics
	defer func() {
		if err := recover(); err != nil {
			BuildError(fmt.Errorf("%v", err))
		}
	}()
	for {
		token := lexer.Parse()
		if token.Kind == lxr.EOF { // EOF
			break
		}
		var pre, post string
		if token.Kind == lxr.Illegal {
			pre, post = "\033[31m", "\033[m"
		}
		if token.Attributes != nil {
			fmt.Printf(
				pre+"%-20s %-8q %v %+v\n"+post,
				lxr.TokenTypes[token.Kind],
				token.Source, token.Position,
				token.Attributes,
			)
		} else {
			fmt.Printf(
				pre+"%-20s %-8q %v\n"+post,
				lxr.TokenTypes[token.Kind],
				token.Source, token.Position,
			)
		}
	}
}
