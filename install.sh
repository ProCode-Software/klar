#!/usr/bin/env bash
set -e -o pipefail

function red() { echo -e "\e[91m$1\e[0m"; }
function yellow() { echo -e "\e[93m$1\e[0m"; }
function progress() { echo -e "\e[35m$1\e[0m"; }

for arg in "$@"; do
    case "$arg" in
    --global) global=1 ;;
    --add-to-path) add_to_path=1 ;;
    --prebuild) use_prebuild=1 ;;
    --help | -h)
        echo "Flags:
    --global       Install Klar globally (requires sudo)
    --add-to-path  Add Klar to PATH
    --help         Show this help message
    --prebuild     Install a prebuild instead of building from source

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

if [[ -t 0 ]]; then
    tty_in=/dev/stdin
elif [[ -r /dev/tty ]]; then
    tty_in=/dev/tty
else
    red "This script needs an interactive terminal to prompt for options. Run with '--help' for details." && exit 1
fi

if [[ -z $global ]]; then
    yellow "Where do you want to install Klar?"
    select location in "Local (current user)" "Global (all users)"; do
        case "$location" in
        "Local (current user)")
            global=0
            read -p "$(yellow "Do you want to add Klar to PATH? (Y/n): ")" -n 1 -r < "$tty_in"
            echo
            if [[ ${REPLY,,} != 'n' ]]; then
                add_to_path=1
            fi
            ;;
        "Global (all users)") global=1 ;;
        esac
        break
    done < "$tty_in"
fi
if [[ -z $use_prebuild ]]; then
    yellow "Do you want to build from source, or use a prebuilt binary?"
    echo -e "Building from source makes the latest features and fixes available, but requires Git and the Go toolchain to be installed, and may take longer to install. Downloading a prebuilt binary is faster, but may not include the latest Klar fixes."
    select build_type in "Build from source" "Download a prebuilt binary"; do
        case "$build_type" in
        "Build from source") use_prebuild=0 ;;
        "Download a prebuilt binary") use_prebuild=1 ;;
        esac
        break
    done < "$tty_in"
fi

echo

build_dir=$(mktemp -d)
cd "$build_dir" || exit

get_exec() {
    GOOS=$1
    klar_exec="klar"
    glas_exec="glas"
    if [[ $GOOS == "windows" ]]; then
        klar_exec+=".exe"
        glas_exec+=".exe"
    fi
}

download_prebuild() {
    # Keys: Output from uname; Values: Binary OS/arch names
    declare -A oses=(
        [darwin]=macos
        [macos]=macos
        [linux]=linux
        [windows]=windows
    ) arches=(
        [x86_64]=x86_64
        [x64]=x86_64
        [arm64]=arm64
        [aarch64]=arm64
    )
    os=$(uname -s)
    # Git Bash on Windows
    if [[ ${os,,} == "msys_nt"* ]]; then os=windows; fi
    prebuild_os_name=${oses[${os,,}]}
    if [[ -z $prebuild_os_name ]]; then
        red "Unfortunately, we don't provide prebuilds for the $os operating system :(
Please build from source instead by rerunning without the '--prebuild' flag" && exit 1
    fi
    arch=$(uname -m)
    prebuild_arch_name=${arches[${arch,,}]}
    if [[ -z $prebuild_arch_name ]]; then
        red "Unfortunately, we don't provide prebuilds for the $arch architecture :(
Please build from source instead by rerunning without the '--prebuild' flag" && exit 1
    fi

    progress "📦 Downloading prebuilt Klar and Glas binaries..."
    products=(klar glas)
    release_json=$(curl -s "https://api.github.com/repos/ProCode-Software/klar/releases?per_page=1")
    if ! command -v jq &> /dev/null; then
        # Alternative for Windows (Git Bash) users without jq
        tag_name=$(echo "$release_json" | grep -oE '"tag_name":[ ]*"[^"]+"' |
            head -n 1 | sed -E 's/"tag_name":[ ]*"([^"]+)"/\1/')
        binary_name=$(echo "$release_json" |
            grep -oE '"browser_download_url":[ ]*"[^"]+"' |
            sed -E 's/"browser_download_url":[ ]*"([^"]+)"/\1/' |
            grep -E 'klar-.*'"$prebuild_os_name"'-'"$prebuild_arch_name"'.*' |
            head -n 1)
    else
        tag_name=$(echo "$release_json" | jq -r '.[0].tag_name')
        binary_name=$(echo "$release_json" |
            jq -r '.[0].assets[] | select(.name | test("^klar-.*'"$prebuild_os_name"'-'"$prebuild_arch_name"'")) | .browser_download_url')
    fi

    if [[ -z $tag_name || $tag_name == null ]]; then
        red "Unfortunately, we couldn't find any Klar releases on GitHub.
  Please build from source instead by rerunning without the '--prebuild' flag" && exit 1
    fi

    if [[ -z $binary_name ]]; then
        red "Unfortunately, we couldn't find a prebuilt binary for $prebuild_os_name-$prebuild_arch_name in release $tag_name.
  Please build from source instead by rerunning without the '--prebuild' flag" && exit 1
    fi

    # Download Klar and Glas
    get_exec "$prebuild_os_name"
    for product in "${products[@]}"; do
        product_url=${binary_name//klar-/$product-}
        product_exec=${product}_exec
        curl -fsSL -o "$build_dir/${!product_exec}" "$product_url"
    done

    # Download the standard library
    progress "📚 Downloading the standard library..."
    curl -fsSL -o "$build_dir/stdlib.zip" "https://github.com/ProCode-Software/klar/releases/download/$tag_name/stdlib.zip"
    unzip -o "$build_dir/stdlib.zip" -d "$build_dir" &>/dev/null
    
    # For the shared stdlib installation step
    [[ $prebuild_os_name == macos ]] && GOOS=darwin || GOOS=$prebuild_os_name
}

build_from_source() {
    # Ensure we have Git and Go installed
    if ! command -v git &> /dev/null; then
        red "git is required to install Klar. Install it at https://git-scm.com." && exit 1
    fi

    if ! command -v go &> /dev/null; then
        red "Go is required to install Klar. Install it at https://go.dev." && exit 1
    fi

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

    get_exec "$GOOS"
    progress "🏗️ Building Klar and Glas binaries..."
    go generate ./...
    go build -ldflags="$LDFLAGS" -o "$klar_exec" ./cmd/klar
    go build -ldflags="$LDFLAGS" -o "$glas_exec" ./cmd/glas
}

if [[ $use_prebuild -eq 1 ]]; then
    download_prebuild
else
    build_from_source
fi

# Install Klar and Glas to bin directory
progress "🚚 Installing Klar and Glas..."
if [[ $global -eq 1 ]]; then
    BIN_DIR="/usr/bin"
    sudo install -m 755 "$klar_exec" "$glas_exec" "$BIN_DIR"
else
    BIN_DIR="$HOME/.local/bin"
    if [[ $GOOS == windows ]]; then BIN_DIR="$LocalAppData/Klar/bin"; fi
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
cp -R ./std/* "$STD_DIR"

echo -e "
\e[1;92m🐨 Klar has been successfully installed!\e[m
To get started, run \e[96m'klar --help'\e[m. To use Glas, run \e[96m'glas --help'\e[m.

\e[1mGitHub: \e[0;34mhttps://github.com/ProCode-Software/klar\e[m"
