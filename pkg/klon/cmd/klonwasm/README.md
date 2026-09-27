# klonwasm

A WebAssembly port of [Klon](../../README.md). Used in the [`@klarlang/klon` NPM package](#) <!-- https://npmx.dev/package/@klarlang/klon -->

## Build

Build with [TinyGo](https://tinygo.org):

```sh
GOOS=js GOARCH=wasm tinygo build -o "$DIR/../lib/klon.wasm" --no-debug .
```
