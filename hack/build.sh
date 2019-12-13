#!/bin/bash

set -e -x -u

go fmt ./cmd/... ./pkg/... ./test/...

# export GOOS=linux GOARCH=amd64
go build -o vendir ./cmd/vendir/...
./vendir version

echo "Success"
