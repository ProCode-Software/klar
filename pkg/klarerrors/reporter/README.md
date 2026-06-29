# Klar Error Reporter

## Examples

Examples are also available in [sample.txt](./sample.txt), and [hints.txt](./hints.txt) for output containing hints.

1. Reported from [`internal/analysis/function.go`]()

```
Type error: This function is supposed to return 'V', but the body contains no 'return' statements (missingReturn)

    ┌─ std/src/klar/_builtin/map.klar:130:1
121 │ // Returns the value for the given key, setting the key to `default` if it doesn't exist.
122 │ public func Map.getOrSet(key: K, default default: V) -> V {
    ·                                                         ^ This function is supposed to return 'V'
123 │     when key in self {
--- ├ ···
130 │ }
    · ^ No 'return' statements in the body
```

2. Reported from [`internal/analysis/function.go`]()

```
Type error: Type mismatch: expected type '#{K: V}', but this has type 'T' (typeMismatch)

    ┌─ std/src/klar/_builtin/map.klar:154:12
146 │ // Returns a new map without the keys that have the same value as those in the provided map.
147 │ public func Map.without(valuesFrom map: #{K: V}) -> #{K: V} {
    ·                                                     ━━━━━━━ The function is supposed to return '#{K: V}'
148 │     result: #{K: V} := clone(self)
--- ├ ···
154 │     return result
    ·            ━━━━━━ The returned value has type 'T'
```

3. Reported from [`internal/analysis/literal.go`]()

````
Type error: I can't determine the item type of this empty list (untypedEmptyList)

   ┌─ std/src/klar/_builtin/Int.klar:47:50
45 │ ```
46 │ */
47 │ public func Int.expandedForm() -> [Int] { return [] }
   ·                                                  ━━ This list is empty and its type can't be inferred

Hint: If you're declaring a variable, add a type annotation before ':='.

Hint: Otherwise, initialize an empty list with a specific type. (Replace 'T' with the intended item type)

    47 │ public func Int.expandedForm() -> [Int] { return [T]([]) }
       ·                                                  ++++  +
````

## API

## Architecture
