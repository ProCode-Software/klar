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
    <ol type="a">
        <li>Private by default</li>
        <li>Private fields should be prefixed with `_`</li>
        <li>Add a new <code>private</code> keyword</li>
    </ol>
7. **How to distinguish variable values from unwrapped variable declarations in `when` pattern matching**
    <br>Example:
    ```klar
    val := "foo"
    when value {
        // Is val the first field in MyStruct, or "foo"?
        MyStruct(val) -> { ... }
        // Is val a type, or "foo"?
        val -> { ... }
    }
    ```
    Note that we don't want the meaning of `foo` to depend on whether it is defined as a variable or type. 
8. **Placement of pipeline values in function calls** Unless the `result` (or `value`) variable is used in a step:
    <ol type="a">
        <li>First parameter</li>
        <li>Last parameter</li>
        <li>Require the use of <code>result</code> for functions that take multiple parameters</li>
    </ol>
9. **Private modules**
    <ol type="a">
        <li><code>internal</code> folder that can't be imported by packages outside the project</li>
        <li>Folders prefixed with <code>_</code></li>
    </ol>
10. **Should bitmask enums be added?** These would allow for multiple enum items to be applied as a single value, with efficient O(1) item detection. Algebraic enums or items with parameters aren't allowed.
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
11. **Iterators**
    <ol type="a">
        <li>New <code>Iterator</code> type (possibly builtin)</li>
        <li>Iterators are just lists, but optimized by the compiler when used in <code>for</code> loops</li>
        <li>Add generator functions that can yield values to the language</li>
    </ol>
12. **Should the compiler enforce identifier capitalization conventions?** (`PascalCase` for types, `camelCase` for variables)
13. **Disallow tuples with 0 or 1 items**
14. **Should `opaque` be renamed?** The semantics of `opaque` is different from other languages, where opaque types can't be indexed.
15. **How should leaking `Task`s be handled by the compiler?**
    <ol type="a">
        <li>Require the function spawning the task to be called with <code>go</code></li>
        <li>Require all tasks to be awaited</li>
    </ol>
16. **Chaining assertions** - to assert both the `Result` and optional in the `Result<Int?>` type:
    <ol type="a">
        <li>Does <code>!!</code> have to be used for each (twice), or</li>
        <li><code>!!</code> asserts both the <code>Result</code> and the optional (asserts all)</li>
    </ol>
17. **Readonly struct fields** - What keyword should be used?
    - `readonly`? But the field isn't actually readonly. It can still be modified internally.
    > For now, `readonly` will be used to refer to this feature in RFCs.
18. **Readonly interface fields**
    <ol type="a">
        <li>Add a <code>writeable</code>/<code>write</code> keyword that can be applied on fields, similar to <code>readonly</code> on structs.</li>
        <li>All fields are writable by default, and <code>readonly</code> can be used to override.</li>
        <li>Interface fields can't be modified at all.</li>
    </ol>
19. **Should lists and tuples use 1-based indexing?**
    - 0-based indexing is a convention taken from C, where array indexing is equivalent to taking memory offsets. We are careful about borrowing semantics from C into Klar, especially the low-level ones.
    - Using 1-based indexing allows the list's `length` to be used as a valid index, avoiding off-by-one bugs.
    - Lua, Julia, MATLAB, R, and Fortran are examples of languages with 1-based indexing.
20. **Errors**
    - TODO
21. **Fallable list indexing** - Should all list index operations return `Result`s, or crash when out of range?
22. **Should accumulative `for` expressions remain in the language?**
    - These allow assignment operators such as `+=` to be used in `for` expressions to accumulate a value instead of creating a new list. (reduce)
        ```klar
        items := [1, 2, 3, 4, 5, 6]
        sum := for item in items += item
        ```
    - Should `for` expression *blocks* remain in the language?
    <br>These allow a new list to be created by returning values, also allowing control flow such as `next` and `stop`. (filter)
        ```klar
        numbers := [1, 2, 3, 4, 5, 6]
        evenNumbers := for num in numbers {
            when item % 2 {
                0 -> return item
                _ -> next
            }
        }
        ```
