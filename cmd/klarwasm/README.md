# Klar WebAssembly Compiler

When run, the `Klar` namespace will be available in `globalThis`.

```ts
namespace Klar {
    /**
     * Compiles a Klar script with the given source. The file name is
     * used in error messages.
     */
    function compile(src, fileName: string): CompileResult
    /**
     * Frees resources held by Go. This should be called when the page is
     * unloaded to free resources held by Go.
     */
    function freeCompiler(): void

    interface CompileResult {
        /** The generated JS code */
        output: string
        /**
         * Returns the errors in the format displayed in the CLI. The returned
         * string will contain ANSI escapes. This function can only be called
         * once per result.
         */
        getErrors: () => string
    }
}
```
