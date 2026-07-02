# Klar 🐨

Klar is Danish for _clear_. In Klar, if you read code, that's what it does, with no hidden behavior.

> [!WARNING]
> This project is a **work in progress**. The language can (will) change at any time and is not recommended for production use yet. We would greatly appreciate feedback and contributions!

## Goals

- ⏩ **Progressive.** Unafraid to redesign features from other languages and pioneer new approaches. _Klar is different from the other langs!_
- 👓 **Clear.** What you read is what it does.
- 🛡️ **Safe.** All errors must be handled. Minimal runtime crashes. Type-safe. No null exceptions.
- 🔮 **Modern.** Klar is a language designed from scratch, without weird C-derived features.
- 😀 **Fun.** Klar code is clean, short, and pretty, making it fun to write.
- 🤏 **Small language.** In Klar, there is only one way to do something, resulting in a lower learning curve.
- 🤓 **Beginner-friendly.** Klar is easy for non-developers to learn, making it a perfect first language.
- 🚀 **Advanced tooling.** Klar comes with a fast compiler, formatter, linter, package manager, version manager, and more built-in.
- ♻️ **Interoperable.** Easy to include JavaScript and adopt incrementally.
- ⚡ **Fast compile times.** The compiler makes great use of caching and incremental compilation, and its written in Go! 🐹

## Inspiration

- Idioms from Go
- Error handling from Rust and [Gleam](https://gleam.run/)
- Type system from TypeScript
- Syntax from Swift and Go
- `when` expressions from Kotlin
- Enums and pattern matching from Gleam and Rust
- Tooling from Go and JavaScript
- Compiler from Go and Gleam

## Installation & Quickstart

To get started with Klar, run the installer with the following command:

```sh
curl -fsSL https://raw.githubusercontent.com/ProCode-Software/klar/main/install.sh | bash
```

If you're on Windows, consider using Git Bash or WSL to run the installer.

The installer gives you two options on how to install Klar:

1. **Build from source**, which builds Klar directly from the latest commits on the main branch. This requires Git and the [Go](https://go.dev/) toolchain to be installed.
2. **Download a prebuilt binary**, which doesn't require Go or Git to be installed. Note that prebuilt binaries are built occasionally and won't have the latest features and bug fixes. The installer will automatically download the appropriate binary from the [releases page](https://github.com/ProCode-Software/klar/releases).

<details>
    <summary><b>Manual prebuild installation</b></summary>

1. Download the Klar and Glas prebuilds for your platform from the [releases page](https://github.com/ProCode-Software/klar/releases).
2. From the same page, download a copy of the standard library (`stdlib.zip`).
3. Extract the standard library to any directory
4. Set the `$KLAR_STD` environment variable to that path. It should point to the unzipped directory named `std` (not `stdlib` or `std/src/std`).
5. When you're done, you can run the downloaded Klar and Glas binaries from the command line.

</details>

After you run the installer, you'll have access to the Klar compiler, toolchain, and Glas package manager.

### Usage

```sh
klar --help  # Show commands
klar repl    # Easily run Klar code without a project. You'll see a printed AST of your code.
klar new     # Interactively start a new project
klar build   # Build your project
klar run     # Quickly compile and run your project
klar test    # Run your project's tests
klar format  # Format the files in your project
glas --help  # Use Glas, the Klar package manager
```

> [!NOTE]
> While the typechecker is in active development, you may set the `NO_TYPECHECK=1` environment variable to disable typechecking. When disabled, only syntax is checked. Otherwise, you can see what we're working on, but you may see type errors for unimplemented functionality.

## Examples

### Factorial

```swift
// Calculates the factorial, the product of all positive integers up to n.
func factorial(n: Int) = when n {
    <= 1 -> 1
    _ -> n * factorial(n - 1)
}

// Print the factorial of some numbers
numbers := [2, 3, 4, 5, 6, 7, 8]
for num in numbers {
    print("Factorial of {num}: {factorial(num)}")
}
```

### HTTP Requests

```swift
import klar.http.requests

BASE_ENDPOINT := 'https://api.github.com'

func getTopLanguage(user user, repo repo: String) -> Result<String?> {
    endpoint := "{BASE_ENDPOINT}/repos/{user}/{repo}/languages"
    res := try requests.get(endpoint)
    json := #{String: Int}(res.json())!!
    pair := json.sortedPairsByValues(reverse: true)[0]
    when pair {
        none -> return none
        (lang, _) -> return lang
    }
}

topLanguage := getTopLanguage(user: "ProCode-Software", repo: "klar")
print(topLanguage)
```

### Testing

```swift
// sum.klar
public func sum(a, b: Int) = a + b

// test/sum.test.klar
import klar.test.{expect}

func testSum() {
    expect(sum(1, 2) == 3)
    expect(sum(-5, 6) == 1)
}
// Run using `klar test`
```

## Contributing, Development, Issues, and PRs

For a basic contribution guide for ths repo, see [CONTRIBUTING.md](https://github.com/ProCode-Software/klar/blob/main/CONTRIBUTING.md). It also contains a guide on learning how the Klar compiler works.

## Feedback, Comments, and RFCs

Klar uses RFCs to collect feedback on how features should be implemented in the language. To view the active RFCs and submit feedback, see the [RFCs discussion page](https://github.com/ProCode-Software/klar/discussions/categories/rfcs).

For other feedback and discussions, where you can request features and comment on existing ones, see the general [discussions page](https://github.com/ProCode-Software/klar/discussions).

## License

The Klar project is licensed under two licenses:

- Compiler and commands (`internal`, `cmd`): [GNU General Public License v3.0](https://github.com/ProCode-Software/klar/blob/main/LICENSE-GPL)
- Standard library, tools and other files in this repo (`pkg`, `std`, `klar-vscode`, etc.): [Apache License 2.0](https://github.com/ProCode-Software/klar/blob/main/LICENSE-APACHE)

The GPL license does not apply to code generated by the Klar toolchain.
