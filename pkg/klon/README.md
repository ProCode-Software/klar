# Klon

Klon (Klar Object Notation) is a configuration format (preferred over an _encoding_) dervied from Klar.

Klon features:

- Comments
- Variables
- Nesting based on hyphens rather than indentation (e.g. YAML)
- String interpolation / Object spreading

## JSON Compatibility

All JSON is valid Klon, **with the exception of `null` literals**. This wasn't the goal, I just happened to think similarly. This is thanks to Klon's inline lists (`[...]`) and objects (`{...}`), and quoted object keys. These are all optional syntax that you don't have to use, but they're supported.

Klon is designed to be a user-friendly configuration language, therefore it has different goals and use cases from JSON. For example, there is less determinism in Klon due to comments. Many `encoding/json/v2` APIs aren't planned for the `klon` package.

## Examples

See [../../samples/basic/all.klon](https://github.com/ProCode-Software/klar/tree/main/samples/basic/all.klon) for an example of all Klon syntax.

## Future Work

- A formal specification. It may be released before or around the same time as Klar's.
- A query language for making modifications to Klon ASTs
    ```go
    _ = klon.MustQuery(manifest, `dependencies += [@npm '@proicons/react' v4.x]`)
    klon.WriteASTTo(manifest, "glas.pack")
    ```
- Specialised `time.Time` serialization
- A builtin `@import` class to import fields and variables from other Klon files
    ```klon
    compilerOptions:
        - <- @import ../tsconfig.base compilerOptions
        - lib: [ESNext, DOM]
        - moduleResolution: NodeNext
        - declarationDir: @import ../tsconfig.shared '$DECLARATION_DIR'
    ```
- Support shebangs at the top of the file. There will probably be some programs that interpret Klon; the feasability is higher for Klon than JSON.

## Architecture
