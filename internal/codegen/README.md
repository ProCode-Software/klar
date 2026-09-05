# Klar-to-JS Code Generation

This package handles the conversions of:

- Klar AST + type info -> JavaScript IR
- JavaScript IR -> source code

This package is used by the compiler to convert Klar ASTs to JavaScript, and also to generate TypeScript .dts declarations.

## Klar to JavaScript IR

Important things to consider:

1. Class methods have to be in a single file, in a JavaScript `class` block.
2. Klar modules are multiple files in a directory. JavaScript modules are individual files. When converting Klar module structure to JavaScript, there must be no cycles. We still want to preserve _most_ of the original Klar file structure when bundling is disabled.

The JS IR is low-level, so bundling is handled during this step. Note that the compiler performs dead code elimination before handing the AST to the `codegen` package. Dead declarations are stored in a map that `codegen` reads from to avoid generation.

### What `codegen` doesn't do

- **Dead code elimination:** That is performed by the optimizer before the AST is handed to `codegen`
- **Inlining:** Also performed by the optimizer
- **Asset loading:** The content of external values (such as `@external` JSON) is sent to `codegen` as IR. For example, for external JSON, the JSON is parsed and given to `codegen` as JavaScript IR to include as the value.

### Steps

1. **Convert individual Klar files to JS IR:** Each Klar AST is converted to JavaScript IR (a `jsir.Module`). This step is unaware of bundling settings and module cycles. Cycles and method grouping will be handled in a later step. This step just handles syntax conversion.
2. **Optimization:** This includes simplifying logical conditions in `if` chains. Klar-level optimizations, such as constant folding, are handled before codegen.
3. **Module reorganization:** Methods are moved to common files where the type is declared. Top-level declarations may be moved to a single file so two files don't reference each other.
4. **Bundling:** Based on build settings, bundling is performed by combining multiple modules into a single file. Name mangling may be performed.

### Organizing Files

## JavaScript IR to .js ([`jswriter`](./jswriter) package)

The `jswriter` package knows nothing about the original Klar code.

### TypeScript Declarations
