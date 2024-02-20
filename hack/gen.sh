#!/bin/bash

set -e -x

# Note if you are not seeing generated code, most likely it's being placed into a different folder
# (e.g. Do you have GOPATH directory structure correctly named for this project?)

# Return a GOPATH to a temp directory. Works around the out-of-GOPATH issues
# for k8s client gen mixed with go mod.
# Intended to be used like:
#   export GOPATH=$(go_mod_gopath_hack)
function go_mod_gopath_hack() {
  local tmp_dir=$(mktemp -d)
  local module="$(go list -m)"

  local tmp_repo="${tmp_dir}/src/${module}"
  mkdir -p "$(dirname ${tmp_repo})"
  ln -s "$PWD" "${tmp_repo}"

  echo "${tmp_dir}"
}
export GOPATH=$(go_mod_gopath_hack)
trap "rm -rf ${GOPATH}; git checkout vendor" EXIT

VENDIR_PKG=carvel.dev/vendir
# Based on vendor/k8s.io/code-generator/generate-groups.sh
#go run vendor/k8s.io/code-generator/cmd/deepcopy-gen/main.go \
echo "Generating deepcopy"
rm -f $(find pkg/vendir|grep zz_generated.deepcopy)
go run vendor/k8s.io/code-generator/cmd/deepcopy-gen/main.go \
  --input-dirs "${VENDIR_PKG}/pkg/vendir/versions/v1alpha1" \
  --input-dirs "${KC_PKG}/pkg/vendir/versions" \
  -O zz_generated.deepcopy \
  --go-header-file hack/gen-boilerplate.txt

# Install protoc binary as directed by https://github.com/gogo/protobuf#installation
# (Chosen: https://github.com/protocolbuffers/protobuf/releases/download/v3.0.2/protoc-3.0.2-osx-x86_64.zip)
# unzip archive into ./tmp/protoc-dl directory
export PATH=$PWD/tmp/protoc-dl/bin/:$PATH
protoc --version

# Generate binaries called out by protoc binary
export GOBIN=$PWD/tmp/gen-apiserver-bin
rm -rf $GOBIN
go install \
  github.com/gogo/protobuf/protoc-gen-gogo \
  github.com/gogo/protobuf/protoc-gen-gofast \
  golang.org/x/tools/cmd/goimports \
  k8s.io/code-generator/cmd/go-to-protobuf
export PATH=$GOBIN:$PATH

rm -f $(find pkg/vendir|grep '\.proto')
go-to-protobuf \
  --proto-import "${GOPATH}/src/${VENDIR_PKG}/vendor" \
  --packages "${VENDIR_PKG}/pkg/vendir/versions/v1alpha1" \
  --vendor-output-base="${GOPATH}/src/${VENDIR_PKG}/vendor" \
  --go-header-file hack/gen-boilerplate.txt
