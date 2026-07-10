#!/usr/bin/env bash
export GOEXPERIMENT=jsonv2
go fix ./...
golangci-lint run "$@" &
"$(dirname "$(realpath "$0")")/gopls_check.sh"

wait