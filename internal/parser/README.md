# The Klar Parser

After tokenizing files, the list of tokens in each file are parsed into an abstract syntax tree (AST).

Before parsing, comments are removed from the token list and statement terminators ("semicolons") are [inferred](#semicolon-inference).

The Klar parser uses a [Pratt parser](https://martin.janiczek.cz/2023/07/03/demystifying-pratt-parsers.html) algorithm, allowing for operator precedence and for literals to be distinguished from operators. See the [Pratt parsing](#pratt-parsing) section for details.

## Semicolon Inference

- `(*Parser).InsertEOS()`
- [`eos.go`](./eos.go)

Klar doesn't use semicolons to terminate statements; line breaks are used instead. Semicolon inference scans the tokens and removes `Newline` where not needed. The remaining newlines act as End-of-Statement (EOS, "semicolon") tokens. All statements must end in them. For example, EOS is not needed if there is a line break after a `(` token, but is needed after `}`.

These rules are defined in the `CanEndStatement` and `ContinuesStatement` functions in `eos.go`. `ContinuesStatement` checks if the token before or after a newline continues a statement, such as infix operators (`+`, `...`, `:`, etc.). `CanEndStatement` checks if the token before a newline doesn't terminate a statement, such as keywords (`func`, `when`, etc.)

As defined in the EOS rules, opening brackets used as infix operators cannot go on a new line, such as `(` in a function call. This fixes issues present in JavaScript where an explicit semicolon is needed before an IIFE or destructure assignment.

There are still caveats in the EOS algorithm, including:

- Statements can't start with a `-` (negative prefix), or `.` (enum shorthand), though this problem is less important because these would be used in unused expressions, which are forbidden in Klar.
- Wildcard imports end in `*`, so there will be no EOS after those import statements. This problem is also mitigated by the parser.
- We can't use the shorthand struct initialization `.(...)` syntax in variable destructuring (e.g. `.(name, age) := Person('John', 32)`) because an EOS won't be inserted before the dot, making it part of the previous expression, then a syntax error because `x.(...)` isn't valid in Klar.
- The worst problem is in when-expressions. In addition to enum shorthands, shorthands for operators are supported (e.g. `when num { < 0 -> ... }`), where the left-hand side is inferred as the subject of the when-expression.
    ```klar
    when color {
        .red -> return 1 // <- No EOS
        .yellow -> return 2
        .blue -> return 3
        _ -> return Error('Invalid color')
    }
    ```
    Newlines are used to separate cases, but no EOS is inserted before `.`, so when parsing the first case body, it will be read as `1.yellow -> ...`. To avoid a syntax error, the user must explicitly put a comma at the end of the case before each `.` or relational operator (e.g. `.red -> return 1,`). This requirement can result in cryptic syntax errors if forgotten, which can easily happen. It is not required before `_` or after `}`.

Future overhauls to the EOS algorithm include on-demand EOS insertion during parsing, so different rules can be set while parsing a when-expression.

Additionally, the `InsertEOS` function also collects and removes comment tokens from the token collection (`(*Parser).ParseComment`).

## Pratt parsing

- Parser functions: [`parser.go`](./parser.go)
- General NUD/LED/statement/type parsing: [`parse.go`](./parse.go)
- Handlers: [`handlers.go`](./handlers.go)
- Binding powers: [`binding_power.go`](./binding_power.go)

The parser is a Pratt parser, which defines literals as NUDs (null denoted), and infix/postfix operators as LEDs (left denoted) with different binding powers (precedence). To parse an expression, the parser reads a NUD, then reads zero or more LEDs, while the binding power of the LED is greater than the given binding power. When a LED is handled, it may parse another expression after it, repeating this entire process, but given the LED's (higher) binding power.

There is a great YouTube [tutorial](https://www.youtube.com/watch?v=1BanGrbOcjs&list=PL_2VhOvlMk4XDeq2eOOSDQMrbZj9zIU_b&index=3) by [Tyler Laceby (GitHub)](https://github.com/tlaceby/parser-series), which was the same one used to write the Klar parser.

> If this tutorial isn't clear enough, refer to the links in the file or consider contributing!

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

## Errors

- Error codes: [`internal/klarerrs/syntax_error.go`](../klarerrs/syntax_error.go)
- `(*Parser).Error`/`(*Parser).ErrorLabelled`
- `(*Parser).Expect...`
- [`parser.go`](./parser.go)
- [`expect.go`](./expect.go)
- [`error.go`](./error.go)

Errors have the prefix `SyntaxErrorPrefix`, and codes and messages are defined in [`internal/klarerrs/syntax_error.go`](../klarerrs/syntax_error.go). Unlike other error prefixes, such as the ones used for type errors, warnings, and reference errors, syntax errors use two sets of 100 error codes, due to the number of codes needed by the compiler (additional syntax error codes are reported by the type checker). Up to 200 error codes are available.

## Parsing contexts

The parser parses in different contexts to separate where operators can be used and how they are parsed. Each context defines its own NUD and LED handlers.

Using different parsing contexts, to parse a simple variable declaration like `x: Int := 2`:

```
func (p *Parser) parseVariableDeclaration() *VariableDeclaration {
    var explicitType ast.Type
	name := p.ParseIdentifier()
	if p.CurrKind() == lexer.Colon {
	    // Explicit type annotation
	    p.Advance() // :
	    explicitType = p.ParseType()
	}
	p.Expect(lexer.ColonEqual) // :=
	value := p.ParseExpression()
	return &VariableDeclaration{
		Name:     name,
		Type:     explicitType,
		Value:    value,
	}
}
```

> [!NOTE]
> This is a simplified example of how the parser parses a variable declaration. The actual implementation
> is more complex, due to destructuring and multiple assignment in a statement.

### Statements

- [`statement.go`](./statement.go)
- [`declaration.go`](./declaration.go) _(Variable/type/function declarations, `public` modifier)_

### Expressions

- [`expression.go`](./expression.go)
- [`primary.go`](./primary.go) _(Strings, numbers, booleans, regexes, `none`)_
- [`when.go`](./when.go) _(When-expressions)_
- [`string_escape.go`](./string_escape.go) _(String escape validation and interpolations)_

### Types

- [`type.go`](./type.go)

## Identifiers

- [`identifier.go`](./identifier.go)
