# Klar Pattern Matching

## Usage

```klar
when subject {
    pattern1 -> ...
    pattern2 -> ...
    // More patterns...
}
```

## Literal Expressions

```klar
num := 3
expected := 4
when num {
    2 -> {}
    expected -> {}
    // Same
    == expected -> {}
}
```

## Comparisons

Numbers can be matched to a range. Patterns may also start with a comparison operator, without a left-hand side. The left-hand side is inferred as the subject of the 'when'. This is allowed for all relational operators (`==`, `!=`, `<`, `<=`, `>`, `>=`, `in`, `!in`) and the `and`/`or` operators.

```klar
value := 10
otherValue := 15
luckyNums := [2, 3, 5, 9, 39, 27]
when value {
    < 5 -> {}
    // value <= otherValue <= 25
    <= otherValue <= 25
    != 12 -> {}
    // Check if value is in a range
    0...10 -> {}
    in luckyNums -> {}
}
```

## Type Matching

Allowed for structs, interfaces, tags, and unions.

```klar
type Person {
    name: String
    age: Int
    hobbies: [String]
}

any: Any := Person('John', 32, ['programming', 'soccer'])
when any {
    // Matches if 'any' has type 'Person'
    Person -> {}
    // Matches if 'any' has type 'Person' where 'name' and 'age' are
    // 'John' and '32'. Remaining fields can be any value. (Use `...`
    // instead of '_' if there are more remaining fields. Optional and
    // fields with default values don't have to be specified).
    Person(name: 'John', age: 32, _) -> {}
    // Equivalent, but without labels
    Person('John', 32, _) -> {}
    // Unwrap 'name' and 'age' as variables
    Person(name, age, _) -> {}
    // Match a Person named 'John', unwrapping the hobbies
    Person('John', _, hobbies) -> {}
    // Match a Person with 'soccer' in hobbies
    Person(:hobbies, ...) if 'soccer' in hobbies -> {}
}
```

### Structs

```klar
jane := Person('Jane', 31)
when jane {
    // Shorthands are allowed (':' is required when using labelled fields)
    .(name: 'Jane', :age) -> print("Jane has age {age}")
}
```

## Strings

```klar
msg := 'John has 3 cats and 1 dog.'
when msg {
    // Exact match
    'John has 3 cats and 1 dog.' -> {}
    // Match any string that starts with 'John'
    'John '... -> {}
    // Match any string that ends in 'dog(s)'
    ...'dog.' | ...'dogs.' -> {}
    // Match any string that contains '3'
    ...'3'... -> {}
    // Match any string that contains 3 cats or dogs
    ..."3 {animal}" if animal == 'cats' or 'dogs' -> {}
    // Equivalent
    ..."3 {'cats' | 'dogs' as animal}" -> {}
    // Match the name of the person if they have 3 cats and 1 dog
    "{person} has 3 cats and 1 dog." -> {}
    // Equivalent
    person + ' has 3 cats and 1 dog.' -> {}
    // Match the string with any number of dogs or cats, matching if they are
    // valid numbers, and unwrapping them as variables. Note that this is
    // unpluralized, and negative numbers could match.
    "John has {cats: Int} cats and {dogs: Int} dogs." -> print(dogs, cats)
    // Match a string that ends with 1 or more dogs
    ..."{count: Int} {'dog' | 'dogs'}" -> {}
}

word := 'hello'
when word[0] {
    // Matches if the string is a single upppercase letter
    'A'...'Z' -> {}
    // Lowercase letter
    'a'...'z' -> {}
    // Matches if the string is empty
    none -> {}
    // Matches a non-nil string that is more than 1 character or is some other character
    _ -> {}
}

when word {
    // Matches any string that begins with a lowercase letter
    ('a'...'z')... -> {}
}
```

## Enums

```klar
type MyEnum {
    .noParams
    .withParams(a, b: Int)
    .withParams3(a: Int, b: String, c: Bool)
}

value := MyEnum.noParams
when value {
    .noParams -> {}
    // Check if the variant matches, disregarding parameters
    .withParams -> {}
    // Equivalent
    .withParams(...) -> {}
    // Check if the variant matches and its parameters
    .withParams(1, 2) -> {}
    // Check if the variant matches, and unwrap its params
    .withParams(a, b) -> print(a, b)
    // Match 1 of its params, and unwrap the others
    .withParams3(2, b, c) -> print(b, c)
    // Equivalent, but with labels
    .withParams3(a: 2, :b, :c) -> print(b, c)
    // Match the first param, with any value for the next 2
    .withParams3(2, _, _) -> {}
    // Equivalent
    // Note: '_' must be used instead of '...' if only 1 item is discarded
    .withParams3(5, ...) -> {}
    // Common items can be indexed
    .withParams | .withParams3 -> print(value.a)
    // `withParams` will contain only common parameters (`a` and `b`)
    .withParams | .withParams3 as withParams -> print(withParams)
    // Common items can also be unwrapped. If 'c' was used over '_', there
    // would be an error.
    .withParams(a, b) | .withParams(a, b, _) -> print(a, b)
}
```

## Lists & Tuples

