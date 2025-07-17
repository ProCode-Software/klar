# TODO
## Lexer
- maybe: ..< operator
## Parser
- map, tuple, and list destructure assignment
- also in for loop
- in when case, map destructuring
- multi-variable assignments
- error for duplicate struct fields or enum members
- Error handling similar to Rust's `?` postfix, but with explicit `return`

## Analysis
- behaviour of () and (x) tuples
    - 1 item: not a tuple, just the item
    - 0 items: Nothing type
- .name and .value fields on enum members
- handle `<self>` and recursive types in structs
- handle recursive enum
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
