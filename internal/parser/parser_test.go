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
	b.ReportAllocs()
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
	var cmtOld, cmtNew int
	var errsOld, errsNew int
	bench := func(name string, mode bool) {
		b.Run(name, func(b *testing.B) {
			for b.Loop() {
				var prog *ast.Program
			p := New(tokens, nil)
			if !mode {
				p.RemoveComments()
				p.InsertEOS()
				tokensOld = p.Tokens
				prog = p.Parse()
				errsOld = len(p.Errors)
				cmtOld = len(prog.Comments)
			} else {
				p.InsertEOSNew()
				tokensNew = p.Tokens
				prog = p.Parse()
				errsNew = len(p.Errors)
				cmtNew = len(prog.Comments)
			}
			_ = prog
			}
		})
	}
	bench("Old", false)
	bench("New", true)
	if cmtOld != cmtNew {
		b.Errorf("number of comments are not equal: old: %d, new: %d",
			cmtOld, cmtNew,
		)
	}
	if errsOld != errsNew {
		b.Errorf("number of errors are not equal: old: %d, new: %d",
			errsOld, errsNew,
		)
	}
	if len(tokensOld) != len(tokensNew) {
		b.Errorf("lengths of old and new EOS tokens are not equal: old: %d, new: %d",
			len(tokensOld), len(tokensNew),
		)
		for name, list := range map[string][]lexer.Token{
			"old": tokensOld,
			"new": tokensNew,
		} {
			file, err := os.Create("tokens_" + name + ".txt")
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
	arrow := (name, age) -> {}
	file := args[0]
	f := File(name: file, contents: fs.read(:file))
	print(f) // pretty print the file
}

run(os.args[1:])

import klar.http.*
import klar.http.server
import klar.json.{encode, decode, D: type Decodable}

users := [
    Person("John", age: 32),
    Person("Jane", age: 31),
    Person("Lucy", age: 28),
    Person("James", age: 34),
]

type Person {
    name: String
    age: Int
}

type DBRequest {
    amount: Int
    filter: (Person) -> Bool
}

server.handle(path: '/get', with: (r, w) -> {
    params: DBRequest? := nil
    r.params.decode(object: params)
    when params {
        ? -> w.error("Invalid params", code: .badRequest)
        _ -> {
            filtered := users.filter(params.filter)
            filtered = when params.amount {
                <= 0 -> filtered
                _ -> filtered[:params.amount]
            }
            w.returnJSON(filtered)
        }
    }
})

print("Starting the server...")
server.start(port: 8080)

for i in object {
}

type Person {
    name: String
    age: Int
    gender: Gender?
}

func Person.greet(otherPerson: String) {
    print("Hello {otherPerson}! My name is {self.name}.")
}

people: [Person] := [
    Person(name: "John", age: 34, gender: .male),
    Person(name: "Jane", age: 32, gender: .female),
]
print(people)
people[0].greet("Lucy")
`
