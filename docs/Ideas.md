# Klar Ideas & Plans

Things we may or may not implement. Comments and new ideas are greatly appreciated in [GitHub discussions](https://github.com/ProCode-Software/klar/discussions).

## CLI

- When `klar build ./samples/basic` is run (as a folder), it shows an error that Klar files aren't allowed in the root. Show a different error such as:
    - "No manifest found"
    - "Run individual files instead"

## Analysis

- Possible builtin functions:
    - `unreachable`
    - `typeOf`??
    - `typeNameOf`??
    - `sleep`??
- Ensure types can't be named after primitives

## Project Management

- A way to define a target platform per module
- `moduleOverrides` property in `glas.pack` files. This can be set to a map of module paths/patterns, and `target` and `deprecated` can be set for each.

## Build

- Dev server for testing JS projects
- `klar.js` or `js` module that provides target-specific type definitions.
- Similar: `klar.build` for build constants so `when build.isJavaScript` or `build.target`
- File extensions to include in compilation:
    - Code: .klar, .js, .ts,
    - Data: .json, .klon, .html, .xml, .txt, .csv, .tsv, .y(a)ml
    - Media: .css, .scss, .less, .png, .jp(e)g, .svg
