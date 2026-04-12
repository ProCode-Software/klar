# Klar Type System

## Builtin Types

- `String` - array of characters
- `Int` - signed 64-bit integer
- `Float` - signed 64-bit double
- `Bool` - boolean (`true` | `false`)
- [`Result<T, E>`](#results) - value of `T` and error type of `E`
- `T?` - [optional](#optionals) value of `T` or `nil`
- `[T]` - list with values of type `T`
- `#{K: V}` - map with keys of type `K` and values of type `V`
- `A | B` - [union](#unions) type; value of either `A` or `B`
- [`func (A, B, ...) -> T`](#functions) - function or lambda with params of type `A`, `B`, ..., and return value `T`
- `Nothing` - return type of functions that don't return anything. Only valid in function return types and not as a value.
- [`(A, B, ...)`](#tuples) - tuple with values of type `A`, `B`. Tuples can contain any amount of items (including zero)

## User-Created Types

### Structs

```klar
type Person {
    name: String
    age: Int
}
```

Here, `Person` is a struct type with two fields, `name`, and `age`.

#### Private Fields

Fields and methods starting with an underscore `_` cannot be referred to from outside modules. All other
fields are public.

#### Optional Fields

```klar
type Person {
    name: String
    age: Int?
}
```

If a field has an optional type, it may be omitted in its initializer.

```klar
john := Person("John")
```

#### Fields with Default Values

Fields with default values may also be omitted in initializers.

```klar
type Person {
    name: String
    age: Int
    hobbies: [String] = []
}

jane := Person("Jane", 31)
```

A default value may reference a struct field via the `self` object. A default value may not contain a function call, though they may contain initializers for other types.

#### Methods

Methods are declared with a similar syntax to functions, but with a receiver/self type before the name
of the method.

```klar
type Person {...}
func Person.greeting() -> String {}
```

Methods must be declared at the **same** level as the type (not nested). A compile-time error
is raised if the method is declared in a higher level than the receiver type, or in a different
module. You can only declare a method on a locally-declared type.

Like fields, methods starting with an underscore `_` are private to the module they are declared in.

A method's scope has access to the receiver via the `self` variable.

```klar
func Person.greeting() = "Hi, my name is {self.name}!"

// `self` can also be accessed in default values
func Person.greeting(nickname: String = self.name) = "Hi, my name is {nickname}!"

// Even in field declarations!
type Person {
    name: String
    nickname: String = self.name
}
```

Methods can't be mutated:

```klar
func Person.foo() { self.greeting = func {...} } // Error
```

#### Casting Structs

Suppose we have two structs with a common field:

```klar
type A {
    x: Int
}

type B {
    x: Int
    y: Int
}
```

Klar allows casting `B` to `A` via a default initializer.

```klar
b := B(1, 2)
a := A(b)
```

Because `A` lacks one of `B`'s fields, type `A` cannot be cast to `B`. However, f `B.y` was
an optional `Int?` or had a default value, casting from `A` would be allowed.

### Interfaces

```klar
type #Readable {
    read(n: Int?) -> Result<String>
    offset: Int
}
```

Interfaces are similar to [union types](#unions), but explicitly define common fields and methods.

Interface method definitions are allowed to use the `Nothing` type in a union for return types:

```klar
type #Writable {
    write(String) -> Result<Int> | Nothing
}
```

```klar
writer: Writable := myWritable

when writer.write("Hello, World!") {
    Nothing -> {...}
    Error -> {...}
    Int as n -> { print("Bytes written: {n}") }
}

```

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

Now, `MyStruct` is assignable to `MyTag`:

```klar
obj: MyTag := MyStruct(...)
```

Tags are useful for interfaces with the sole purpose of restricting implementations to specific objects.

Tags are also implicitly opaque, so types declared in another module can't mark themselves or implement the tag.
Explicitly applying the `opaque` modifier to a tag is a compile-time error.

Because tags don't require implementations to have specific fields or methods, indexing a tag is not
allowed. In order to reference any of `MyStruct`'s fields or references, `obj` must be explicitly downcasted.

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

## Tuples

A tuple groups two or more values of any type into a single value. The length of the tuple and the types for each item are annotated with a tuple.

```klar
(1, 2, "hello") // Inferred as (Int, Int, String)
x: (Int, Bool) := (2, true)
type RGB = (r, g, b: Int)
color: RGB := (255, 42, 30)
```

### Spreading Tuples

A tuple can spread unlabelled parameters into a function call if the items in each have corresponding types.

```klar
type MyTuple = (Int, Int)
func volume(n1, n2, n3: Int) -> Int {...}

tuple: MyTuple := (2, 5)
volume(tuple..., 6)
```

None of the function parameters may be labelled where the tuple is being spread.

A tuple may also be spread into lists or variadic functions:

```klar
_ = [(1, 2, 3)...]
_: [Int | String] = [(1, "hello", 2, "world")...]
```

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

## Functions

Functions may take parameters to execute statements and/or return values.

A function may have zero or more parameters. Each parameter must have an explicit type annotation.

Statements may be executed with the context of parameters. A function may return a value using `return` statements. The return type of a function must be explicitly labelled after `->`, unless the function returns `Nothing`.

```klar
func add(n1: Int, n2: Int) -> Int {
    return n1 + n2
}
```

If multiple parameters in a row have the same type, you may provide the type only after the last parameter.

```klar
func add(n1, n2: Int) -> Int
```

Or, if a function only returns an expression, that can be provided after `=`. If `=` is used for the function body, an explicit return type is not required. The expression is allowed to return `Nothing`.

```klar
func add(n1, n2: Int) = n1 + n2
```

Function parameter and return types always stay the same, however, they may be set to a union.

All functions must have bodies, either using `=` or in curly braces, unless the function is annotated as [external]().

Functions with generics or parameters that are labelled or have defaults cannot be type-annotated.

### Nothing
The `Nothing` type indicates that the function doesn't return any value. `return` statements don't have expressions after them or are not present in the function body.

The `Nothing` type cannot be used in any union type (except in interface method return types).

### Labelled Parameters

A labelled parameter is declared with another name before the name of the parameter/variable used internally.

```klar
func String.replace(substr: String, with rep: String) -> String
```

The second parameter requires the label `with`, and `rep` is the name of the variable used by the function. It would be called like:

```klar
"Hello, World!".replace("World", with: "John")
```

The label is always required when assigning the parameter. If a function signature declares any non-optional, labelled parameters, the function cannot be assigned to a variable, as lambdas can't declare parameter labels.

Using the example method above, this is not allowed:

```klar
replace := "Hello, World!".replace
```

The `replace` variable must wrap the function:

```klar
replace := func(old, new: String)
    -> "Hello, World!".replace(old, with: new)
```

### Optional Parameters

A parameter may be omitted from a function call if:

- The parameter is an optional type (e.g. `Bool?`), or
- The parameter has a default value

And:

- The parameter isn't followed by any non-optional parameters, or
- The parameter has a label

```klar
func greet(person: Bool, formally formal: Bool = false)

greet("John") // formal = false
greet("John", formally: false) // Same as above
```

### Variadic Parameters
Variadic parameters are syntax sugar for lists. A variadic parameter has the list type, but its items can be provided outside a list literal.
Only the last parameter or a labelled parameter can be variadic. Zero or more items may be passed to a variadic parameter.

```klar
func print(items: ...Any)

print(1, 2, 3)
print([1, 2, 3]...)
print()
```

### Parameters with Default Values

### Generics
Functions may declare generics, which are type parameters. One or more generics can be defined in angle brackets, and used in parameter types and the return type.

```klar
func first<T>(of list: [T]) -> T
```

All generics are inferred, along with comparible operations. Each declared generic must be used in function parameters.

### Function Overloads
Functions may have more than one set of parameters, as long as they return the same parameter types.

<!-- TODO: Should we allow Results of the same type? -->

```klar
func greet(person: Person) -> String
func greet(name: String) -> String

greet("John")
greet(Person("Jane", 31))
```

To avoid ambiguity, there are some restrictions for overloads with closely related parameters:

1. Variadic vs. positional params of compatible types
```klar
func String(charCodes: ...Int)
func String(charCode: Int)

String(67) // Which overload is this referring to?
``` 

2. Unions
```klar
func sum(n1, n2: Int) -> Float
func sum(n1, n2: Int | Float) -> Float

sum(2.0) // Overload #2
sum(2) // Which overload?
```

3. Optionals
```klar
func sum(n1, n2: Int) -> Int
func sum(n1, n2: Int?) -> Int

sum(nil, nil) // Overload #2
sum(5, 8) // Which overload?
```

4. Overloads with exact or compatible parameters
```klar
type #Readable {}
type File: Readable {}

func readAmount(reader: Readable, n: Int) -> Result<String>
func readAmount(reader: File, n: Int) -> Result<String>
```

## Initializers and Type Casts

### Struct Initializers
A struct can be created using an initializer:

```klar
person := Person("John", 32)

// With explicit keys
Person(name: "John", age: "32")
```

This uses a default initializer found on all struct types. Explicit keys are optional, but following the rules of labelled parameters in functions.

These are all valid:

```klar
Person("John", 32)
Person(name: "John", age: 32)
Person(age: 32, name: "John")
Person("John", age: 32)
```

All unlabelled parameters must be in order, but labelled parameters can be in any order. All fields that don't have default values or optional types must be present in the initializer. With unlabelled parameters, fields
with default values or optional types must be provided if they precede required fields. If they are the
last fields, they can be skipped.

#### Inferred Initializers
Similar to [enums], if the expected type is known (a concrete type), the type's name can be substituted with a dot.

```klar
person: Person := .("John", 32)
```

### Cast Initializers
Type casts are initializers that take a single, unlabelled value to convert it to the target type.

All structs can be initialized using a type cast that takes a compatible value:

```klar
type A { x, y: Int }
type B { x: Int }

a := A(1, 2)
b := B(a)
```



### Primitive Initializers


### Custom Initializers

## Optionals

An optional is composed of an underlying type, but its value could be `nil`. For safety, optionals cannot be indexed and their underlying values cannot be used, without unwrapping the optional.

Akin to `Result`s, optionals may be unwrapped using a `when` expression that specifically handles its `nil` value, or using an assertion (`!!`), which crashes the program if the value if `nil`.

```klar
maybeInt: Int? := 2
print(maybeInt + 1) // Error: 'maybeInt' needs to be unwrapped

// These are allowed
print(maybeInt!! + 1) // Beware: Crashes the program if `maybeInt == nil`

when maybeInt {
    nil -> doSomethingElse()

    // The compiler knows that maybeInt isn't nil; it has been unwrapped.
    _ -> print(maybeInt + 1)
    // This may be used instead, but it is not idiomatic:
    Int -> print(maybeInt + 1)
}
```

## Unions

```klar
type A { x, y: Int }
type B { x: Int }

type Union = A | B
```

A union type can be two or more types. They may only access common fields, methods, and operations. In the example above, `Union` can access field `x: Int` because all types in the union have the common field.

## Results

`Result<T, E>` is a union between `T` and error type `E`.

## Assertions

The assert operator (`!!`) may be used with `Result` and [optional](#optionals) values to crash the program if the value is an error or `nil` respectively.

Assertions should be used carefully, especially in production. It is more useful to show a better error message to the user or handle the special case. Klar has options for the typechecker to restrict the use of assertions to avoid hidden program crashes.

## Operators

|         Name          |                Operator                 |                        Types                        |
| :-------------------: | :-------------------------------------: | :-------------------------------------------------: |
|       Addition        |                   `+`                   |         `Int`, `Float`, `String`, map, list         |
|      Arithmetic       | `-` (infix/postfix), `*`, `/`, `^`, `%` |                   `Int`, `Float`                    |
| String multiplication |                   `*`                   |            `String * Int` (exact order)             |
|       Assertion       |                  `!!`                   |                 `Result`, optional                  |
|          Try          |                  `try`                  |                      `Result`                       |
|          In           |               `in`, `!in`               |             `K in #{K, V}`, `T in [T]`              |
|         Range         |          `...` (infix), `..<`           | `Int`, `Float`, `String` (single A-Z/0-9 character) |
|         Rest          |             `...` (postfix)             |                      list, map                      |
