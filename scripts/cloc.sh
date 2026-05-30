#!/usr/bin/env bash

all_flag=false
list_flag=false
list_limit=10
ext_arg=""

CODE_EXTS=(go sh klar ts js)

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
    --all | -a)
        all_flag=true
        ;;
    -l | --list | --board)
        list_flag=true
        if [[ $2 =~ ^[0-9]+$ ]]; then
            list_limit="$2"
            shift
        fi
        ;;
    -*)
        # Ignore unknown flags or handle them if needed
        ;;
    *)
        # First non-flag argument is treated as a custom extension
        if [ -z "$ext_arg" ]; then
            ext_arg="$1"
        fi
        ;;
    esac
    shift
done

# Build file list
if [ "$all_flag" = true ]; then
    if [ -n "$ext_arg" ]; then
        files=$(git ls-files "*.$ext_arg")
    else
        files=$(git ls-files)
    fi
else
    if [ -n "$ext_arg" ]; then
        files=$(git ls-files "*.$ext_arg")
    else
        patterns=()
        for ext in "${CODE_EXTS[@]}"; do
            patterns+=("*.$ext")
        done
        files=$(git ls-files "${patterns[@]}")
    fi

    # Exclude generated files
    if [ -n "$files" ]; then
        # grep -L returns files that DO NOT match the pattern
        files=$(xargs grep -L '^// Code generated .* DO NOT EDIT\.$' <<< "$files" 2> /dev/null)
    fi
fi

if [ -z "$files" ]; then
    echo "0"
    exit 0
fi

if [ "$list_flag" = true ]; then
    xargs -I {} wc -l {} <<< "$files" | sort -nr | head -n "$list_limit"
else
    # shellcheck disable=SC2086
    xargs cat <<< "$files" | grep -v '^\s*//' | grep -cv '^\s*$'
fi
