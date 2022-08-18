#!/usr/bin/env bash

set -e
set -u

tag="${1}"
dir=$(mktemp -d)

git clone --depth 1 --branch "${tag}" git@github.com:kubernetes/kubernetes.git "${dir}"
cp -r "${dir}/pkg/credentialprovider/." .

find . \( -name "OWNERS" \
  -o -name "OWNERS_ALIASES" \
  -o -name "BUILD" \
  -o -name "BUILD.bazel" \) -exec rm -f {} +


oldpkg="k8s.io/kubernetes/pkg/credentialprovider"
newpkg="github.com/vdemeester/k8s-pkg-credentialprovider"

find ./ -type f -name "*.go" \
  -exec sed -i "s,${oldpkg},${newpkg},g" {} \;
sed -i "s,\tk8s\.io/\(.*\) v.*,\tk8s.io/\1 v0.${tag#v1.},g" go.mod
sed -i "s,\tk8s\.io/klog/v2 v.*,\tk8s\.io/klog/v2 v2.8.0,g" go.mod

# Remove plugin folder
rm -fR plugin

# Run go mod tidy
go mod tidy
