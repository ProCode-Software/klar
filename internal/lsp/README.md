# Klar Language Server (KlarLS)

KlarLS is the official implementation of Microsoft's [Language Server Protocol](https://microsoft.github.io/language-server-protocol/) for **Klar and Klon files**. It powers the features such as completion, diagnostics, hover, and formatting for these files in LSP-compatible editors such as VSCode, Zed, and Neovim.

KlarLS can be started by running the `klar lsp` command on the normal Klar binary. Advantages with a first-party language server built into the compiler CLI include:

- Less dependencies and disk space to get started with Klar development. Normally when you download a language extension in VSCode or Zed, the extension downloads a separate binary for the actual LSP. While including an LSP with the compiler increases the size of the binary, it is justified because:
    - Having a separate compiler binary duplicates the core of the Klar compiler written in Go, using even more space. I saw this with the Glas binary before making it a symlink of the Klar CLI and eliminating a second program.
    - Therefore the size increase of the Klar CLI will be a penalty for users that don't use the LSP. However, I expect the vast majority of Klar users to use the LSP in their editors. Most people who wouldn't use the LSP are highly experienced users (remember that beginners are Klar's target audience).
- Compile cache is also used for the LSP, so **project compilation outside the editor will also be faster by using the LSP**!

## Installation

- **VSCode:** See the [klar-vscode](../../klar-vscode) extension
- **Zed:** See [packages/klar-zed](../../packages/klar-zed)
- **Other editors:** Set the `klar lsp` command for `.klar` and `.klon` files.

The `klar-zed` and `klar-vscode` extensions also provide syntax highlighting for Klar, Klon, and `glas.lock` files.

## Features

These are the _planned_ early-phase features for KlarLS, and also Klar editor extensions.
For the technical capabilities of KlarLS, see the `getCapabilities` method in [server.go](./server.go).

### Klar files

- [ ] Diagnostics for compile errors and lints
- [ ] Hover
- [ ] Completion
- [ ] Signature help
- [ ] Formatting
- [ ] Refactoring / Code actions
- [ ] Rename
- [ ] Go to / Find all references
- [ ] Editor commands for running scripts (with support for passing CLI arguments)
- [ ] Semantic tokens

### Klon files

- [ ] Diagnostics for syntax errors, type errors, and lints
- [ ] Hover
- [ ] Schema validation for schemas defined as [JSON Schemas](https://json-schema.org/) or Klar files
- [ ] Schema-based completion
- [ ] Formatting
- [ ] Symbols (variables and object keys)
- [ ] Rename (variables)
- [ ] Go to / Find all references for variables
- [ ] Refactoring / Code actions (such as converting object/list syntax)

### Glas

KlarLS will also have Glas package management in the editor. These features include:

- [ ] Code actions/lenses for updating dependencies and other common Glas commands
- [ ] Hover info for dependency versions in the manifest
- [ ] Semantic highlighting for dependency specifiers in `glas.pack` files

### 🔮 Future / Phase 2

Features I would like for KlarLS to support, but aren't a priority. These aren't required for Klar 1.0.

- Debugging (as a [DAP](https://microsoft.github.io/debug-adapter-protocol/) implementation)
- Folding in `glas.lock` files (already supported in Zed because it uses Tree-sitter for the info instead of the LSP)

### 🚫 Not Planned / Distant Future

Features that are not a priority for a stable Klar release, or are out-of-scope for the Klar language. Some of these are defined in the LSP spec, but I don't even have the knowledge about in my editor.

- Moniker
- Inline completion
- Type hierarchy (inheritance isn't a focus in Klar's type system)
- Hover info in `glas.lock` files

## Public Protocol Package

The [`pkg/lsp`](../../pkg/lsp) package provides type definitions for the objects defined in the LSP spec. The type definitions are generated via a script that fetches the official [meta model](https://github.com/microsoft/language-server-protocol/blob/gh-pages/_specifications/lsp/3.18/metaModel/metaModel.json). Additionally, `pkg/lsp/rpc` provides definitions and functions for JSON RPC communication (message types, encoding/decoding).

## Compatibility

KlarLS is implemented for **LSP 3.18**. The original implementation was based on modern features that may not be fully supported in older LSP clients. These include the use of pull diagnostics (`textDocument/diagnostic`) with no support for push diagnostics (`textDocument/publishDiagnostics`). Legacy compatibility for spec versions before 3.18 is not a goal.

Currently, KlarLS only supports JSON RPC communication over stdio. It may support WebSockets or remote in the future, mostly for the purpose of running KlarLS in browsers/Wasm.

## Architecture

## Future Work

- Make a mock LSP client for testing KlarLS
- Port KlarLS to the browser/WebAssembly for use in the Klar Playground
