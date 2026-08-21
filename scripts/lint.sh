#!/usr/bin/env bash
go fix ./...
golangci-lint run "$@" &
go vet -composites=false ./...
"$(dirname "$(realpath "$0")")/gopls_check.sh"

wait