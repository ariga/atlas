#!/bin/bash
set -e

source dev-container-features-test-lib

check "atlas is on PATH" which atlas
check "atlas version" atlas version

reportResults
