#!/bin/bash

set -e -x -u

function get_latest_git_tag {
  git describe --tags | grep -Eo 'v[0-9]+\.[0-9]+\.[0-9]+'
}

VERSION="${1:-$(get_latest_git_tag)}"

go fmt ./cmd/... ./pkg/... ./test/...
go mod vendor
go mod tidy

# related to https://github.com/vmware-tanzu/carvel-imgpkg/pull/255
# there doesn't appear to be a simple way to disable the defaultDockerConfigProvider
# Having defaultDockerConfigProvider enabled by default results in the imgpkg auth ordering not working correctly
# Specifically, the docker config.json is loaded before cli flags (and maybe even IaaS metadata services)
git apply --ignore-space-change --ignore-whitespace ./hack/patch-k8s-pkg-credentialprovider.patch

git diff --exit-code vendor/github.com/vdemeester || {
  echo 'found changes in the project. when expected none. exiting'
  exit 1
}

# makes builds reproducible
export CGO_ENABLED=0
LDFLAGS="-X github.com/vmware-tanzu/carvel-vendir/pkg/vendir/version.Version=$VERSION"
repro_flags="-trimpath -mod=vendor"

GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" $repro_flags -o vendir-darwin-amd64 ./cmd/vendir/...
GOOS=darwin GOARCH=arm64 go build -ldflags="$LDFLAGS" $repro_flags -o vendir-darwin-arm64 ./cmd/vendir/...
GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" $repro_flags -o vendir-linux-amd64 ./cmd/vendir/...
GOOS=linux GOARCH=arm64 go build -ldflags="$LDFLAGS" $repro_flags -o vendir-linux-arm64 ./cmd/vendir/...
GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" $repro_flags -o vendir-windows-amd64.exe ./cmd/vendir/...

shasum -a 256 ./vendir-*
