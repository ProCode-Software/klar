# TODO
- When `klar build ./samples/basic` is run (as a folder), it shows an error that Klar files aren't allowed in the root. Show a different error such as:
    - No manifest found
    - Run individual files instead

- Error reporter: Don't draw arrows if there is no message. Underline only.
## Lexer

## Parser

## Analysis
- .name and .value fields on enum members
- recursive type aliases
- overhaul struct inheritance
- custom initializers
- only allow inheriting one struct??
- more pattern matching:
    ```klar
    when msg {
        3 + 5
        "Hello"... -> // when string begins with 'Hello'
    }
    ```
- builtins:
    - crashout
    - TODO
    - unreachable
    - typeOf??
    - typeNameOf??
    - sleep??

## Other
- target property in manifest for defining target platform per module
- `moduleOverrides` property in `glas.pack` files. This can be set to a map of module paths/patterns, and `target` and `deprecated` can be set for each.

## Build Ideas
- Dev server for testing JS projects
- `klar.js` or `js` module that provides target-specific type definitions.
- Similar: `klar.build` for build constants so `when build.isJavaScript` or `build.target`
File extensions to include in compilation:
- Code: .klar, .js, .ts,
- Data: .json, .klon, .html, .xml, .txt, .csv, .tsv, .y(a)ml
- Media: .css, .scss, .less, .png, .jp(e)g, .svg
option to convert klon to json

# Proposals
- `shadow` keyword:
```klar
x: Int := 1
shadow x: String := "foo"
```
