package parser

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ProCode-Software/klar/internal/ast"
	"github.com/ProCode-Software/klar/internal/lexer"
)

func BenchmarkParser(b *testing.B) {
	lex := lexer.NewLexer(strings.NewReader(testDocument))
	var tokens []lexer.Token
	for {
		tok := lex.Tokenize()
		tokens = append(tokens, *tok)
		if tok.Kind == lexer.EOF {
			break
		}
	}
	var tokensOld, tokensNew []lexer.Token
	bench := func(name string, mode bool) {
		b.Run(name, func(b *testing.B) {
			var prog *ast.Program
			p := New(tokens, nil)
			if !mode {
				prog = p.Parse()
				tokensOld = p.Tokens
			} else {
				prog = p.ParseNew()
				tokensNew = p.Tokens
			}
			_ = prog
		})
	}
	bench("Old", false)
	bench("New", true)
	if len(tokensOld) != len(tokensNew) {
		b.Errorf("lengths of old and new EOS tokens are not equal: old: %d, new: %d",
			len(tokensOld), len(tokensNew),
		)
		for name, list := range map[string][]lexer.Token{
			"old": tokensOld,
			"new": tokensNew,
		} {
			file, err := os.Create("tokens_"+name+".txt")
			if err != nil {
				panic(err)
			}
			defer file.Close()
			for _, tok := range list {
				fmt.Fprintf(file, "%-20s %-5s %#q\n", tok.Kind, tok.Position, tok.Source)
			}
		}
	}
}

const testDocument = `#!/usr/bin/env klar
import klar.os
import klar.fs

type File {
	name: String
	contents: String
}

// main function
func run(args: [String]) {
	when args.length {
		0 -> {
			// file not provided
			print("Missing file", to: .standardError)
			os.exit(1)
		}
	}
	file := args[0]
	f := File(name: file, contents: fs.read(:file))
	print(f) // pretty print the file
}

run(os.args[1:])`
