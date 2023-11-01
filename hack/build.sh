#!/bin/bash

set -e -x -u

VERSION="${1:-0.0.0+develop}"

# makes builds reproducible
export CGO_ENABLED=0
LDFLAGS="-X carvel.dev/vendir/pkg/vendir/version.Version=$VERSION"
repro_flags="-trimpath -mod=vendor"

go mod vendor
go mod tidy
go fmt ./cmd/... ./pkg/... ./test/...

# export GOOS=linux GOARCH=amd64
go build -ldflags="$LDFLAGS" $repro_flags -o vendir ./cmd/vendir/...
./vendir version

# compile tests, but do not run them: https://github.com/golang/go/issues/15513#issuecomment-839126426
go test --exec=echo ./... >/dev/null

echo "Success"
