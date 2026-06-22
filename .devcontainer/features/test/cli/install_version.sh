#!/bin/bash
set -e

source dev-container-features-test-lib

check "atlas is on PATH" which atlas
check "atlas version" atlas version
check "atlas reports requested version" bash -c "atlas version | grep -q 'v1.2.0'"

reportResults
