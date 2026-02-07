# TODO
## Lexer

## Parser

## Analysis
- behaviour of () and (x) tuples
    - 1 item: not a tuple, just the item
    - 0 items: Nothing type
- .name and .value fields on enum members
- handle `<self>` and recursive types in structs
- handle recursive enum
- overhaul struct inheritance
- attributes in enum
- custom initializers
- only allow inheriting one struct??
- fix regex handling of // or \//
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
    - assert??
    - typeOf??
    - typeNameOf??
    - sleep??

## Other
- target property in manifest for defining target platform per module

## Build Ideas
- Dev server for testing JS projects
- Packages that just import globals like `import node`
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

# RFCs
1. String formats
2. Regex literal formats (`#/.../` vs `%/.../`, multi-`/`)
3. Resource-using (`use`, `using`)
4. Should it be called `nil` or `none`?
5. Should methods be allowed on interfaces?
6. Struct publicity for fields and methods. Should they be private by default? Or should private fields be prefixed with `_`?
7. How to distinguish variable values from unwrapped variable declarations in `when` pattern matching
8. Should the value be the first, or last parameter in each pipeline step? Or should steps explicitly use `value` if they take multiple parameters?
9. Private modules (`internal` folder, and/or `_` prefix?)
10. Bitmask enums
11. New `Iterator` type, or iterators are `List`s with compiler optimizations?
12. Should the compiler enforce identifier capitalization conventions? (`PascalCase` for types, `camelCase` for variables)
13. Disallow tuples with 0 or 1 items