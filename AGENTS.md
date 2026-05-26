# Instructions for AI Agents

This file contains rules for AI agents working with this codebase.

## Development Conventions

- Always use the latest Go syntax and features (from the Go version defined in `go.mod`). This includes:
    - `range` loops
    - `strings.SplitSeq`
    - `WaitGroup.Go`
    - `new(expr)`
    - `reflect.Type/Value.Fields/Methods`

You may research online about new language and stdlib features.

- **Do not create or run tests** unless explicitly asked, or you modify code that requires testing
- **Do not generate, remove, or modify blank lines and comments** except when code really needs explanation, or requested
- Don't modify or remove my comments unless they're irrelevant
- Group variables of 3+ with `var ( ... )` in Go
- Use `(*testing.B).Loop()` instead of `b.N` in benchmarks
- Use `min(...)` and `max(...)` when appropriate
- Use `cmp.Or()` when appropriate and possible

## Code Style

- **Line length**: Up to 90 columns for code; up to 94 is acceptable if parentheses exceed
- Doc comments: Target 90 columns, up to 95 for single-line comments
- **Formatter**: `gofumpt` (stricter than `gofmt`)
- Look at my TypeScript/JavaScript or Go codebase (depending on what you're working on) to see how I:
    - Separate logic using functions and spacing
    - Name objects
    - Write comments (No periods unless the sentence is long or more than one sentence)
    - Decide where to break lines between functions and composites
- When writing new functions, or when I ask to reorganise code, all types go at the top of the file, then functions and methods are ordered by when and the level at which they're called.

## Build & Development

- **Run `./run` to run the Klar CLI**
- Run `make build` to compile the main Klar binary
- Run `make gen` before submitting if AST types changed, or if objects/types that need to be generated have changed.
- Run `./scripts/lint.sh` to validate code quality
- Run `./scripts/format.sh` before submitting changes
- Always run Go tools with `GOEXPERIMENT=jsonv2` when they accept it, unless I explicitly tell you not to. This project uses the experimental `encoding/json/v2` package, which is faster and more configurable than the standard `encoding/json` package. Note that the `./run` script automatically sets `GOEXPERIMENT=jsonv2` when running the CLI.
- Never use `git restore` to restore your own changes, unless you stage yourself before you make changes! Don't undo MY changes!

## Architecture

Klar is a compiler for a modern programming language:

1. **Lexer** → tokenizes source code
2. **Parser** → builds AST from tokens
3. **Analysis** → type checking and semantic validation
4. **Code generation** → outputs JavaScript, and other languages in the future
5. **Runtime** → In the distant future

The file extension is `.klar` only.

Klar projects follow a specific directory structure specified in [docs/Project Structure](docs/ProjectStructure.md), and implemented in [internal/module](internal/module) and [internal/build/resolve.go](internal/build/resolve.go) (inexhaustive list).

See the [samples](samples/) directory for example projects and scripts. [samples/basic/all.klar](samples/basic/all.klar) is a Klar script that demonstrates all syntax features in the language.

### Klon
Klon is a markup language that will be Klar's flagship configuration format. `glas.pack`, `klar.build`, `klar.config`, and `*.klon` files are written in Klon.

[samples/basic/all.klon](samples/basic/all.klon) is a Klon file that demonstrates all syntax features in the language.

> Klon IS NOT YAML. For example, consecutive hyphens (`-`) at the beginning of lines indicate indentation and depth in Klon. Leading spaces are solely for formatting.

## Key Packages

- `cmd/klar/` — Main CLI
- `cmd/glas/` - Glas CLI
- `internal/*` — Core compiler
- `pkg/*` — Public APIs
- `std/` — Standard library written in Klar
- `klar-vscode/` VSCode extension for Klar

## Important Notes

- The project is in active development; some CLI commands are placeholders
- Do not manually edit generated files (`*_string.go`, `equal_nodes.go`, `walk_nodes.go`)
- Package manager (`glas`) is under development
- **DO NOT DO ANYTHING I DON'T EXPLICITLY TELL YOU TO DO**
- If you wrote code to files, I likely renamed some of the variables. Respect my naming conventions, and DO NOT change them back.
- Be aware that I write code too.
- Do what I ask. Don't do anything else.
- Ask me questions before starting or continuing. I want good code, so ask me to clarify or further explain.
- NEVER GUESS. Ask me. You may look at other examples in the codebase for suggestions that may be part of your question.
- If possible, call a tool to prompt the user with questions.