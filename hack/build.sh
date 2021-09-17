#!/bin/bash

set -e -x -u

function get_latest_git_tag {
  git describe --tags | grep -Eo '[0-9]+\.[0-9]+\.[0-9]+'
}

VERSION="${1:-`get_latest_git_tag`}"

# makes builds reproducible
export CGO_ENABLED=0
LDFLAGS="-X github.com/vmware-tanzu/carvel-vendir/pkg/vendir/version.Version=$VERSION -buildid="

go mod vendor
go mod tidy
go fmt ./cmd/... ./pkg/... ./test/...

go build -ldflags="$LDFLAGS" -trimpath -mod=vendor -o vendir ./cmd/vendir/...
./vendir version

# compile tests, but do not run them: https://github.com/golang/go/issues/15513#issuecomment-839126426
go test --exec=echo ./... >/dev/null

echo "Success"
