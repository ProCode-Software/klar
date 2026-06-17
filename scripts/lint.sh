#!/usr/bin/env bash
golangci-lint run "$@" &
"$(dirname "$(realpath "$0")")/gopls_check.sh"

wait