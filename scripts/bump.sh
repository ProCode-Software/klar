#!/usr/bin/env bash
set -e
newVersion=$1
packages=(**/package.json ./package.json)

for package in "${packages[@]}"; do
    sed -i 's/"version": ".*"/"version": "'"${newVersion}"'"/' "$package"
done

cat <<-EOF > ./internal/version/version.go
package version

const KlarVersion = "${newVersion}"
EOF

echo "Bumped Klar version to v${newVersion}"