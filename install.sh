#!/usr/bin/env bash
set -e -o pipefail

function red() { echo -e "\e[91m$1\e[0m"; }
function yellow() { echo -e "\e[93m$1\e[0m"; }
function progress() { echo -e "\e[35m$1\e[0m"; }

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

if [[ -n $add_to_path && $global -eq 1 ]]; then
    red "Can't enable '--add-to-path' with '--global'" && exit 1
fi

# Ensure we have Git and Go installed
if ! command -v git &> /dev/null; then
    red "git is required to install Klar. Install it at https://git-scm.com." && exit 1
fi

if ! command -v go &> /dev/null; then
    red "Go is required to install Klar. Install it at https://go.dev." && exit 1
fi

if [[ -z $global ]]; then
    yellow "Where do you want to install Klar?"
    select location in "Local (current user)" "Global (all users)"; do
        case "$location" in
        "Local (current user)")
            global=0
            read -p "$(yellow "Do you want to add Klar to PATH? (Y/n): ")" -n 1 -r
            echo
            if [[ ${REPLY,,} != 'n' ]]; then
                add_to_path=1
            fi
            ;;
        "Global (all users)") global=1 ;;
        esac
        break
    done
    echo
fi

build_dir=$(mktemp -d)
cd "$build_dir" || exit

# Clone Klar repository
progress "📖 Cloning Klar repository..."
set +e
err=$(git clone https://github.com/ProCode-Software/klar.git . 2>&1)
status=$?
set -e
if [[ $status -ne 0 ]]; then
    red "Failed to clone Klar repository. The error was:"
    echo "$err"
    exit 1
fi

# Get the current GOOS and GOARCH
GOOS=$(uname -s)
export GOOS=${GOOS,,}

case "$(uname -m)" in
x86_64) GOARCH=amd64 ;;
arm64) GOARCH=arm64 ;;
i386 | i686) GOARCH=386 ;;
*) red "Unsupported architecure: $(uname -m)" && exit 1 ;;
esac
export GOARCH

# Build Klar and Glas executables
export GOEXPERIMENT=jsonv2
VERSION="0.1.0"
LDFLAGS="-X 'github.com/ProCode-Software/klar/internal/cli.KlarVersion=$VERSION'"

klar_exec="klar"
glas_exec="glas"

if [ "$GOOS" = "windows" ]; then
    klar_exec+=".exe"
    glas_exec+=".exe"
fi

progress "🏗️ Building Klar and Glas binaries..."
go generate ./...
go build -ldflags="$LDFLAGS" -o "$klar_exec" ./cmd/klar
go build -ldflags="$LDFLAGS" -o "$glas_exec" ./cmd/glas

# Install Klar and Glas to bin directory
progress "🚚 Installing Klar and Glas..."
if [[ $global -eq 1 ]]; then
    BIN_DIR="/usr/bin"
    sudo install -m 755 "$klar_exec" "$glas_exec" "$BIN_DIR"
else
    BIN_DIR="$HOME/.local/bin"
    install -m 755 "$klar_exec" "$glas_exec" "$BIN_DIR"
    # Only add to PATH if it's not already there
    if [[ $add_to_path -eq 1 && ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
        echo "export PATH=\"\$PATH:$BIN_DIR\"" >> ~/.bashrc
        # shellcheck disable=SC1090
        source ~/.bashrc
    fi
fi

# Copy the standard library
progress "📚 Installing the standard library..."
# Keep paths in sync with ./internal/module/system_path.go (KlarStdDir)
# shellcheck disable=SC2154
case "$GOOS $global" in
"darwin 1") STD_DIR="/Library/Application Support/Klar/std" ;;
"darwin 0") STD_DIR="$HOME/Library/Application Support/klar/std" ;;
"windows 0") STD_DIR="$LocalAppData/Klar/std" ;;
"windows 1") STD_DIR="$ProgramData/Klar/std" ;;
*" 1") STD_DIR="/usr/share/klar/std" ;;
*" 0") STD_DIR="$HOME/.local/share/klar/std" ;;
esac

mkdir -p "$STD_DIR"
cp -R ./std "$STD_DIR"

echo -e "
\e[1;92m🐨 Klar has been successfully installed!\e[m
To get started, run \e[96m'klar --help'\e[m. To use Glas, run \e[96m'glas --help'\e[m.

\e[1mGitHub: \e[0;34mhttps://github.com/ProCode-Software/klar\e[m"
