#!/usr/bin/env bash
set -euo pipefail

# Feature options are passed in as upper-cased environment variables.
VERSION="${VERSION:-latest}"
FLAVOR="${FLAVOR:-}"

echo "Installing Atlas CLI (version: ${VERSION}${FLAVOR:+, flavor: ${FLAVOR}})..."

# Ensure the dependencies required by the install script are available.
check_packages() {
    if ! dpkg -s "$@" >/dev/null 2>&1; then
        if [ "$(find /var/lib/apt/lists/* 2>/dev/null | wc -l)" = "0" ]; then
            echo "Running apt-get update..."
            apt-get update -y
        fi
        apt-get -y install --no-install-recommends "$@"
    fi
}

if command -v apt-get >/dev/null 2>&1; then
    check_packages curl ca-certificates
fi

if [ "${VERSION}" = "latest" ]; then
    ATLAS_VERSION="latest"
else
    # Normalize to exactly one leading "v" (accepts both "1.2.0" and "v1.2.0").
    ATLAS_VERSION="v${VERSION#v}"
fi

curl -sSfL https://atlasgo.sh \
    | ATLAS_VERSION="${ATLAS_VERSION}" ATLAS_FLAVOR="${FLAVOR}" sh -s -- --yes

# Verify installation
atlas version
echo "Atlas CLI installed successfully."
