#!/bin/bash

set -e -x -u

./hack/build.sh

# makes builds reproducible
export CGO_ENABLED=0
repro_flags="-ldflags=-buildid= -trimpath -mod=vendor"

GOOS=darwin GOARCH=amd64 go build $repro_flags -o vendir-darwin-amd64 ./cmd/vendir/...
GOOS=linux GOARCH=amd64 go build $repro_flags -o vendir-linux-amd64 ./cmd/vendir/...
GOOS=windows GOARCH=amd64 go build $repro_flags -o vendir-windows-amd64.exe ./cmd/vendir/...

shasum -a 256 ./vendir-*-amd64*
