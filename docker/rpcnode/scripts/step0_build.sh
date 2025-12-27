#!/usr/bin/env sh

# Input parameters
ARCH=$(uname -m)

# Build tabid
echo "Building tabid from local branch"
git config --global --add safe.directory /tabilabs/tabi-v3
LEDGER_ENABLED=false
make install
mkdir -p build/generated
echo "DONE" > build/generated/build.complete
