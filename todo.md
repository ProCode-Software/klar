# TODO
## Lexer
- maybe: ..< operator
- [done] indentation for `` string
## Parser
- [done] switch concrete type for ast.Node to pointer
    (e.g. Node is pointer to *BinaryExpression)
- [done] regex literals
- [done] version literals in attributes
- [done] use strings in FloatLiteral and IntegerLiteral to preserve formatting
- add position of operator to BinaryExpression type
- [done] enum parameters (and maybe generic)
    ```klar
    .some(1)
    type Option { .some(Any) | none }
    type Option<T> { .some(T) | none } // with generic
    ```
- [done] handle eos between param label and name
- multi-variable assignments
- list casting
    ```klar
    [Int](list)
    ```
- map, tuple, and list destructure assignment
- in when case, map destructuring
- [1/2] `for` loop
    - tuple declarations also needed
- [done] shorthand map items
- [done] chained function types
- [done] string pattern matching in `when` case

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
