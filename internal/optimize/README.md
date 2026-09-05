# Klar Optimizer

This package handles Klar-level AST optimizations before JavaScript codegen.

Some or all optimizations may be disabled, depending on compiler settings.

- When compiling Klar for `klar run` or `klar test`, little to no optimizations will be performed. Compilation will be faster, at the expense of runtime performance.
- For `klar build`, the full set of optimizations will be enabled, so production code runs faster, but this means compilation will take longer.

## Optimizations performed by the Klar compiler

### Dead Code Elimination

Declarations that aren't referenced in the code are removed. If they have side effects, such as function calls, they will need to be preserved.

The type checker emits warnings for unused declarations. Many of them can get removed, but not all of them. And, some declarations may be eliminated during optimization even if there was no warning during analysis.

```klar
// Module klar.os
public func openFile(path: String) -> Result<File>

public func readFile(path: String) -> Result<String> {
    file := try openFile(path)
    // ...
}

public func delete(path: String) -> Result

// Current module
content := try os.readFile("./words.txt")
```

In this example, The only object from `os` the module references is `readFile`. `readFile` depends on `openFile`, so that will also be included. `os.delete` will be eliminated.

#### Tree-Shaking Module Values

Modules are always tree-shaken so only the exports referenced by the program are included. If we have:

```klar
import klar.os

args := os.args
```

Only `args` from `os` is included, along with its dependencies.

But also, in Klar, imported objects are first-class values. So you can `import klar.text` and then use `text` as a value outside of a selector. Modules can implement interfaces, so these are optimized. Exports that aren't needed as interface fields will be eliminated.

There are several downsides to this, such as how tree-shaking won't occur when the module is assigned as a map or to `Any`. It is also possible for code to downcast. Modules don't have an actual type that can be written in a type annotation, so many downcasts are out of scope. However, interfaces can be downcast to other interfaces, and tree-shaking without enough context can break behavior.

```klar
import klar.os // Provides open() and write()

type #FileSystem {
    open(String) -> Result<File>
}
type #WritableFileSystem: FileSystem {
    write(String, to: String) -> Result
}

fs: FileSystem := os
when fs {
    // Depending on how optimization is implemented, this may not be true,
    // despite the `os` package having a `write()` implementation
    WritableFileSystem -> {}
    _ -> {}
}
```

It is important to note that `klar run` may not optimize, but `klar build` does, so this creates a discrepancy where behaviour during development could be different from production. To avoid this, the entire module may have to be included, without tree-shaking, which is extremely inefficient.

### Constant Folding

The type checker can analyze a lot of constants. Many things are already evaluated during analysis.

Before:

```klar
when msg.starts(with: "Hello ") {
    true -> return msg["Hello ".length...]
    _ -> return msg
}
```

After:

```klar
when msg.starts(with: "Hello ") {
    true -> return msg[6...]
    _ -> return msg
}
```

### Inlining

Small (short) function declarations are removed, and the function's body is copied wherever the original function was used.

Before:

```klar
func sum(a, b: Int) -> Int = a + b

sum1 := sum(1, 1)
for i in 5 {
    sum(i, 2 * i) |> print
}
```

After:

```klar
sum1 := 1 + 1 // Later optimized to just '2'
for i in 5 {
    i + 2 * i |> print
}
```

Very short functions like the one above may always be inlined, but the optimizer tries to balance the size of the function being inlined and how often it is used. Functions that are longer that the one above, but still considered short enough to be inlined, may not be inlined if it is called several times. This is so file sizes aren't drastically increased.
