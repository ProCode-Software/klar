#!/usr/bin/env bash
set -e
shopt -s globstar

diagnostics=$(gopls check --severity=hint ./**/*.go | grep -v -E 'unkeyed|unused')
echo "$diagnostics"

if [[ $1 == --fix ]]; then
    files=$(grep -o -P '^.+(?=\:\d+\:[\d\-]+)' <<< "$diagnostics")
    echo -e "\e[1;33mFinding code actions:\e[m"
    for file in $files; do
        codeaction=$(gopls codeaction -list -kind=quickfix "$file" | head -n 1 | grep -o -P '\".*' &)
        echo -e "    \033[1m$file:\033[;m $codeaction" 
    done
    wait
    echo -e -n "\033[32mPress Enter to apply these changes\033[;m: "
    read -r res
    if [[ $res != "" ]]; then
        exit 1
    fi
    for file in $files; do
        gopls codeaction -exec -w -d -kind=quickfix "$file"
    done
fi
