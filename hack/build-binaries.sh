#!/bin/bash

set -e -x -u

./hack/build.sh

function get_latest_git_tag {
  git describe --tags | grep -Eo 'v[0-9]+\.[0-9]+\.[0-9]+'
}

VERSION="${1:-`get_latest_git_tag`}"

# makes builds reproducible
export CGO_ENABLED=0
LDFLAGS="-X github.com/vmware-tanzu/carvel-vendir/pkg/vendir/version.Version=$VERSION -buildid="
repro_flags="-trimpath -mod=vendor"


GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" $repro_flags -o vendir-darwin-amd64 ./cmd/vendir/...
GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" $repro_flags -o vendir-linux-amd64 ./cmd/vendir/...
GOOS=linux GOARCH=arm64 go build -ldflags="$LDFLAGS" $repro_flags -o vendir-linux-arm64 ./cmd/vendir/...
GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" $repro_flags -o vendir-windows-amd64.exe ./cmd/vendir/...

shasum -a 256 ./vendir-*
