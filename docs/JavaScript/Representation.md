# JavaScript Representation of Klar

For libraries, the compiler tries to produce JavaScript that can easily and cleanly be called from outside of Klar. Klar's goal is not to replace JavaScript or TypeScript, but to provide a better development experience while still supporting the large JavaScript ecosystem. Klar provides many features to use JavaScript libraries in Klar; and in order not to split the ecosystem, the case should be similar vice-versa.

For research purposes, some possible outputs are available in the [jstest](./jstest) folder. Please send feedback and suggest changes on GitHub. We want Klar to produce the most efficient JavaScript possible, by file size and runtime performance.

The generated representation for Klar types may not pass TypeScript typechecking. When using Klar and TypeScript in your project, it is recommended you configure TypeScript to avoid checking files generated from Klar.

**Some basic things to know when converting Klar language features to JS:**

- Labelled parameters are passed as positional in JavaScript
- Structs are compiled to ES6 classes

## Structs

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

## Tags

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

## Overloads and Custom Initializers

Functions with multiple overloads and custom initializers will be defined with one base function, with the original name. The function checks the types of the parameters to call another unexported/private function with a similar name, but numbered. The base function takes the same number of parameters as the Klar overload with the most. TypeScript declarations will still be defined by overloads.

<table>
<tr><th>Klar</th> <th>JavaScript</th></tr>
<tr>
<th>

```klar
public func replace(old: String, with new: String, in str: String) -> String {
    // #1
}
public func replace(old: text.Regex, with new: String, in str: String) -> String {
    // #2
}
```

</th>
<th>

```js
/**
 * @param {string | RegExp} old
 * @param {string} new_
 * @param {string} in_
 * @returns {string}
 */
export function replace(old, new_, in_) {
    if (old instanceof RegExp) return replace2(old, new_, in_)
    return replace1(old, new_, in_)
}

function replace1(old, new_, in_) {
    // #1
}
function replace2(old, new_, in_) {
    // #2
}
```

</th>
<tr>
</table>

Custom initializers are similar, but the base function would be `constructor`.

When the overloads define many uncommon parameters, those uncommon parameters will be accepted in an object literal instead of passing all of them as parameters.

<table>
<tr><th>Klar</th> <th>JavaScript</th></tr>
<tr>
<th>

```klar
public func String(num: Float) // #1
public func String(num: Float, includeZeroDecimal includeZero: Bool) // #2
public func String(num: Float, maxDecimalPlaces decimalPlaces: Int) // #3
public func String(num: Float, exactDecimalPlaces decimalPlaces: Int) // #4
public func String(num: Float, exponential exponential: Bool) // #5
public func String(
    num: Float, maxDecimalPlaces decimalPlaces: Int,
    exponential exponential: Bool,
) // #6
public func String(
    num: Float, exactDecimalPlaces decimalPlaces: Int,
    exponential exponential: Bool,
) // #7
```

</th>
<th>

```ts
class Float {
    toString(params?: {
        includeZeroDecimal: boolean
        maxDecimalPlaces: number
        exactDecimalPlaces: number
        exponential: boolean
    }): string {
        if (!params) return this.#toString1()
        const {
            includeZeroDecimal: includeZero,
            maxDecimalPlaces: decimalPlaces,
            exactDecimalPlaces: decimalPlaces2,
            exponential,
        } = params
        // The user may pass `null` or `undefined` for each option. `null == undefined` in JS.
        if (includeZero != null) return this.#toString2(includeZero)
        if (decimalPlaces != null)
            return exponential == null
                ? this.#toString3(decimalPlaces)
                : this.#toString6(decimalPlaces, exponential)
        if (decimalPlaces2 != null)
            return exponential == null
                ? this.#toString4(decimalPlaces2)
                : this.#toString7(decimalPlaces2, exponential)
        if (exponential != null) return this.#toString5(exponential)
        // Params is an empty object `{}`, but not null or undefined
        return this.#toString1()
    }
    #toString1(): string {} // #1
    #toString2(includeZero: boolean): string {} // #2
    #toString3(decimalPlaces: number): string {} // #3
    #toString4(decimalPlaces: number): string {} // #4
    #toString5(exponential: boolean): string {} // #5
    #toString6(decimalPlaces: number, exponential: boolean): string {} // #6
    #toString7(decimalPlaces: number, exponential: boolean): string {} // #7
}
```

</th>
<tr>
</table>

For more information on how initializers/casts for builtin types are represented in JS, see the [Custom Initializers for Builtins](#custom-initializers-for-builtins) section.

The user would call `toString` like:

```js
float.toString()
float.toString({}) // Same
float.toString({ maxDecimalPlaces: 2 })
float.toString({ exactDecimalPlaces: 2, exponential: true })
```

## Enums

Klar:

```klar
type TokenType {
    .leftParen
    .string(content: String)
}
```

For the JavaScript representation, see [jstest/enum.js](./jstest/enum.js)

## Custom Initializers for Builtins

## Interface Checking
