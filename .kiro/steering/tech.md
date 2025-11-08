## Tech Stack

- Language: Go 1.25.0
- Build: Standard Go toolchain with Makefile
- Package manager: Go modules
- VSCode extension: TypeScript/JavaScript with Bun

Make use of new Go 1.25 features, such as `strings.SplitSeq`, `any` instead of `interface{}`, generics, and `range` syntax.

Make use of multithreading for the Klar type checker.

## Key Dependencies

- `github.com/sanity-io/litter` - Pretty printing
- `github.com/ergochat/readline` - REPL line editing
- `golang.org/x/tools` - Go tooling support

## Common Commands

Build the compiler:
```bash
make build
# or
go build -o klar ./cmd/klar
```

Format code:
```bash
./scripts/format.sh
```

Generate code (stringer, etc):
```bash
./scripts/generate.sh
```

## Code Generation

The project uses `go generate` with `stringer` for enum string methods. Generated files follow the pattern `*_string.go`.

## VSCode Extension

Located in `klar-vscode/`. Uses Bun for package management and vsxtools for building.
