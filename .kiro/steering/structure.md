## Project Organization

The repository follows Go conventions with additional structure for the language toolchain:

### `/cmd`
Entry points for executables:
- `cmd/klar/` - Main CLI tool with subcommands (run, repl, help)
- `cmd/glas/` - Package manager (not yet implemented)

### `/internal`
Core compiler implementation (not importable by external projects):
- `lexer/` - Tokenization and lexing
- `parser/` - Parsing tokens into AST
- `ast/` - Abstract syntax tree node definitions
- `analysis/` - Type checking and semantic analysis
- `errors/` - Error types and pretty printing
- `types/` - Type system implementation
- `build/` - Build system and module compilation
- `module/` - Module resolution and project management
- `cli/` - CLI utilities (ANSI colors, arg parsing, tab writer)
- `runtime/` - Runtime context and values
- `char/`, `ranges/`, `paths/` - Utility packages

### `/pkg`
Public APIs (importable by external projects):
- `pkg/klarml/` - KlarML parser and decoder for config files
- `pkg/parser/` - Public parsing interface
- `pkg/printer/` - Token printing utilities
- `pkg/analysis/` - Public type checking interface

### `/samples`
Example Klar code and test projects

### `/docs`
Project documentation including ProjectStructure.md

### `/klar-vscode`
VSCode extension with syntax definitions and language configuration

### `/scripts`
Development scripts for formatting, code generation, and utilities

## Conventions

- Go code uses tabs for indentation (per `.editorconfig`)
- Four spaces to indent other files
- Internal packages are not exposed outside the project
- AST nodes implement the `Node` interface with position tracking
- Errors use custom types in `internal/errors/` with pretty printing
- Parser uses Pratt parsing with binding powers
- Module paths use dot notation (e.g., `klar.http.requests`)
- Klar project structure follows specific conventions (see ProjectStructure.md)


