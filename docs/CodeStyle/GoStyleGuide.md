# Go Style Guide

This guide provides an overview of the Go conventions used in the Klar repository. It is still recommended that you read a variety of files in the codebase to understand all of the conventions used in the project. See the [Examples](#examples) section for some good examples.

## General Formatting

- Format using [gofumpt](https://github.com/mvdan/gofumpt) (a stricter version of `gofmt`)
- 90-column line length
    - Up to 94 columns (and sometimes 98 for function signatures) is acceptable if it's not worth breaking the line.

## File Names

- Use `snake_case` for file names (e.g. `type_declaration.go`).
- Avoid the use of single, short abbreviations in file names (use `expression.go` instead of `expr.go`, but use `util.go` instead of `utils` or `utilities`).

### Newlines & Grouping

AI tends to add unnecessary newlines between, related statements inside functions, usually after checks and before returns.

- Use a single blank line between functions and methods (enforced by gofumpt).
- **Do not** add unnecessary newlines between short, related statements inside functions.
- Don't always add newlines before returns.
- Try to remove newlines separating short groups of statements. The longer each group gets, the more justifiable the blank line separator is.
- Try to collapse single-statement functions into a single line if they fit within 90-columns (there's less leeway to go over that limit).
- No newline separating each collapsed function if they have the same receiver type and similar purposes.

Blank lines may separate:

- Distinct paths functions can take (e.g. typechecking an interface field vs. method)
- Major steps within a function

## Code Organization

### File Layout

1. Type declarations first
2. Factory functions (`NewType`) after EACH type
3. Variables before or after ALL types, depending on how important they are, and if they're exported. Variables commonly used by multiple files should be placed before types.
4. Methods and functions, ordered top-down by call level.
    1. `func Parse()`
    2. Functions called directly by `Parse()`, such as `ParseStatement()`
    3. After ALL of those, functions called directly by `ParseStatement()`, repeating the pattern

### Sectioning

Sometimes, sections can be used to divide major groups of functionality in a single file. The number of `=` characters don't match the title length. Section doesn't have to match -- just type a few. You don't have to count, and don't make them too long

```go
// Basic Literals
// =======

// Accessors & Collections
// ======

// To implement [reporter.Error].
// ======

// Assignments & Destructuring
// =======
```

### Lettered/Numbered Sections

> AI is going to overuse this, so I'm going to say use this sparingly.

Within functions, lettered/numbered comments can be used to mark steps in a function, or paths it can take.

- Use numbers to indicate the order of execution.
- Use letters to indicate the path taken by the function.

The best use of this is in [`ParseStatement`](../../internal/parser/parse.go) in the parser

```go
func (p *Parser) ParseStatement(flags parseFlags) ast.Statement {
	kind := p.CurrKind()
	defer func() { ... }()

	// A. Try to parse a full statement
	if res, handled := p.handleStatement(kind); handled {
		expectEOS()
		return res
	}
	var expr ast.Expression
	var ok bool
	// B. Start with a NUD
	if next := p.PeekKind(); kind == lexer.Underscore && (...) {
		// Allow discard assignments
		expr = rangeFromToken(&ast.Discard{}, p.Advance())
	} else if expr, ok = p.handleNUD(kind); !ok { // Expression NUD
		p.nudError()
		return &ast.BadExpression{Token: kind}
	}

	// C. Try to parse a statement LED
	// Don't parse comma assignments if comma is a terminator
	if flags&allowCommaTerminator == 0 || p.CurrKind() != lexer.Comma {
		if stmt, ok := p.handleStatementLED(p.CurrKind(), expr); ok {
			return stmt
		}
	}
	// D. Otherwise parse an expression LED
	expr = p.ParseLED(expr, ExpressionBindingPower)

	// E. Then parse a statement LED after the expression, unless a comma
	// is a terminator (don't parse comma assignments)
	if flags&allowCommaTerminator == 0 || p.CurrKind() != lexer.Comma {
		if stmt, ok := p.handleStatementLED(p.CurrKind(), expr); ok {
			// Assignment statement
			return stmt
		}
	}
	// F. This statement is an expression statement
	return copyPos(expr, &ast.ExpressionStatement{Expression: expr})
}
```

Though some steps are sequential, there are still various important paths the function can take, making letters more useful than numbers. The comments after the letters do the work of distinguishing between options and sequential steps.

Here's another example using numbered sections, in [`Loader`](../../internal/build/loader.go):

```go
// Load loads the modules of ld's Input as well as their dependencies in the same
// package, parsing their files, and returns a [Loaded] struct.
func (ld *Loader) Load() (*Loaded, error) {
	loaded := &Loaded{}

	// 1. Resolve and parse modules
	modules, _, err := ld.ResolveInputModules()
	if err != nil {
		return nil, err
	}
	var (
		cachedCh         = make(chan *Module, len(modules))
		needsTypeCheckCh = make(chan *Module, len(modules))
		eg               errgroup.Group
	)
	for _, mod := range modules {
		eg.Go(func() error { ... })
	}
	...

	// 2. Delete removed modules/files from cache
	for mod := range cachedCh {
		loaded.cached = append(loaded.cached, mod)
		// TODO
	}
	close(needsTypeCheckCh)

	// If the input is a single file, we're not going to sort the modules.
	if ld.IsSingleFile() {
		if len(needsTypeCheckCh) > 0 {
		    ...
		}
		return loaded, nil
	}

	// 3. Order the modules by dependency order
	g := graph.New[string]()
	for mod := range needsTypeCheckCh {
	    ...
		g.AddVertex(importPathStr)
		for dep := range mod.Deps {
			// 4. Stdlib imports are added to a separate slice to be loaded,
			// unless we're currently loading the stdlib itself.
			if dep.IsStdlib() {
				loaded.stdlibDeps = append(loaded.stdlibDeps, dep)
				continue // Stdlib modules are always compiled first
			}
			g.AddEdge(dep.String(), importPathStr)
		}
	}
	// 5. Load the dependency modules that are in the current package but not
	// inputs before sorting.
	// Example: If the input is a.b, and a.b depends on a.c, we have to load it.
	if err := ld.loadPackageDeps(g); err != nil {
		return nil, err
	}

	if loaded.sortedDeps, err = g.Toposort(); err != nil {
		return loaded, &InterfaceError{Code: ErrDepCycle, Err: err}
	}
	return loaded, nil
}
```

### Type and Variable Grouping

Group 3+ related variables or types using a block:

```go
var (
	commands = klarcmd.KlarCommands
	aliases  = klarcmd.KlarCommandAliases
	profiler prof
)

type (
	StringFragment interface{ StringFrag() }
	EscapeFragment struct{ Value StringEscape }
	TextFragment   = lexer.TextFragment
)
```

Groups should be used very sparingly for types. Never use them for multiple medium to large-sized structs or interfaces.

## Naming Conventions

Ensure readers know the meaning of your variable names, without having to go to the declaration. Rarely, if only a long name can fully describe a variable's purpose, you can use a shorter name (usually <= 10 chars), and add a comment where it's declared.

### Receiver Names and Single-Letter Variables

Unlike other languages, it is common in Go for variables to be a single letter. These single-letter names are allowed for:

- Most method receivers
- A single function parameter or return result that is the subject of the function; the one that is mutated and read throughout the function.

Sometimes, multi-letter method receivers are more suitable. Some examples of types and their receivers:

| Type              |   Context    |   Receiver    |
| :---------------- | :----------: | :-----------: |
| `Parser`          |              |      `p`      |
| `Lexer`           |              |      `l`      |
| `Checker`         | Type checker |      `c`      |
| `Tuple`           |    Types     |  `t`, `tup`   |
| `Error`           |   klarerrs   |  `e`, `err`   |
| `InterfaceError`  |   Compiler   | `err`, `ierr` |
| `Loader`          |   Compiler   |     `ld`      |
| `PackageCompiler` |              |     `pkc`     |
| `ProjectCompiler` |              |  `pc`, `pjc`  |
| `Compiler`        |              |      `c`      |
| `Command`         |     CLI      |     `cmd`     |
| `Lockfile`        |  glas.lock   |     `lf`      |

Always consistently use the same receiver name for all methods on a type.

### Loop Variables

- Most of the time, a single-letter is sufficient for numeric indices. `i` should be the default.
    - This variable can be shadowed. In fact, it is common for far-away loops (such as typechecking each item in an interface, then each parameter if it is a method).
    - If the next loop is tightly related to the previous one, use `j` for the next one.
    - Otherwise, use more meaningful names so readers know the purpose of the variable (`somethingI`). Never use `somethingJ`, use `somethingElseI`.
- For a map value, or list element, follow the same rules as local variables.

## Local Variables

You should be able to have an idea of the variable's purpose without needing to go back to the definition. If you are shortening a name, and its meaning is ambiguous, spell it out (e.g. `count` instead of `ct` or `cnt`)

"Element": `el`, `elm`, or `elem` are allowed
"Item": Only use `item` (or `someItem`), not `it` or anything else
"Input": `inp` only (Use `i` sparingly -- it may be confusing with a numeric index)
Function variables: `fn`, not `fnc`. `cb` is allowed for "callback", and `pred` is allowed for "predicate".

### Forbidden Names

- Do not include `idx` in any name (e.g. `idx`, `colorIdx`, `getCharIdx`). For local variables, use `i` (e.g. `i`, `colorI`). For functions and struct fields, spell `index` (e.g. `currIndex`, `getCharIndex`).

## Error Messages

> These don't apply to user-facing errors (`klarerrs.Error`, `build.InterfaceError`, etc.). These apply to general errors returned by functions (such as where the `errors` package is used), and panic messages.

- The first sentence should start with a lowercase letter. For a user-facing error like `klarerrs.Error` that capitalizes error messages, but also implements Go's `error` interface, don't worry about it being capitalized.
- Contractions are allowed
- In panic messages, try to explain what shouldn't happen, rather than always using `panic("unreachable")`.

## Exports

Prefer exporting fields over getter (and setter) functions. (`Token.Attrs` instead of `Token.attrs` with `Token.Attrs()`)

## Comments & Documentation

### Doc Comments

- Start with the name of the symbol being documented.
- Make them below 90 columns (preferably 75-85), unless the whole comment fits on one line (maybe up to 96), and it isn't worth wrapping.
- Use square brackets for symbol cross-references: `[lexer.Token]`.

```go
// A Parser parses lexer tokens into an abstract syntax tree (AST).
type Parser struct { ... }

// Curr return the [lexer.Token] at the current parser index.
func (p *Parser) Curr() lexer.Token { ... }
```

### Internal Comments

- Use `// TODO:` for missing features or improvements. Messages are usually lowercase
- Comments should provide educational value or context about the code. They should help others learn the architecture, and keep code maintainable. Use them to explain complex or obscure logic, not to mention every step (something AI loves to do).
- They should be easy to understand (in terms of reading level). Avoid high-level words unless they're related to tech.
- Abbreviations (e.g. AST, expr, stdlib) are allowed, but for exports or long comments, use them less. Be aware of the background of developers working on this project; use abbreviations most of them would understand.
- Comments usually don't end in periods, unless they're long or multiple sentences.
- Quote code using backticks or single quotes.
- Indent Klar code snippets similar to Go doc comments.

## Go Language Features

Always make use of the latest Go syntax and standard library features. Consult the `go.mod` at the root of the project for the latest version.

- Use **`any`** instead of `interface{}` (never allowed except for stub interfaces).
- Use **`range`** and iterators, including custom iterators (great examples include [Checker.followDestructure](../../internal/analysis/destructure.go) and [Tokenizer.Tokenize](../../internal/lexer/lexer.go)) to avoid creating slices solely for iteration.
- Use **`min()`**, **`max()`**, and **`cmp.Or()`** where appropriate.
- Use **`new(expr)`** for allocating and initializing pointers.
- Modernize whenever your LSP suggests it.
- Use `(*testing.B).Loop()` instead of `b.N`.
- Favor `slices` and `maps` packages over manual implementations.
- Use `strings.Builder` for efficient string concatenation. Never concatenate to a string in a loop.
- **ProTip:** For sorting keys in a map, use the following one-liner:

```go
keys := slices.Sorted(maps.Keys(m)) // OMG WOW!!!
```

## Testing & Benchmarking

- Use `t.Run` to group subtests.

## Examples

Here are some examples in the codebase that well-illustrate our Go conventions. You may read a few of the files below to get a glimpse of what we want when writing Go.

- [internal/build/static_parser.go](../../internal/build/static_parser.go)
- [internal/lexer/tokenizers.go](../../internal/lexer/tokenizers.go)
- [internal/module/resolve.go](../../internal/module/resolve.go)
- [internal/build/run.go](../../internal/build/run.go)
- [internal/analysis/declaration.go](../../internal/analysis/declaration.go)
- [internal/parser/parse.go](../../internal/parser/parse.go)
- [pkg/klarerrors/reporter/syntax.go](../../pkg/klarerrors/reporter/syntax.go)
- [internal/ranges/ranges.go](../../internal/ranges/ranges.go)
