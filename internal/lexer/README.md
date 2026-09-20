# Klar Lexer

The lexer reads a stream of characters from a Klar file and converts sequences of characters to tokens. It is the first step in the parsing, and ultimately, the compilation, of a Klar file.

## `Token`

The [`Token`](./token.go) defines a lexer token.

```go
type Token struct {
	Position
	Kind       TokenType
	Source     string
	Attributes map[string]any
}
```

The `Attributes` field stores information about the token based on its type, such as the number format used if the token is a number.

## Token Types

### Keywords

### Operators

## How the lexer works

## `Tokenizer` type
