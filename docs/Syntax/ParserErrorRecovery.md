# Klar Parser Error Recovery

## Goals for the parser

Language servers like [KlarLS](https://github.com/ProCode-Software/klar/tree/main/internal/lsp/README.md) need to have error-tolerant parsers so they can still function while the user is still typing. To improve the user experience, a single unmatched bracket shouldn't trigger several more errors. To make a universal parser suitable for use in LSPs, we have defined these goals:

1. The parser **must not** crash or hang with any program given, including invalid programs.
2. Errors **should not** cascade from a single token.
3. Fixable errors **should** still produce a valid AST. (This is so the formatter in the LSP can fix them. It **must not** remove any token the user typed.)
4. It is **recommended** for small style opinions (such as brace placement) to not be errors so the formatter can fix them.

## Scenarios

```klar
a(b, c+
return true
// ^ stop at statement keyword

when x {
    _ -> print(1)
return // <-

type X: {
    // ^ "expected type"
type Z
// ^ invalid token for struct field or enum item; start a new statement

type A: B<Int {
           // ^ continue from here (should we auto-insert '>' when formatting?)

f(a+,
 // ^ continue from here

  f(,)
//  ^ "expected expression"; skip

func x(name( ) {}
        // | ^- continue from here
        // skip
```

## 1. Continuing Tokens

When skipping tokens after an error, stop skipping after reaching one of these.

- In a function declaration: `)`, `=`, and `}`
- Type declaration: `:` and `{`
- Function call: `,` and `)`

## 2. Stacks

```klar
  f(1, ['a', +)
// |   |      ^ outer CT found; exit both
// |   CTs `]` and `,` pushed
// CTs `,` and `)` pushed

   // CTs: `,` and `(`
   // |            | CT: `,` and `>`
func x(name: Result<String { // <- CT: `}`
   // *------------*-------^ end
   //              *-------^
    y := process([1, 2,
             // |^ CTs: `,` and `]`
             // CTs: `,` and `}`
} // <- list, call, and func end here
```

Multiple stacks will be needed to keep track of the continuing tokens from previous levels.

## Another Scenario

```klar
   func x() =
   func y() {}
// ^ parsed as lambda expression

   func z() =
   while {}
// ^ starts to read expression, but this is a statement
```

### 3. Solutions

- a. Allow names in lambda expressions in error recovery mode, and braces in for-expressions
- b. Rely on indentation: no indent = 2 statements
    ```klar
       func x() = // <- stop
       func y() {}
    // ^ unindented

     func x() = // <- continue
       func y() {}
    // |    ^ name not allowed in lambda expression
    // indented
    ```
- c. Ignore this scenario and report more errors

## 4. Implementation

### New Parser Fields

```go
type Parser struct {
    stack Stack // Continuing tokens
    collectionStack Stack // Continuing tokens that are brackets / represent collections
    collectionEnd bool // Whether the current collection has ended
}
```

Type `Stack` will be a bitmask with bits representing all possible continuing tokens in the Klar language.

```go
type Stack uint16 // Could also be uint32, depending on the count

const (
    commaCT Stack = 1 << iota
    leftBraceCT
    // ...
)
```

### Keeping track of the stack

**collectionEnd** is set to `true` when:

- A continuing token is encountered
    ```klar
    x := [(1, 2), (3] // <- at `]`
    ```
- Or, a new statement starts
    ```klar
    func x() =
    while {} // <- at `while`
    ```

`Parser.restoreTokenStack()` sets **`collectionEnd`** to false when the parser has fully recovered. When a collection expression starts, the method will push continuing tokens to the stack, then in a `defer` call, restore the stack. This allows us to keep a stack of O(1) space rather than a slice of stack nodes that each define its continuing tokens and whether it has ended.

```klar
x := #{a: [(1, 2), (3}
                  // ^ collectionEnd = true for tuple and list, false immediately after
y := [1, 2, 3,
func z() {}
// ^ collectionEnd = true for list, false once exited

type A: B<C { // <- ends here; different collection starts
    // ^ collectionStack |= `{`
```

In Go, a collection would be parsed like this:

```go
func (p *Parser) ParseList() *ast.ListLiteral {
    oldStack, oldCollStack := p.pushTokenStack(rightBracketCT|commaCT, rightBracketCT)
    defer p.restoreTokenStack(oldStack, oldCollStack)

	var items []ast.Expression
	p.Expect(lexer.LeftBracket)
	for p.HasTokens() && !p.collectionEnd && p.CurrKind() != lexer.RightBracket {
	    items = append(items, p.ParseExpression(ExpressionBindingPower))
		// Another option is to omit the `!p.collectionEnd` check and not report
		// an error when a comma is missing if collectionEnd if true
		if !p.collectionEnd && p.CurrKind() != lexer.RightBracket {
		    p.Expect(lexer.Comma)
		}
	}
	return &ast.ListLiteral{Items: items}
}
```

### Skipping

Skip on any unexpected/invalid token until any continuing token is reached. **It may be in the same place**.
