#!/usr/bin/env bash
if [[ $1 == "-l" ]]; then
    git ls-files "*.${2:-go}" | xargs -I {} wc -l {} | sort -nr | head -n "${3:-10}"
else
    git ls-files "*.${1:-go}" |
        xargs cat |
        grep -v '^\s*//' |
        grep -cv '^\s*$'
fi
