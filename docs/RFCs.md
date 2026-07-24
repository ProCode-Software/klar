# Klar Requests for Comments (RFCs)

For some language features, we want extra comments from developers before finalizing syntax and semantics.

Discussions are open on [GitHub](https://github.com/ProCode-Software/klar/discussions) for each RFC below. You may comment and react to other comments on those threads.

Where given, consideration options (listed alphabetically) aren't exhaustive. You may request other options, or comment about or propose changes to existing ideas.

## Open RFCs

1. **String literal formats**
    - Ideas for existing quote styles:
        - Double quotes `"` for multiline strings with interpolation
        - Single quotes `'` for single-line strings without interpolation
        - Backticks <code>`</code> for multiline strings with no interpolations or character escapes (raw strings)
    - New string formats (what quoting style to use?)
        - Strings that can be formatted multiline in source code, but are concatenated into a single line. ("folding strings")
        - Strings that allow escaping quotes??
2. **Regex literal formats**
    - Should regex literals be `#/.../` or `%/.../`?
    - How can regex literals be defined with multiple-`/`? (to allow including `/` without escaping)
3. **Resource management**
    - Implementations in other languages:
        - Python: `with` blocks
        - Gleam: [`use` declarations](https://tour.gleam.run/advanced-features/use/) that can be set to functions that take a callback.
        - JavaScript: [`using` declarations](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/using) that can be set to objects with a dispose method
        - Java: `try` blocks (Try-with-Resources)
    - Should the keyword be `use` or `using`?
4. **Should it be called `nil` or `none`?**
5. **Should methods be allowed on interfaces?** - These would allow interfaces to contain methods that use interface values as receivers/`self`.

    ```klar
    type #MyInterface { key: Int }
    func MyInterface.method() { print(self.key) }

    impl: MyInterface := someObject
    impl.method()
    ```

6. **Publicity for struct fields/methods**
    - a. Private by default
    - b. Private fields should be prefixed with `_`
    - c. Add a new `private` keyword. Downside: Another keyword

    For methods, which are declared in the same block as module-level functions:
    - a. should the `public` be used (for consistency with module-level functions), or
    - b. methods are public by default, and `_` should be used to make them private?

    ```klar
    public func Struct.method()
    func Struct._method()
    public func moduleFunc()
    ```

7. **How to distinguish variable values from unwrapped variable declarations in `when` pattern matching**

    Example:

    ```klar
    val := "foo"
    when value {
        // Is val declared as a variable set to the first field in MyStruct, or this checks whether the first field's value is "foo"?
        MyStruct(val) -> { ... }
        // Is val a type, or "foo"?
        val -> { ... }
    }
    ```

    Current approach: `val` as a direct case (2nd) will compare to a variable, and unwrapping `val` in `MyStruct` (1st) is a variable.

8. **Placement of pipeline values in function calls:** Unless the `value` variable is used in a step:
    - a. First parameter
    - b. Last parameter
    - c. Require the use of <code>value</code> for functions that take multiple parameters

    ```klar
    val := 'foo'
    func bar(x: String, y: Int) {}
    func reverse(str: String) -> String {}

    // Option A
    val |> reverse |> bar(2)

    // With Option C, this is required
    val |> reverse |> bar(value, 2)
    ```

9. **Private modules**
    - a. `internal` folder that can't be imported by packages outside the project
    - b. Folders prefixed with `_`
    - c. Don't enforce module privacy, but also don't guarantee backwards compatibility when importing objects from them. While backwards compatibility will be tied with semantic versioning for packages, this won't be enforced for internal modules.
10. **Should bitmask enums be added?** These would allow for multiple enum items to be applied as a single value, with efficient O(1) item detection. Algebraic enums or items with parameters will be allowed.
    - Example of bitmask types in Go:

        ```go
        type MyEnum uint8

        const (
            Option1 MyEnum = 1 << iota
            Option2
            Option3
        )

        var myValue MyEnum = Option1 | Option3

        // Check for an option
        (myValue & Option3) != 0

        // Assign an option
        myValue |= Option2

        // Remove an option
        myValue &^= Option1 // JavaScript: myValue &= ~Option1
        ```

    - For Klar:

        ```klar
        @flag // or '@flags'
        type MyEnum {
            .option1
            .option2
            .option3
        }

        myValue := MyEnum(.option1, .option3)
        myOpts: [MyEnum] := [.option1, .option3]
        myValue := MyEnum(myOpts...)

        // Possible approaches for checking
        myValue.has(.option3)
        (.option3 in myValue)


        // Possible approaches for setting
        myValue.set(.option2, true) // Mutating
        myValue.set(.option2) // `true` as a default param
        myValue = myValue.with(.option2) // Non-mutating
        myValue = MyEnum(myValue..., .option2)
        myValue += .option2

        // Possible approaches for removing
        myValue.set(.option1, false) // Mutating
        myValue = myValue.without(.option1, false) // Non-mutating
        myValue -= .option1
        ```

    - To consider: If items with parameters are allowed, how can they be checked?

11. **Iterators**
    - a. New `Iterator<T>` type (possibly builtin).
      Downsides
        1. Another (builtin) type with niche applications that beginners still have to learn.
        2. Iterators can yield 1 or 2 items. How will that be implemented in the type signature? Have a separate `Iterator2<A, B>` type, similar to Go, and be required to duplicate methods from `Iterator`? Or `Iterator<T>` and `Iterator<(A, B)>`
    - b. Iterators are just lists, but optimized by the compiler when used in for-loops (preferred, but future work). A function can return a list, and if it is created and iterated over using specific operations, the compiler can optimize it.
    - c. Add generator functions that can yield values to the language
12. **Should the compiler enforce identifier capitalization conventions?** (`PascalCase` for types, `camelCase` for variables). No `snake_case` for variables.
13. **Disallow tuples with 0 or 1 items?** Currently, to create a tuple with 1 item, you have to add a trailing comma `(1,)`.
14. **How should leaking `Task`s be handled by the compiler?**
    - a. Require the function spawning the task to be called with `go`
    - b. Require all tasks to be awaited
15. **Chaining assertions** - to assert both the `Result` and optional in the `Result<Int?>` type:
    - a. Does <code>!!</code> have to be used for each (twice), or
    - b. <code>!!</code> asserts both the <code>Result</code> and the optional (asserts all -> `Int`)
16. **Readonly struct fields** - What keyword should be used?
    - `readonly`? But the field isn't actually readonly. It can still be modified internally.
        > For now, `readonly` will be used to refer to this feature in RFCs.
17. **Readonly interface fields**
    - a. Add a <code>writable</code>/<code>write</code> keyword that can be applied on fields, similar to <code>readonly</code> on structs. The downside is that is another keyword, when we already have <code>readonly</code>.
    - b. All fields are writable by default, and <code>readonly</code> can be used to override.
    - c. Interface fields can't be modified at all. Mutation requires a setter (e.g. <code>setFoo()</code>).
      Note that requiring fields to be writable makes the interface harder to implement. If a struct has a field `readonly foo`, it can't implement any interface that requires `foo`. Option B implicitly makes this too restrictive.
18. **Should lists and tuples use 1-based indexing?**
    - 0-based indexing is a convention taken from C, where array indexing is equivalent to taking memory offsets. We are careful about borrowing semantics from C into Klar, especially the low-level ones.
    - Using 1-based indexing allows the list's `length` to be used as a valid index, avoiding off-by-one bugs.
    - Lua, Julia, MATLAB, R, and Fortran are examples of languages with 1-based indexing.
    - Using 1-based indexing means the `..<` operator will be removed, so only `...` is needed.
19. **Errors**
    - **My goal is for error objects to have codes attached**. My preferred way is one defined by the module creating the error, which may be an enum.

    ```
    // Module myMod
    public type ErrorCode {
        .notFound(path: String)
        .accessDenied
    }
    // A user of 'myMod' could check like:
    when err.code {
        .notFound(path) -> ...
        .accessDenied -> ...
    }
    ```

    - a. Make `Error` an interface, similar to Go, with the message as a String field, and a code (`Any` type). Also, should they be fields, or methods (so structs can compute messages; but that could also be done by an initializer)?
    - b. Make `Error` a struct
        ```klar
        type Error {
            message: String
            code: Any
        }
        ```

20. **Fallable list indexing** - Should all string/list index/slice operations return `Result`s, or crash when out of range?

21. **`Result` methods:** Should these methods be added to the `Result` type?

    ```klar
    type Result<T, E>

    func Result.isSuccess() -> Bool
    ok := myResult.isSuccess()
    // Without (idiomatic and consistent):
    ok := when myResult {
        Error -> false
        _ -> true
    }

    func Result.else<U>(errValue: U) -> T | U
    value := myResult.else('default value') // Could also be called 'Result.or()'
    // Without (also idiomadic and consistent):
    value := when myResult {
        Error -> 'default value'
        _ -> myResult
    }

    func Result.mapSuccess<U>(to successValue: U) -> Result<U, E>
    func Result.mapSuccess<U>(with mapper: func(successValue: T) -> U) -> Result<U, E>

    func Result.mapError<U>(to errValue: U) -> Result<T, U> // Could also be called 'Result.and()'
    func Result.mapError<U>(with mapper: func(errValue: E) -> U) -> Result<T, U>

    func Result.swap() -> Result<E, T> // Could also be called 'Result.invert()'
    func Result.takeSuccess() -> T?
    func Result.takeError() -> E?
    ```

22. **Should `Task` and `Regex` be builtin types?**
    - a. Builtin types (either or both) (that beginners have to be introduced to)
    - b. They should be imported from the standard library (ex. `concurrency.Task` and `regex.Regex`). The downside is they have native syntax in the language, but to annotate, they have to be imported.
