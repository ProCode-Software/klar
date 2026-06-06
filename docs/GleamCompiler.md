# ⭐🩷 Gleam Compiler Notes (for Project Compilation)

1. Dependencies from manifests are installed
2. Those deps are compiled first
    1. Deps are toposorted
    2. Deps are loaded from cache, or compiled using the build tool listed in the manifest `manifest.toml` (for Hex)
    3. Calls `compile_gleam_dep_package`
        1. The package's root is resolved and its `gleam.toml` is parsed
        2. Calls normal `compile_gleam_package`, returning its modules
3. **Warnings** from those compilation are reset. Checks for warnings as errors
4. Root package is compiled
    - `compile_gleam_package` with `is_root = true`
    - Return value of type `Compiled` is cast to `Package`
5. Warnings as errors are shown
6. Build complete!

## `fn compile_gleam_package`

- [compiler-core/src/build/project_compiler.rs](https://github.com/gleam-lang/gleam/blob/874b0bb616a9111d2d34dd79b4ee8763c9767c5b/compiler-core/src/build/project_compiler.rs#L581)

```rust
fn compile_gleam_package(
    &mut self,
    config: &PackageConfig,
    is_root: bool,
    root_path: Utf8PathBuf,
) -> Outcome<Compiled, Error>
```

For compiling deps, `is_root = false`

1. Out path is resolved
2. For JS target, makes a config whether to emit dts files & sourcemaps
3. `PackageCompiler` initialized
4. Whether to compile modules
    ```rust
   compiler.compile_modules = !(self.options.compile == Compile::DepsOnly && is_root);
   // compiler.compile_modules = self.options.compile != Compile::DepsOnly || !is_root;
    ```
5. Enforce target support
    ```rust
    compiler.target_support = if is_root {
        // When compiling the root package it is context specific as to whether we need to
        // enforce that all functions have an implementation for the current target.
        // Typically we do, but if we are using `gleam run -m $module` to run a module that
        // belongs to a dependency we don't need to enforce this as we don't want to fail
        // compilation. It's impossible for a dependecy module to call functions from the root
        // package, so it's OK if they could not be compiled.
        self.options.root_target_support
    } else {
        // When compiling dependencies we don't enforce that all functions have an
        // implementation for the current target. It is OK if they have APIs that are
        // unaccessible so long as they are not used by the root package.
        TargetSupport::NotEnforced
    };
    ```
6. [`PackageCompiler.compile`]

## `PackageCompiler.compile`

- [compiler-code/src/build/package_compiler.rs](https://github.com/gleam-lang/gleam/blob/874b0bb616a9111d2d34dd79b4ee8763c9767c5b/compiler-core/src/build/package_compiler.rs#L117)

1. Check Gleam version compat
2. Load packages via [`PackageLoader`]. This also loads cached packages, reemitting cached warnings
3. Calls `analyse` for the modules to compile (new/changed, `Loaded.to_compile`)
4. Inlining (disabled)
5. Codegen

## PackageLoader.run

1. `PackageLoader.read_sources_and_caches`: Runs `ModuleLoader.load` on each Gleam file in the `src` (etc.) folder
    1. Checks the mtime for the file
    2. If it's been updated, parse the file ([`read_source`]) and return `Input::New`, otherwise `Input::Cached`. No parsing happens -- `Input` is just an enum 4. `read_source` returns an `UncompiledModule`: AST with dependency information via `UntypedModule.dependencies` ([compiler-core/src/ast.rs](https://github.com/gleam-lang/gleam/blob/874b0bb616a9111d2d34dd79b4ee8763c9767c5b/compiler-core/src/ast.rs#L145))
    3. For Erlang, ensure Gleam modules don't have the same names as Erlang std modules
2. Look for modules that exist in cache but not the filesystem anymore. Mark them as stale so their dependents will be refreshed.
3. Toposort the inputs. Before that, they're sorted A-Z for determinism
4. Check to see if any caches need to be invalidated
    - New, uncached module (`Input::New`): Stale + to compile
    - Cached (`Input::Cached`), with stale dependencies: Stale this module + to compile
    - Cached, no stale deps: Load from cache
    - Returns `Loaded`
        ```rust
        struct Loaded {
            pub to_compile: Vec<UncompiledModule>,
            pub cached: Vec<ModuleInterface>,
        }
        ```

## `analyse`

- [compiler-core/src/build/package_compiler.rs](https://github.com/gleam-lang/gleam/blob/874b0bb616a9111d2d34dd79b4ee8763c9767c5b/compiler-core/src/build/package_compiler.rs#L551)

Direct deps: Normal deps + dev deps

1. Check if each uncompiled module can be typechecked
    1. If it depends on modules with errors, it will be skipped
    2. If it depends on skipped modules
2. Typecheck. `module_types` is passed to the typechecker. That is used as `importable_modules`. The importer looks for module names from that map.
3. There is a `module_types` map passed to `analyse`. After the module is typechecked, the type info from the typed AST is inserted
    ```rust
    let _ = module_types.insert(module.name.clone(), module.ast.type_info.clone());
    ```
4. Emit empty module warnings
5. If typechecking partially fails, its module data is still registered for the LSP. But will not be reloaded from cache
