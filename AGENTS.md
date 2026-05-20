# Instructions for AI Agents

This file contains rules for AI agents working with this codebase.

## Development Conventions

- Always use the latest Go syntax and features (from the Go version defined in `go.mod`). This includes:
    - `range` loops
    - `strings.SplitSeq`
    - `WaitGroup.Go`
    - `new(expr)`
- **Do not create or run tests** unless explicitly asked
- **Do not generate, remove, or modify blank lines and comments** except when code really needs explanation (or requested)
- Group variables of 3+ with `var ( ... )` in Go
- Use `(*testing.B).Loop()` instead of `b.N` in benchmarks
- Use `min(...)` and `max(...)` when appropriate
- Use `cmp.Or()` when appropriate and possible

## Code Style

- **Line length**: Up to 90 columns for code; up to 94 is acceptable if parentheses exceed
- Doc comments: Target 90 columns, up to 95 for single-line comments
- **Formatter**: `gofumpt` (stricter than `gofmt`)

## Build & Development

- **Run `./run` to run the Klar CLI**
- Run `make build` to compile the main klar binary
- Run `make gen` before submitting if AST types changed
- Run `./scripts/lint.sh` to validate code quality
- Run `./scripts/format.sh` before submitting changes
- Always run Go tools with `GOEXPERIMENT=jsonv2` when they accept it, unless I explicitly tell you not to.
- Never use `git restore` to restore your own changes, unless you stage yourself before you make changes! Don't undo MY changes!

## Architecture

Klar is a compiler for a modern programming language:

1. **Lexer** → tokenizes source code
2. **Parser** → builds AST from tokens
3. **Analysis** → type checking and semantic validation
4. **Code generation** → outputs JavaScript

## Key Packages

- `cmd/klar/` — Main CLI
- `cmd/glas` - Glas CLI
- `internal/*` — Core compiler
- `pkg/analysis/`, `pkg/parser/` — Public APIs
- `std/` — Standard library written in Klar
- `klar-vscode/` VSCode extension for Klar

## Important Notes

- The project is in active development; some CLI commands are placeholders
- Do not manually edit generated files (`*_string.go`, `equal_nodes.go`, `walk_nodes.go`)
- Package manager (`glas`) is under development
- **DO NOT DO ANYTHING I DON'T EXPLICITLY TELL YOU TO DO**
