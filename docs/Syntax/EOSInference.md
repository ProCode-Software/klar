# New EOS Inference Architecture

## The Problem

Currently, terminators are never inserted before and after infix operators. In some places, such as after generic type declarations, this isn't wanted.

1. **Type declarations with a generic value**

```klar
type X = Result<String>
func X() = "Hello"

// Is read to the parser as:
type X = Result<String > func X() = "Hello"
```

To prevent this, the type must be put in parentheses.

2. **Star imports**

```klar
import klar.http.*
print("Hello, World!")

// Read as:
import klar.http. * print("Hello, World!")
```

This is prevented in the parser by not requiring a terminator after a statement that ends with an asterisk.

3. **Shorthand comparisons in when-expressions**

```klar
when num {
    1 -> return 0
    <= 5 -> return 1
}

// Read as:
when num {
    1 -> return 0 <= 5 -> return 1
}
```

This can be prevented by ending the previous case with a coomma.

4. **Pipeline statements that start with an enum (or negative number)**

```klar
func getColor(num: Int) -> Color {
    when num {
        < 0 -> return .green
    }
    .red |> return
}

// Read as:
func getColor(num: Int) -> Color {
    when num {
        < 0 -> return .green
    }.red |> return
}
```

To prevent this, the enum must be put in parentheses.

## New token tracking in the parser

```go
type Parser struct {
    prevTok, currTok, peekTok Token
    // The name of this is subject to change
    eosRule eosFunc
}

// The Token passed will always be of kind [lexer.Newline] or [lexer.EOF].
type eosFunc func(t Token, i int) (insertEOS bool)
```

`eosRule` can dynamically be changed depending on the parsing context:

- One implementation would be the default EOS algorithm
- Another would be used in type declarations so an EOS is inserted after `>`
    ```klar
    type Outcome = Result<Nothing> // <- EOS should be inserted here
    ```
- One in when-expressions so an EOS is inserted before binary operators used as shorthand conditions (not indented)
    ```klar
    when value {
        < 0 -> return false // <- EOS inserted
        // This is a separate case
        >= 5 -> return someVeryLongFunction().value // <- No EOS
            != 0 // indented
    }
    ```
    With this change, commas won't be needed in the case before a shorthand condition.

## New EOS Rules

1. **Partially indentation-based**. For operators that can possibly start a statement or expression, an EOS will be inserted if it isn't indented.

    ```klar
    x := 'Hello' // EOS inserted
    .length // not indented

    x := 'Hello' // <- no EOS
        .length // indented
    ```

    Note that the `.` token can denote an enum shorthand literal.

2. **Bracket-aware**. If the newline is inside parentheses or brackets, no terminator will be inserted.
    ```klar
    (1 // <- no EOS
    +2)

    (
        1 // <- no EOS
    +2)
    ```
3. **Syntax-aware**

    > [!IMPORTANT]
    > I think this is very controversial and makes Klar's EOS inference more similar to JavaScript's. Please send your feedback on this and if this should be a rule.

    ```klar
    x := 2 // <- no EOS
    + 2 // '+' can't be used to start a statement or expression (no '+' prefix in Klar)
    // Indenting that statement won't change anything

    x := 2 // <- EOS inserted
    - 2 // '-' is the negate operator, which can start an expression.
    // Indenting that statement will cause an EOS to not be inserted
    ```
