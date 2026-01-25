# Klar 🐨
Klar is Danish for *clear*. In Klar, if you read code, that's what it does, with no hidden behavior.

> [!WARNING]
> This project is a **work in progress**. The language can change at any time and is not recommended for production use yet. We would appreciate contributions!

## Goals
- 👓 **Clear.** What you read is what it does.
- 🛡️ **Safe.** All errors must be handled. Type-safe. No null exceptions.
- 🔮 **Modern.** Klar is a language designed from scratch, without the C nonsense.
- 😀 **Fun.** Klar code is clean, short, and pretty, making it fun to write.
- 🤏 **Small language.** In Klar, there is only one way to do something. This means a lower learning curve.
- 🤓 **Beginner-friendly.** Klar is easy for non-developers to learn, making it a perfect first language.
- 🚀 **Advanced tooling.** Klar comes with a fast compiler, formatter, linter, package manager, version manager, and more built-in.
- ♻️ **Interoperable.** Easy to include JavaScript and adopt incrementally.

## Inspiration
- Idioms from Go
- Error handling from Rust and Gleam
- Type system from TypeScript
- Syntax from Swift and Go
- `when` expressions from Kotlin
- Enums and pattern matching from Gleam and Rust
- Tooling from Go and JavaScript
- Compiler from Go and Gleam

## Examples
### Factorial
```swift
// Calculates the factorial, the product of all positive integers up to n.
func factorial(n: Int) -> Int {
    when n {
        <= 1 -> return 1
        _ -> return n * factorial(n - 1)
    }
}

// Print the factorial of some numbers
numbers := [2, 3, 4, 5, 6, 7, 8]
for num in numbers {
    print("Factorial of {num}: {factorial(num)}")
}
```

## License
Klar is licensed under the [Apache License 2.0](https://github.com/ProCode-Software/klar/blob/main/LICENSE), [with additional clauses](https://github.com/ProCode-Software/klar/blob/main/LICENSE#L176).
