# lsp/generate

This tool generates Go types for everything in the Language Server Protocol (LSP).

The LSP GitHub repo keeps a [`metaModel.json`](https://github.com/microsoft/language-server-protocol/blob/gh-pages/_specifications/lsp/3.18/metaModel/metaModel.json) file with all the LSP types. This is what we're using to generate. The schema is kept in [`metaModel.ts`](https://github.com/microsoft/language-server-protocol/blob/gh-pages/_specifications/lsp/3.18/metaModel/metaModel.ts), so the generator is based around the types it defines.

Additional notes:

- Nullable types are indirected as pointers, except for strings, numbers, and slices.
- When 2-/3-way unions are defined, the generator uses `rpc.Union2/3`. Unions with more types fall back to `any`.
- List syntax and `@link`s in doc comments are converted to Go doc syntax.
- Declarations are put into files based on the method request/notification that uses them as parameters.

## Before finishing

- Run `goimports` to remove the unused `rpc` imports (`goimports -w .`) and format the file
