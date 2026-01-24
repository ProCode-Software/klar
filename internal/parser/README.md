# The Klar Parser

After tokenizing files, the list of tokens in each file are parsed into an abstract syntax tree (AST).

Before parsing, comments are removed from the token list and statement terminators ("semicolons") are [inferred](#semicolon-inference).

The Klar parser uses a [Pratt parser](https://martin.janiczek.cz/2023/07/03/demystifying-pratt-parsers.html) algorithm, allowing for operator precedence and for literals to be distinguished from operators. See the [Pratt parsing](#pratt-parsing) section for details.

## Semicolon Inference

- `(*Parser).InsertEOS()`
- [`eos.go`](./eos.go)

Klar doesn't use semicolons to terminate statements; line breaks are used instead. Semicolon inference scans the tokens and replaces `Newline` tokens with `EndOfStatement` (EOS) "semicolon" tokens where needed, otherwise removes them. For example, EOS is not needed if there is a line break after a `(` token, but is needed after `}`.

These rules are defined in the `CanAddEOSAfter` and `CanGoOnNewline` functions in `eos.go`. `CanGoOnNewline` checks if the token before or after a newline continues a statement, such as infix operators (`+`, `...`, `:`, etc.). `CanAddEOSAfter` checks if the token before a newline doesn't terminate a statement, such as keywords (`func`, `when`, etc.)

As defined in the EOS rules, opening brackets used as infix operators cannot go on a new line, such as `(` in a function call. This fixes issues present in JavaScript where an explicit semicolon is needed before an IIFE or destructure assignment.

There are still caveats in the EOS algorithm, including:

- Statements can't start with a `+` (positive prefix), `-` (negative prefix), or `.` (enum shorthand), though this problem is less important because these would be used in unused expressions, which are invalid in Klar.
- Commas instead of newlines are required after comparison shorthands are in when cases, or spread expressions in maps. These issues can be mitigated by the parser, but at the expense of ergonomics and codebase readability.
- Wildcard imports end in `*`, so there will be no EOS after those import statements. This problem is also mitigated by the parser.

Additionally, the `InsertEOS` function also collects and removes comment tokens from the token collection (`(*Parser).ParseComment`).

## Pratt parsing
- Parsing: [`parse.go`](./parse.go)
- Handlers: [`handlers.go`](./handlers.go)
- Binding powers: [`binding_power.go`](./binding_power.go)

The parser is a Pratt parser, which defines literals as NUDs (null denoted), and infix/postfix operators as LEDs (left denoted) with different binding powers (precedence). To parse an expression, the parser reads a NUD, then reads zero or more LEDs, while the binding power of the LED is greater than the given binding power. When a LED is handled, it may parse another expression after it, repeating this entire process, but given the LED's (higher) binding power.

There is a great YouTube [tutorial](https://www.youtube.com/watch?v=1BanGrbOcjs&list=PL_2VhOvlMk4XDeq2eOOSDQMrbZj9zIU_b&index=3) by @tlaceby, the same one used to write the Klar parser.

### Binding power
As a simple definition, binding powers determine how much tokens stick to each other. Logical operators (`||`, `&&`) have the lowest binding power, while index operators (`.`, `[`) have the highest. Multiplication/division/modulo operators are higher than addition/subtraction. For short, if A has a higher binding power than B, A goes inside B.

Binding powers are also defined for primary expressions (literals; highest binding power) and unary operators. While unary operators are null-denoted, binding powers are used for limiting the binding power of operators that can go inside those expressions. Addition wouldn't go inside unary expressions, otherwise `-2 + 3` would be `-(2 + 3)`.

> Similar to Python, the exponentiation operator (`^`) has a higher precedence than unary, so `-2 ^ 4` is the same as `-(2 ^ 4)`.

### NUDs
NUDs begin subexpressions. Examples of NUDs include:
- Opening parentheses (`(`) used for grouping and tuples
- Strings, numbers, `true`, `false`, `nil`
- Lists (`[`)
- The `.` that begins enum shorthand literals (`.enum`)
- Unary operators (logical inverse `!`, negative sign `-`, `go`)
- `when`
- `for`

### LEDs
LEDs are always preceded by a NUD.
- Arithmetic (`+`, `-`, `*`, etc.)
- Indexing (`.`, `[`)
- Logical operators (`||`, `&&`)
- Relational operators (`<`, `==`, `>=`, etc.)

## The AST
- [`internal/ast/ast_types.go`](../ast/ast_types.go)

The AST nodes are defined in the `ast` package. These define all of the statements and expressions used by Klar. The parser finishes with a `ast.Program` node that contains nested nodes for all statements in the file. The program can be used later for analysis and formatting.

## Parsing contexts
The parser parses in different contexts to separate where operators can be used and how they are parsed. Each context defines its own NUD and LED handlers.

### Statements
- [`statement.go`](./statement.go)
- [`declaration.go`](./declaration.go)
- [`modifiers.go`](./modifiers.go)

### Expressions
- [`expression.go`](./expression.go)
- [`primary.go`](./primary.go)

### Types
- [`type.go`](./type.go)

### Destructures
- [`destructure.go`](./destructure.go)

Note that in the destructure context, everything is handled as a NUD.

## Errors
- [`internal/errors/parse_error.go`](../errors/parse_error.go)
- `(*Parser).Error`
- `(*Parser).Expect...`

Errors are stored as `ParseErrors`