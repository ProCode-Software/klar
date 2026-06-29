#!/usr/bin/env bash
set -e -o pipefail

function red() { echo -e "\033[31m$1\033[0m"; }

for arg in "$@"; do
    case "$arg" in
    --global) global=1 ;;
    --add-to-path) add_to_path=1 ;;
    --help | -h)
        echo "Flags:
    --global       Install Klar globally (requires sudo)
    --add-to-path  Add Klar to PATH
    --help         Show this help message

https://github.com/ProCode-Software/klar"

        exit 0
        ;;
    -*)
        red "Unknown flag: $arg" && exit 2
        ;;
    esac
done

if ! command -v git &> /dev/null; then
    red "git is required to install Klar. Install it at https://git-scm.com." && exit 1
fi

if ! command -v go &> /dev/null; then
    red "Go is required to install Klar. Install it at https://go.dev." && exit 1
fi

build_dir=$(mktemp -d)
cd "$build_dir" || exit

git clone https://github.com/ProCode-Software/klar.git

# Convert current platform to GOOS
GOOS=$(uname -s)
export GOOS=${GOOS,,}

case "$(uname -m)" in
x86_64) GOARCH=amd64 ;;
arm64) GOARCH=arm64 ;;
i386 | i686) GOARCH=386 ;;
*) red "Unsupported architecure: $(uname -m)" && exit 1 ;;
esac
export GOARCH

ext=""
if [ "$GOOS" = "windows" ]; then
    ext=".exe"
fi

export GOEXPERIMENT=jsonv2
VERSION="0.1.0"
LDFLAGS="-X 'github.com/ProCode-Software/klar/internal/cli.KlarVersion=$VERSION'"

bin_name="klar$ext"
go build -ldflags="$LDFLAGS" -o "$bin_name" ./cmd/klar

BIN_DIR=

if [[ $global == 1 ]]; then
    BIN_DIR="/usr/bin/klar"
else
    BIN_DIR="$HOME/.local/bin"
fi

if [[ $global == 1 ]]; then
    sudo install -m 755 "$bin_name" "$BIN_DIR"
else
    install -m 755 "$bin_name" "$BIN_DIR"
fi

if [[ $add_to_path == 1 ]]; then
    echo 'export PATH="$PATH:'"$BIN_DIR"'"' >> ~/.bashrc
fi

echo -e "\e[32mKlar has been successfully installed!

To get started, run 'klar --help'\e[32m"