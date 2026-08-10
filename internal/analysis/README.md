# Klar Analysis / Type Checker

## Grammar

In [compatibility.go](./compatibility.go), you'll see syntax in comments that describe typing properties.

- `A => B` means type _A_ is assignable to _B_.
- Uppercase letters are type variables. `A => A` refers to the same type on both sides.
- Lowercase letters are variables (not types). Types can be assigned to them using the syntax `a: T`, meaning variable _a_ has type _T_.
- Klar type syntax is also used, so `[A]` is a list, and `A | B` is a union type.
- `if`, `and` and `or` are also used. `A => B | C if A => B or A => C` means type _A_ is compatible with the union of _B_ and _C_, if _A_ is assignable to either _B_ or _C_.
- `A =!> B` means type _A_ is not assignable to _B_.
- `A = B` means type _A_ is equal to _B_ (similar meaning for `!=`).
- `A <=> B` means type _A_ is assignable to _B_ and vice versa.

## Overload Resolution

### Overload Declaration Rules

#### Variadics

Either all or no overloads must have variadic positional parameters.

<table>
<tr><td>Not allowed</td><td>Not allowed</td></tr>
<tr><td>

```klar
func(...Int)
func(Int, Int)
```

</td><td>

```klar
func(String, ...Int)
func(...String | Int)
func(String, Int)
```

</td></tr>
</table>

#### Compatibility

<table>
<tr><td>Allowed (interface/tag)</td><td>Not allowed (union)</td></tr>
<tr><td>

```klar
type #Vehicle
type Car {}

func(Vehicle)
func(Car)
```

</td><td>

```klar
func(A | B)
func(A)
```

</td></tr>
</table>

#### Optionals

<table>
<tr><td>Not allowed</td><td>Not allowed</td></tr>
<tr><td>

```klar
func(Int?)
func(Int)
```

</td><td>

```klar
func(String, Int, Bool)
func(String, Int?, Bool)
```

</td></tr>
</table>
