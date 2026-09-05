# Klar Project Documentation

The `docs` folder contains documentation related to the internals of the Klar project. The user-facing documentation and guides are located in a different repository (TODO: provide the link).

- **Compiler:** [`internal/build`](../internal/build/README.md)
    - Lexer: [`internal/lexer`](../internal/lexer/README.md)
    - Parser: [`internal/parser`](../internal/parser/README.md)
    - Klar AST: [`internal/ast`](../internal/ast/README.md)
    - Analysis/Typechecker: [`internal/analysis`](../internal/analysis/README.md)
    - Optimizer: [`internal/optimize`](../internal/optimize/README.md)
    - JavaScript IR: [`internal/ir/jsir`](../internal/ir/jsir/README.md)
    - JavaScript Codegen: [`internal/codegen`](../internal/codegen/README.md)
- **Klon:** [`pkg/klon`](../pkg/klon/README.md)
- **Language Server:** [`internal/lsp`](../internal/lsp/README.md)
- **CLI Diagnostics:** [`pkg/klarerrors/reporter`](../pkg/klarerrors/reporter/README.md)