```klar
list := [1, 2, 3, 4, 5, 6]

when list {
    // Matches if the list only contains `1`
    [1] -> {}
    // Matches if the list starts with 1, and contains 1 or more items
    [1, ...] -> {}
    // Matches if the list is empty
    [] -> {}
    // Matches if the list contains any amount of items. Useful for matching on,
    // non-concrete types, which will match any list, but when matching on a
    // concrete list, this is an error.
    [...] -> {}
    // Matches if the list contains exactly 2 items, with the second being `2`.
    // The first item is unwrappped as a variable.
    [first, 2] -> print(first) // 1
    // Similar to above, but the list can contain more items after `2`.
    [first, 2, ...] -> print(first)
    // Similar to above, but the unwrappped item is discarded. This essentially
    // matches a list where the second item is `2`.
    [_, 2, ...] -> {}
    // Matches a list with at least 3 items, starting with `1`, `2`, and `3`. The
    // remaining items are unwrapped into a new list `rest` (may be empty).
    [1, 2, 3, rest...] -> print(rest) // [4, 5, 6]
    // Matches a list with 6 items, ending with `5` and `6`. The items before are
    // unwrapped into a new list `rest`, with a length of 4.
    [rest..., 5, 6] if rest.length == 4 -> print(rest) // [1, 2, 3, 4]
    // Equivalent, and more idiomatic
    [[_, _, _, _] as rest, 5, 6] -> print(rest) // [1, 2, 3, 4]
    // Matches a list that contains `3`
    [..., 3, ...] -> {}
    // Matches a list that contains `3`, unwrapping the parts before and after it.
    [before..., 3, ...after] -> print(before, after) // [1, 2], [4, 5, 6]
}
```

Tuples have similar matching syntax, but more strict about length. `...` can't be used where 1 item or less will be unwrappped.

```klar
tuple := ('a', 'b', 3)
when tuple {
    // Neither are valid, because `rest` unwraps 1 item and 0 items respectively
    (_, _, rest...) | (_, _, _, rest...) -> {}
    // Invalid: Length mismatch
    ('a', 'b', 3, 4) -> {}
    // Invalid for the same reason
    (_) -> {}
    // This is valid when the subject isn't a concrete tuple. This matches any tuple
    (...) -> {}
    // This is allowed when the subject isn't a concrete type. `rest` will be a list of `Any`
    (rest...) -> {}
    // Compile-time error when the subject is a concrete tuple, because it always matches
    (_, ...) -> {}
    // Invalid when the subject is a concrete tuple, without a guard. You could
    // just assign (via `(first) := tuple`)
    (first, ...) -> {}
    // Invalid the same way
    (_, rest...) -> {}
    // Valid
    (1, ...) -> {}
    // Also valid
    (2, rest...) -> {}
}
```

## Maps

```klar
map := #{a: 1, b: 2, c: 3}
when map {
    // True if `a` has a value of `1` and is the only key. For `map`, this doesn't match
    #{a: 1} -> {}
    // True if `a` exists and unwraps it value. Other keys may exist
    #{:a, ...} -> print(a)
    // Same as above, but remaining keys are unwrapped into a new map `rest`.
    // It may be empty.
    #{:a, rest...} -> print(a)
    // True if keys `a` and `b` are the only keys, with any value. Those
    // variables are unwrapped.
    #{:a, :b} -> print(a, b)
    // True if `a` and `b` are both `2`.
    #{a, b: 2} -> {}
    // Matches if `a` exists, unwrapping its value into the variable `a`
    #{:a as a, ...} -> print(a)
    // Equivalent
    #{:a, ...} -> {}
    // True if `a` and `b` have the same value
    #{a, b: _} -> {}
    // True if `a` exists with any value, and `b` exists with a value of `2`
    #{a: _, b: 2} -> {}
    // True if `a` exists and has a value of `1`
    #{a: 1, ...} -> {}
    // Matches any map. Useful for interfaces, but for a concrete map, this
    // is a compile error.
    #{...} -> {}
    // Not allowed. In the future, this could match if the map has any
    // key with a value of `2`. (`#{_ as key: 2}` could also be allowed)
    #{_: 2} -> {}
}
```

## Bypassing Pattern Matching

For several types including strings, maps, and lists, places where an identifier is used unwraps a value rather than matching it to an existing variable.

```klar
expected := 3
list := [2, 4]
when list {
    // Unwraps `list[1]` into the variable `expected`, doesn't check if the
    // list equals `[2, 3]`. So this always matches any list with 2 items.
    [2, expected] -> {}
}
```

To evaluate variables instead (e.g. `list == [2, 3]` from the code above), the most
idiomatic way is to include `==` at the beginning of the case, without the left-hand
side. The left is inferred as the subject of the when:

```klar
when list {
    // Matches a list that equals [2, 3]
    == [2, expected] -> {}
}
```

This is also similar for `!=`.

Or, you can wrap the case expression in parentheses. If the whole case expression is in parentheses, pattern matching is disabled, and a literal expression will be matched for equality.

```klar
when list {
    // Matches a list that equals [2, 3]
    ([2, expected]) -> {}
}
```
