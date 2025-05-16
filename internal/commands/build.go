package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ProCode-Software/klar/internal/args"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func BuildError(err error) {
	fmt.Fprintf(os.Stderr, "\033[1;31m❌ Build failed\033[0;1;2m:\033[0;1m\n    %v\033[m\n", err)
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
	startTime := time.Now()
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
	endTime := time.Since(startTime)
	fmt.Printf(
		"\033[1;32m✅ Build succeeded in \033[36m%dms\033[32m!\033[m\n",
		endTime.Milliseconds(),
	)
}

func buildFile(file *os.File) {
	// Recover if panics
	defer func() {
		if err := recover(); err != nil {
			BuildError(fmt.Errorf("Internal Error: %v", err))
		}
	}()

	// ========================
	// LEXER
	// ========================
	lex := lexer.NewLexer(file)
	lex.IncludeComments = true

	// Estimate token capacity
	stat, err := file.Stat()
	if err != nil {
		panic(err)
	}
	byteSize := stat.Size()
	tokens := make([]lexer.Token, 0, byteSize/4)

	for {
		token := lex.Tokenize()
		tokens = append(tokens, *token)
		if token.Kind == lexer.String {
			fmt.Printf("%#v\n", token)
		}
		if token.Kind == lexer.EOF {
			break
		}
	}

	// ========================
	// PARSER
	// ========================
}
