#!/bin/bash
set -e

# Optional: Import test library bundled with the devcontainer CLI.
# See https://github.com/devcontainers/cli/blob/main/docs/features/test.md
source dev-container-features-test-lib

# Definition specific tests
check "atlas is on PATH" which atlas
check "atlas version" atlas version

# Report results
reportResults
