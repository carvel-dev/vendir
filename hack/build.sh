#!/bin/bash

set -e -x -u

VERSION="${1:-0.0.0+develop}"

# makes builds reproducible
export CGO_ENABLED=0
LDFLAGS="-X github.com/vmware-tanzu/carvel-vendir/pkg/vendir/version.Version=$VERSION"
repro_flags="-trimpath -mod=vendor"

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

go fmt ./cmd/... ./pkg/... ./test/...

# export GOOS=linux GOARCH=amd64
go build -ldflags="$LDFLAGS" $repro_flags -o vendir ./cmd/vendir/...
./vendir version

# compile tests, but do not run them: https://github.com/golang/go/issues/15513#issuecomment-839126426
go test --exec=echo ./... >/dev/null

echo "Success"
