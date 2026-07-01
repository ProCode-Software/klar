# Klar Contributing Guide

Welcome to the Klar repository! We appreciate your interest in contributing to Klar.

## Quick Links

Here are some topics you may have been directed to this file about. Click on a link below to jump to that section.

- [Issues & Bug Reports](#bug--crash-reports)
- [Pull Requests](#pull-requests)
- [Discussions vs. Issues vs. PRs](#discussions-vs-issues-vs-prs)
- [Repo Development Guide](#development-guide)
- [Project Structure](#project-structure) & [Common Go Packages](#common-packagesdirectories)
- [Using AI](#using-ai)

## Goals

Klar is a programming language designed to be different from existing languges, unafraid to rethink features to make an experience that is more consistent and easier to learn. It's also a language that can be taught as a first programming language. The feature set stays small and opinionated. There aren't multiple ways to implement something in Klar; just one idiomatic solution.

We also want Klar to be a safe language. Error values instead of exceptions, as little runtime panics as possible, a real (but simple) type system that can be used to catch errors at compile-time.

Klar compiles to JavaScipt, and in the future, possibly its own runtime. Related to JavaScript, we want Klar to be interopable with it so JS developers can gradually migrate more functionality to Klar and experience its safety and build speed benefits.

Other smaller goals:

- Errors for humans. Friendly error messages that people can actually understand, rather than sounding robotic. Display specific context and suggest hints.
- An extensive standard library so developers have all the functionality they need, first-party and pre-installed. This reduces the reliance on external libraries, which may not have the same quality as the standard library, and also the growing issue of supply chain attacks (that NPM and the JS ecosystem are notorious for)
- An uncomplicated package manager. And one that is safe: Display packages' details to the user before installing them
- A decentralized package registry. Developers can upload their packages to platforms their familiar with --- the same place they upload their code. This means less passwords users have to remember --- and less attack vectors for developers to get hacked. No occupied package names or name scalping/squatting.

## Developer Background

### Klar Users

### Project Contributors

## Commenting on RFCs

When we started working on Klar, for most things in the language, we had an exact idea on how to implement them. But there are still a lot that we need feedback for.

We collect feedback through requests for comments (RFCs), which are on the repo's [discussions] page. When commenting on an RFC, things we recommend you doing include:

- Voting on approaches we or other users created, that you like best
- Asking questions (safety implications, how easy is it for beginners to learn, how much do they have to remember, etc.)
- Creating a new approach, from your own ideas, or combining ideas discussed by others

## Feedback on Existing Features

> Expect this process to change after Klar v1.0.

While we're pre-1.0, we want to perfect the language's features, so your feedback is critical!

## Feature Requests & Language Proposals

> Expect this process to change after Klar v1.0.

We also want you to be aware that Klar's goal is to stay a small, opinionated, learnable language. We only add features for things that haven't been possible before; not another way to do something already in the language. If you really want us to change our current approach, convince us by sending feedback about it in discussions.

Do not create a PR with a new feature before discussing it! Only start a PR for an existing issue started by a maintainer, or something on our roadmap.

## Bug & Crash Reports

When filing a crash report issue, please include:

- The revision of the compiler you ran (the commit number of the version of Klar you built/downloaded); and additionally, modifications you made before building Klar. If those are applicable, we recommend submitting a repository with your changes.
- What you did to encounter the error, such as:
    - The exact commands you ran
    - The Klar project you're building. If those intentionally contain errors, explain them.
- The platform you're running Klar on (operating system, architecture)

We also allow reporting bugs that you haven't experienced, but are still possible (based on the source code) under certain circumstances and steps. If you file these, make sure you look at the compiler's source code, and provide steps that could likely produce the bug.

As this project is still in early development, **do not file bug reports on unimplemented or incomplete features.** You can still file bug reports on parts we consider complete, such as the lexer.

Examples of incomplete features you shouldn't report bugs on:

- The type checker
- Incomplete features from the [roadmap](https://github.com/klar-lang/klar/tree/main/ROADMAP.md)
- Stubs, placeholders, and code with TODO comments

Examples of complete features you can report bugs on:

- Areas of functionality that haven't seen major commits in a while (except bug fixes), such as the parser
- Features marked as complete in our roadmap
- Panics outside of features we haven't committed to recently.

When reporting bugs, consider looking at smaller features and details, rather than the full feature. For example, you can report a bug in the Klon decoder, but don't expect everything _else_ in the Klon library to be complete.

## Pull Requests

**Make sure there's already an issue filed before starting a PR!**. PRs aren't an excuse to skip the issue/discussion commenting process.

Types of PRs that can be submitted without a reference issue are:

- [PRs that fix typos](#fixing-typos)
- [Documentation updates](#contributing-to-documentation)
- PRs that implement TODOs (found such as in the code)

Also, for repo organization purposes, you should include only **one feature per PR**.

## Fixing Typos

If there's a typo in a Markdown file or documentation in the repo, you're welcome to fix it and send it to us. As another exception, when reporting typos, include as many changes as possible!

## Discussions vs. Issues vs. PRs

| Post in Discussions           | Create an issue                            | Start a PR               |
| ----------------------------- | ------------------------------------------ | ------------------------ |
| Feature requests              | Specification/implementation discrepancies | Items on the roadmap     |
| Language proposals            | Potential bugs                             | Typo/grammar corrections |
| Comments on RFCs              | Bug reports                                | Updates to docs          |
| New RFC proposals             |                                            | Implementing TODOs       |
| Comments on existing features |                                            |                          |

## Using AI

You may use AI to write code and tests. However, **you may not use AI to write issues, PRs, or discussions in this project.** We want thoughts of, and interactions and collaboration, between humans in the Klar project. Remember that this programming language is targeted to humans and beginners, not computers and AI agents. Low-effort or AI-generated PRs, issues, comments, or discussions will be closed quickly.

### Writing Code with AI

If you choose to write code with AI, make sure it follows the same style as the rest of the project's code. The author should be responsible for checking; don't put it on us. We're more likely to close issues with poor code style if they contain more AI-generated content. We do not want to be disgusted by AI slop code that will make us immediately close your PR.

> [!TIP]
> Use the [AGENTS.md](./AGENTS.md) and this contributing guide (CONTRIBUING.md) as steering guides for LLMs

Some common signs of LLM-generated code that violate our code style (for Go code), based on my personal experiences with AI agents:

- Excessive separation of lines in functions, between groups of logic and returns
- Excessive comments explaining _absolutely everything_
- Excessive numbering/lettering groups of logic after an LLM sees it in the codebase
- Excessive nesting of if-statements
- Short or generic variable names. Or using names I don't like, such as `idx` ("index").

    ```go
    type Compiler struct {
        WorkDir   string
        Mode      BuildMode
        Reporter  *reporter.Reporter
        StartTime time.Time
        Errors    []*klarerrs.Error
        Warnings  []*klarerrs.Error
        Progress  Progress
        Parser    Parser
        collectMu sync.Mutex // Correct
        mu        sync.Mutex // Incorrect, what LLMs write
        *slog.Logger
    }
    ```

    - _For context, `collectMu` is used for concurently sending `Error`s to the `Errors` and `Warnings` slices (as files are parsed in parallel)_
    - If a mutex for this purpose were to be added to the multi-purpose `Compiler` object, it should be named after what it's used for, not just `mu` just because it's a mutex. What happens if we need multiple mutexes for the Compiler?

- Not using the latest language features, or rejecting them as incorrect or nonexistent (such as `new(expr)` to return a pointer to `expr`, added in Go 1.26)

TODO: Add examples of each

### Fixing Bugs / Filing Vulnerabilities with AI

In recent months (as of 2026), in various open-source projects, maintainers have seen an influx in AI-generated security reports, with pressure to quickly patch them.

We're welcome to security reports discovered by AI (with the human-written report requirement), but we ask that **if you can use AI to find a bug, you should also use it to suggest (and implement) a fix.** You should not fully rely on maintainers to fix them.

Be aware that many open-source contributors work on this project in their freetimes, and pressure to fix vulnerabilities take time and can lead to burnout. We would also like to prioritize and reward vulnerabilities filed by humans that took their time to investigate.

## Development Guide

### Project Structure

- `cmd` - Entry points for CLI commands, including `klar` and `glas`.
- `docs` - Documentation related to the project. We plan for the public-facing docs website to be in its own repo.
- `internal` - Go packages internal to Klar.
- `klar-vscode` - The VSCode extension for Klar, containing language definitions and syntax highlighting, and an LSP for Klar and Klon in the future.
- `pkg` - Go packages publicly available for other projects to import
- `samples` - Klar project samples
- `scripts` - Scripts for project development
- `std` - The Klar standard library

### Compiler (Go)

All you need to start working on the compiler is the latest version of the [Go compiler](https://go.dev). This project uses the latest version of Go, defined in the [`go.mod`](./go.mod) file in the root of the repo.

Then run `go mod tidy` to install the dependencies and tools defined in the project's `go.mod`.

```sh
# Run the Klar CLI
./run [args...] # Use this instead of `go run ./cmd/klar`

# Build the Klar CLI to an executable
go build ./cmd/klar

# Generate code prior to running/building
make gen # Use this over `go generate`

# Run Go tests
go test ./...

# Format Go files
./scripts/format.sh
```

#### Before Submitting Code

We have more information about what to do before submitting changes, but as a TL;DR, run the following:

```sh
make gen # Create generated files
./scripts/format.sh # Format
go test ./... # Run tests
./scripts/lint.sh # Lint
./scripts/spellcheck.sh # Spellcheck strings and comments
```

#### Common Packages/Directories

Klar's project structure is based on Go's, with packages internal to the compiler in the `internal` folder, commands in the `cmd` folder, and public packages in `pkg`.

Below are some packages you're likely to work on when working on the compiler. Some of the listed folders may have an additional readme for more specific explanations of the architectures.

- [`cmd/klar`](./cmd/klar), [`cmd/glas`](./cmd/glas) - Entry points for the Klar and Glas CLIs
    - [`cmd/klar/internal/klarcmd`](./cmd/klar/internal/klarcmd), [`cmd/glas/internal/glascmd`]() - Contain the list of CLI commands
    - `cmd/{klar,glas}/internal/[command]` - Entry points for individual commands, such as `klar build` located in [`cmd/klar/internal/klarcmd/build`](./cmd/klar/internal/klarcmd/build). Each package also contains descriptions for each command, and flags for the commands.
- [`internal/lexer`](./internal/lexer) - Implementation of the Klar lexer, which reads tokens from a file stream.
- [`internal/parser`](./internal/parser) - The Klar parser, which converts a list of tokens to an abstract syntax tree (AST), validating the language's syntax and reporting syntax errors.
- [`internal/analysis`](./internal/analysis) - The Klar type checker, which creates type information from the ASTs of a module. Correct usage of syntax and types are checked, and errors are reported.
- [`internal/build`](./internal/build) - The core of the Klar compiler. It handles input resolution, parsing and typechecking of modules, the compiling pipeline, and reporting erors.
- [`internal/config`](./internal/config) - Contains the definitions for Klar config formats, such as `glas.pack`, `klar.build`, and `glas.lock`.
- [`internal/command`](./internal/command) - Implements extra features for the Klar CLI intended to make the CLI easier to use and understand, such as displaying the flags of commands, and validating argument usage.
- [`internal/klarerrs`](./internal/klarerrs) - Contains the representation for compile errors and all error codes and messages reported by the compiler. It is used by the parser and and analysis.
- [`internal/target`](./internal/target) - Contains the definitions for Klar targets
- [`internal/version`](./internal/version) - Implements the Klar versioning format, with a `Version` object and a parser.
- [`pkg/klon`](./pkg/klon) - Implements the Klon format, a custom markup language used in `glas.pack` and other Klar configuration files. Contains a parser and a public API.
- [`pkg/argparse`](./pkg/argparse) - A public library used by Klar for CLI flag parsing
- [`pkg/klarerrors/reporter`](./pkg/klarerrors/reporter) - The implementation of Klar's CLI error printer, displaying errors with hints and source code context in the terminal.

#### Generated Code

Some code in the Klar compiler is generated. Generated Go files have a comment at the top stating _DO NOT EDIT_. Do not gitignore generated files.

Regenerating files is required after making changes to the following:

- The `internal/ast` package
- Error codes in `internal/klarerrs`

To generate these files, run `make gen`.

#### Linting

Lint your Go code by running the `./scripts/lint.sh` script. Also, ensure comments and strings have correct spelling by running `./scripts/spellcheck.sh`.

### Standard Library (Klar)

It is recommended to compile the standard library using a custom-built Klar compiler, as what will happen in our release pipeline. See the [compiler development guide]() on information about building the Klar compiler.

To use your modified standard library after making changes, **set the `$KLAR_STD` environment variable to the path of the `std` folder.** If you run the Klar CLI using the `./run` script at the root, `$KLAR_STD` is automatically set to the `std` folder located in the Klar repo. If you forget to do this, the globally-installed standard library will be used instead of your modified one!

### JavaScript Code

This project uses the [Bun](https://bun.sh) runtime and package manager. Use the latest version to be safe, but the hard requirement is the one defined in each `package.json` file. You'll need Bun to run package.json scripts. Wherever you find a `package.json`, run `bun install` to install dependencies before you get started.

### Building for Production

We use the `./scripts/build.sh` to build Klar binaries for production. It involves:

- Generating code
- Testing
- Building cross-platform Klar/Glas binaries (Linux, macOS, Windows systems, each for `x86_64` and `arm64` architectures) in the `bin` folder
- Compiling a Wasm version of the Klar compiler, also in `bin`
- Zipping the standard library so the install script can download it

### What Not to Do

- No vendor-specific LLM instruction files (`CLAUDE.md`, `GEMINI.md`, `copilot-instructions.md`, `.gemini/`). You may gradually add them to `.gitignore`. Creating `AGENTS.md` files are allowed as they will work with most LLMs.
- No JavaScript lockfiles other than `bun.lock` (including `package-lock.json`, `yarn.lock`)
- Don't add new top-level files or folders without explaining to the maintainers in a PR.

## Code Style

With more code being written by AI, to maintan the code quality from humans, we've created style guides for some languages. See the [docs/CodeStyle](./docs/CodeStyle) folder for the languages we've written style guides for. For Go, see [docs/CodeStyle/GoStyleGuide.md](./docs/CodeStyle/GoStyleGuide.md)

In the Klar codebase, we want the source code to be **understandable and maintainable**. Developers looking for inspiration from other compilers should be able to look at the Klar codebase, and understand our implementation. Additional benefits of understandable and maintainable code:

- Once you write it, you don't need AI just to edit it
- Bugs are easier to discover and fix
- Intentions of code can be understood
- Easier to write specs on
- Easier to factor and reuse

### Formatters

- **Go:** [gofumpt](https://github.com/mvdan/gofumpt) (a more opinionated version of gofmt) via `./scripts/format.sh`
- **JavaScript, TypeScript, JSON**: [oxfmt](https://oxc.rs/) via `bun oxfmt`

Ensure you format your code before submitting your PR. Never file a PR just to format unformatted files.

## Contributing to Documentation

> For fixing typos and grammar, see the [Fixing Typos](#fixing-typos) section.

Some documentation files that can be updated include:

- AGENTS.md
- Files in `docs/`
- README.md files in the root and throughout the tree
- CONTRIBUTING.md

When submitting your changes, you're allowed to:

- Update multiple files with information about the same topic, or
- Update a single file with information about multiple topics
