#!/bin/bash

set -e -x -u

function get_latest_git_tag {
  git describe --tags | grep -Eo '[0-9]+\.[0-9]+\.[0-9]+'
}

VERSION="${1:-`get_latest_git_tag`}"

# makes builds reproducible
export CGO_ENABLED=0
LDFLAGS="-X github.com/vmware-tanzu/carvel-vendir/pkg/vendir/version.Version=$VERSION -buildid="

GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" -trimpath -mod=vendor -o vendir-darwin-amd64 ./cmd/vendir/...
GOOS=darwin GOARCH=arm64 go build -ldflags="$LDFLAGS" -trimpath -mod=vendor -o vendir-darwin-arm64 ./cmd/vendir/...
GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -trimpath -mod=vendor -o vendir-linux-amd64 ./cmd/vendir/...
GOOS=linux GOARCH=arm64 go build -ldflags="$LDFLAGS" -trimpath -mod=vendor -o vendir-linux-arm64 ./cmd/vendir/...
GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -trimpath -mod=vendor -o vendir-windows-amd64.exe ./cmd/vendir/...

shasum -a 256 ./vendir-darwin-amd64 ./vendir-darwin-arm64 ./vendir-linux-amd64 ./vendir-linux-arm64 ./vendir-windows-amd64.exe
