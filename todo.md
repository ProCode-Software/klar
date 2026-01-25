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
- String formats
- Regex literal formats
- Resource-using
- Should it be called `nil` or `none`?
- Should methods be allowed on interfaces?