#!/bin/bash

set -e -x -u

go clean -testcache

export VENDIR_BINARY_PATH=$PWD/vendir
export GOCACHE=$(go env GOCACHE) # avoid polluting $HOME
export HOME=$(mktemp -d -t vendir-home-dir-XXXXXXXXXX)

go test ./test/e2e/ -timeout 60m -test.v $@

# Directory _must_ be empty for rmdir to succeed
# (If directory is not empty that means that vendir sync
# and tools it calls leaks some state -- not wanted.)
find $HOME
rmdir $HOME

echo E2E SUCCESS
