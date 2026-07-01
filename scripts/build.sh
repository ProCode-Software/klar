#!/usr/bin/env bash
set -e -o pipefail

KLAR_ROOT=$(realpath "$(dirname "$0")/..")
PRODUCTS=(klar glas)

# GOOS and GOARCH names
OSES=(linux darwin windows)
declare -A ARCHES=(
    [linux]='amd64 arm64'
    [darwin]='amd64 arm64'
    [windows]='amd64 arm64'
)

# Filenames for the binaries
declare -A OS_NAMES=(
    [linux]=linux
    [darwin]=macos
    [windows]=windows
) ARCH_NAMES=(
    [amd64]=x86_64
    [arm64]=arm64
)

export GOEXPERIMENT=jsonv2

VERSION="0.0.1"
COMMIT=$(git rev-parse --short HEAD)
[[ $COMMIT_VER -eq 1 ]] && ver_name=$COMMIT || ver_name=$VERSION
LDFLAGS=("-X 'github.com/ProCode-Software/klar/internal/cli.Klar"{Version=$VERSION,Commit=$COMMIT}"'")

function pre_build() {
    # Generate code
    (cd "$KLAR_ROOT" && make gen)
    # Run tests
    for os in "${OSES[@]}"; do
        for arch in ${ARCHES[$os]}; do
            echo "Testing $os/$arch..."
            # GOOS=$os GOARCH=$arch go test "$KLAR_ROOT/..."
        done
    done
}

function build_binaries() {
    rm -rf "${KLAR_ROOT:?}/bin"
    mkdir -p "$KLAR_ROOT/bin"

    for os in "${OSES[@]}"; do
        for arch in ${ARCHES[$os]}; do
            for product in "${PRODUCTS[@]}"; do
                echo "Compiling $product for $os/$arch..."
                out_path="$KLAR_ROOT/bin/$product-$ver_name-${OS_NAMES[$os]}-${ARCH_NAMES[$arch]}"
                if [[ $os == "windows" ]]; then
                    out_path+=".exe"
                fi
                GOOS=$os GOARCH=$arch go build -ldflags="${LDFLAGS[*]}" --trimpath \
                    -o "$out_path" \
                    "$KLAR_ROOT/cmd/$product"
            done
        done
    done
}

function build_klar_wasm() {
    echo Compiling klarwasm...
    out_path="$KLAR_ROOT/bin/klar-$ver_name-browser.wasm"
    GOOS=js GOARCH=wasm go build -ldflags="${LDFLAGS[*]}" --trimpath \
        -o "$out_path" \
        "$KLAR_ROOT/cmd/klarwasm"
}

function pack_stdlib() {
    echo Packing standard library...
    zip -r "$KLAR_ROOT/bin/stdlib.zip" "./std" > /dev/null
}

function main() {
    pre_build
    build_binaries
    build_klar_wasm
    pack_stdlib
    echo Build complete!
}

main
