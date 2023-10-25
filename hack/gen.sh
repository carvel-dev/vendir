#!/bin/bash

set -e -x

# Note if you are not seeing generated code, most likely it's being placed into a different folder
# (e.g. Do you have GOPATH directory structure correctly named for this project?)

# Based on vendor/k8s.io/code-generator/generate-groups.sh
go run vendor/k8s.io/code-generator/cmd/deepcopy-gen/main.go \
	--input-dirs carvel.dev/vendir/pkg/vendir/versions/v1alpha1 \
	-O zz_generated.deepcopy \
	--bounding-dirs carvel.dev/vendir/pkg/vendir \
	--go-header-file ./hack/gen-boilerplate.txt

# Install protoc binary as directed by https://github.com/gogo/protobuf#installation
# (Chosen: https://github.com/protocolbuffers/protobuf/releases/download/v3.0.2/protoc-3.0.2-osx-x86_64.zip)
# unzip archive into ./tmp/protoc-dl directory
export PATH=$PWD/tmp/protoc-dl/bin/:$PATH
protoc --version

# Generate binaries called out by protoc binary
rm -rf tmp/gen-apiserver-bin/
mkdir -p tmp/gen-apiserver-bin/
go build -o tmp/gen-apiserver-bin/protoc-gen-gogo vendor/github.com/gogo/protobuf/protoc-gen-gogo/main.go
go build -o tmp/gen-apiserver-bin/protoc-gen-gofast vendor/github.com/gogo/protobuf/protoc-gen-gofast/main.go
go build -o tmp/gen-apiserver-bin/goimports vendor/golang.org/x/tools/cmd/goimports/{goimports,goimports_not_gc}.go
export PATH=$PWD/tmp/gen-apiserver-bin/:$PATH

go run vendor/k8s.io/code-generator/cmd/go-to-protobuf/main.go \
  --proto-import vendor \
  --packages "carvel.dev/vendir/pkg/vendir/versions/v1alpha1" \
  --go-header-file ./hack/gen-boilerplate.txt

# TODO It seems that above command messes around with protos in vendor directory
git checkout vendor/
