# TODO
## Lexer
- maybe: ..< operator
- fix number with trailing decimal point
## Parser
- map, tuple, and list destructure assignment
- also in for loop
- new enum syntax
- in when case, map destructuring
- multi-variable assignments
- error for duplicate struct fields or enum members
- Error handling similar to Rust's `?` postfix, but with explicit `return`
- _ in lambda param: `_ -> ...`, `(_) -> ...`, `(_: Int) -> ...`

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

## Other
- target property in manifest for defining target platform per module
- change lexer.Position type to be either uint64 combo of line * 32 + col * 32 bytes,
    or use uint32 values in struct


## Build Ideas
- Dev server for testing JS projects
- Packages that just import globals like `import node`
- Similar: `klar.build` for build constants so `when build.isJavaScript` or `build.target`
File extensions to include in compilation:
- Code: .klar, .js, .ts,
- Data: .json, .klarml, .html, .xml, .txt, .csv, .tsv, .y(a)ml
- Media: .css, .scss, .less, .png, .jpg, .jpg, .svg
option to convert klarml to json


# Proposals
- `shadow` keyword:
```klar
x: Int := 1
shadow x: String := "foo"
```
- Cascade
```klar
vehicle := Vehicle()
    |> .name = "Toyota"
    |> .range = 3000
    |> .drive(10)
```
.[] for computed fields