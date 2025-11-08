Klar is a programming language compiler and toolchain. The name means "clear" in Danish.

The project consists of:
- A compiler written in Go that parses and analyzes Klar source code (.klar files)
- A CLI tool (`klar`) for running scripts, REPL, and managing projects
- A VSCode extension for syntax highlighting and language support
- A configuration language called KlarML for project manifests

Klar is a statically-typed language with features like:
- Type inference and optional types
- Pattern matching with `when` expressions
- Enums, structs, and interfaces
- Generics and type unions
- String interpolation and multi-line strings
- Async/await with `go` and `await` keywords
- Module system with dot-separated imports
- Pipeline operators for functional composition

## Klar Project Structure

Klar projects use `glas.pack` manifest files and organize code in specific directories:
- `src/` - Main source code and modules
- `cmd/` - Executable commands
- `pkg/` - Reusable packages
- `shared/` - Shared modules within project
- `external/` - Foreign language code for FFI
- `.klar/` - Project cache and dependencies (gitignored)