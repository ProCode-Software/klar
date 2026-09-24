#!/usr/bin/env sh
# Run this script from the /internal/ast folder
if ! go run ./generate > /dev/null 2>&1; then
    bun run ../../scripts/replaceASTNodeImpls.ts
    go run ./generate
fi
