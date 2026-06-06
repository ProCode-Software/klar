#!/usr/bin/env bash
set -e
shopt -s nullglob globstar extglob

new_ver=$1

if [ -z "$new_ver" ]; then
    echo -e "\033[91mExpected a target version (Usage: $0 <version>)\033[m"
    exit 1
fi

# Bump package.json versions
mapfile -t package_jsons < <(
    find . -path '*/node_modules' -prune -o -name package.json -print
)
for file in "${package_jsons[@]}"; do
    sed -i 's/"version": ".*"/"version": "'"${new_ver}"'"/' "$file"
done

# Bump glas.pack versions
glas_pack_klar_vers=(./std/glas.pack)
for file in "${glas_pack_klar_vers[@]}"; do
    sed -E -i 's/version: .+$/version: v'"${new_ver}/" "$file"
done

# Bump minimum Klar version for sample projects
glas_pack_klar_vers=(./samples/*/glas.pack ./std/glas.pack)
for file in "${glas_pack_klar_vers[@]}"; do
    sed -E -i 's/klar: .+$/klar: v'"${new_ver}/" "$file"
done

echo -e "\e[92mBumped Klar version to v${new_ver}\e[m"
