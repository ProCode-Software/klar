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

### Representation

For research purposes, some possible outputs are available in the [jstest](./jstest) folder. Please send feedback and suggest changes. We want Klar to produce the most efficient JavaScript possible.

The generated representation for Klar types may not pass TypeScript typechecking.

#### Structs

<table>
<tr><th>Klar</th> <th>JavaScript</th></tr>
<tr>
<th>

```klar
type Person {
    name: String
    age: Int
    hobbies: [String] = []
}
```

</th>
<th>

```js
class Person {
    name
    age
    hobbies = []

    constructor(name, age, hobbies = []) {
        this.name = name
        this.age = age
        this.hobbies = hobbies
    }
}
```

</th>
<tr>
</table>

#### Tags

Tags are JavaScript `Symbol`s. Structs and enums that implement tags have the tag's symbol as a property.

<table>
<tr><th>Klar</th> <th>JavaScript</th></tr>
<tr>
<th>

```klar
public type #Expression

public type BinaryExpression: Expression {}
public type VariableDeclaration {}
```

</th>
<th>

```js
// The string argument to Symbol() is trivial, but is set for debugging purposes
export const Expression = Symbol('Expression')

export class BinaryExpression {
    // Native JS doesn't support computed properties directly inside the class
    // (like below), so Tag has to be set in the constructor or in a dot-index.
    //   [Expression] = null
    constructor() {}
}
// `null` indicates that the type implements the tag, while `undefined`
// means it doesn't.
BinaryExpression.prototype[Expression] = null

export class VariableDeclaration {
    constructor() {}
}
```

</th>
<tr>
</table>

To check if a class implements an interface from JavaScript:

```js
const binaryExpr = new BinaryExpression()
const varDecl = new VariableDeclaration()
// Must be compared strictly (===), because nonstrict `null == undefined` in JS
console.log(binaryExpr[Tag] === null, varDecl[Tag] === null)
// Output: true (implements Tag), false
```

#### Overloads and Custom Initializers

#### Enums

See [jstest/enum.js](./jstest/enum.js)

#### Interface Checking

## JavaScript IR to .js ([`jswriter`](./jswriter) package)

The `jswriter` package knows nothing about the original Klar code.

### TypeScript Declarations
