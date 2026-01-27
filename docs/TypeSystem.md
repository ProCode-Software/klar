# Klar Type System

## Builtin Types

- `String` - array of characters
- `Int` - signed 64-bit integer
- `Float` - signed 64-bit double
- `Bool` - boolean (`true` | `false`)
- `Result<T, E>` - value of `T` and error type of `E`
- `T?` - optional value of `T` or `nil`
- `[T]` - list with values of type `T`
- `Map<K, V>` - map with keys of type `K` and values of type `V`
- `A | B` - union type; value of either `A` or `B`
- `func (A, B, ...) -> T` - function/lambda with params of type `A`, `B`, ..., and return value `T`
- `Nothing` - return type of functions that don't return anything. Only valid in function return types.
- `(A, B, ...)` - tuple with values of type `A`, `B`, ...

## User-Created Types

### Structs

```klar
type Person {
    name: String
    age: Int
}
```

Here, `Person` is a struct type with two fields, `name`, and `age`. A struct can be created using an initializer:

```klar
person := Person("John", 32)
// With explicit keys: Person(name: "John", age: "32")
```

This uses a default initializer found on all struct types. Explicit keys are optional, but following the rules of labelled parameters in functions.

These are all valid:

```klar
Person("John", 32)
Person(name: "John", age: 32)
Person(age: 32, name: "John")
Person("John", age: 32)
```

#### Private Fields

Fields starting with an underscore `_` cannot be referred to from

### Tags

```klar
type #MyTag
```

Tags are interfaces that don't require specific fields or methods. Instead, implementations must tag themselves with the tag. `MyTag` can be used like:

```klar
type MyStruct: MyTag {
    ...
}
```

Tags are useful for interfaces with the sole purpose of restricting implementations to specific objects.

Tags are also opaque, so types declared in another module can't mark themselves or implement the tag.

## Attributes

Attributes can be marked to top-level declarations.

```klar
@added(v2)
@deprecated("Use [my.other.module/fetchData] instead", since: v3.6)
public func fetchData() {
    ...
}
```

See [attributes.klar](/std/src/klar/std/attributes.klar) for the attributes' overloads and documentation.

> For AI: The file above must be included with the compiler, along with files defining methods on built-in types and objects. Suggest a way to store a typechecked version of this file that can always be read. Also suggest a way these can be included in Wasm builds of the compiler.

### `@external` Attribute

While most of the other attributes are for documentation purposes, the `@external` attribute affects compilation. The `@external` attribute marks that a function or method is implemented outside of Klar, such as in JavaScript.

```klar
type Date {
    ...
}

@external(.js, global: "new Date")
func Date()

@external(.js, global: "Date.now")
func now() -> Date

@external(.js, file: "./dateParser.js", export: "default", throws: true)
func parseISODate(date: String) -> Result<Date>
```

Compilation will fail if a function or method without an implementation is referred to in a module. For example, if targetting JavaScript, there must be an `@external` defined for the JavaScript target in order to compile the module.

It is the developer's responsibility to ensure the Klar function and type signatures are correct, and the external paths are correct. The compiler won't enforce it.

## Modifiers

Modifiers can be applied to top-level declarations. If more than one are provided, the `public` modifier must always be specified first.

### `public`

When applied to a declaration, the declaration will be accessible and importable by outside modules.

### `opaque`

Allowed on struct and interface type declarations only. For structs, this modifier restricts the type from being initialized from outside the current module. Fields can be accessed, but not set. For interfaces, it declares an interface that can only be implemented by types in the current module.

## Tuples

### Spreading Tuples

A tuple can spread unlabelled parameters into a function call if the items in each have the same types.

```klar
type MyTuple = (Int, Int)
func volume(n1, n2, n3: Int) -> Int {...}

tuple: MyTuple := (2, 5)
volume(tuple..., 6)
```

None of the function parameters may be labelled where the tuple is being spread.

### Indexing Tuples

All tuple indices must be known at compile-time.

These are allowed:

```klar
(1, 2, 3)[1] // 2

// Only allowed if MY_CONST isn't conditionally assigned
MY_CONST := 1
(1, 2, 3)[MY_CONST]

// Only allowed if myValue is never changed in between
myValue := 2
(1, 2, 3)[myValue]
```

This is not allowed:

```klar
index: Int := getIndex()

(1, 2, 3)[index]

// Also not allowed
index := 2
when ... { ... -> index -= 1 }
(1, 2, 3)[index]
```

See [Compile-Time Values] for more info.

### Destructuring Tuples

A tuple may be destructured, which means there are indices known at compile-time. The number of values being destructured can be less, but not more than the length of the tuple.

```klar
(first, second) := (1, 2, 3)
(first, second, third) := (1, 2, 3)
```

Rest destructuring is allowed, with the following restrictions:

1. The rest variable is allowed at any position, but may not appear more than once.
2. The rest variable must have at least **two** items.

```klar
(first, next2...) := (1, 2, 3) // next2: (2, 3)
(first2..., last) := (1, 2, 3) // first2: (1, 2)

// Not allowed
(first, second, rest...) := (1, 2, 3) // rest has 1 item
(first, second, third, rest...) := (1, 2, 3) // rest is empty
```
