#!/bin/bash

set -e -x -u

go clean -testcache

export VENDIR_BINARY_PATH=$PWD/vendir

go test ./test/e2e/ -timeout 60m -test.v $@

echo E2E SUCCESS
