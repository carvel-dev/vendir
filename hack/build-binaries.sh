#!/bin/bash

set -e -x -u

BUILD_VALUES= ./hack/build.sh

GOOS=darwin GOARCH=amd64 go build -o vendir-darwin-amd64 ./cmd/vendir/...
GOOS=linux GOARCH=amd64 go build -o vendir-linux-amd64 ./cmd/vendir/...
GOOS=windows GOARCH=amd64 go build -o vendir-windows-amd64.exe ./cmd/vendir/...

shasum -a 256 ./vendir-*-amd64*
