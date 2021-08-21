#!/bin/bash

set -e -x

# Note if you are not seeing generated code, most likely it's being placed into a different folder
# (e.g. Do you have GOPATH directory structure correctly named for this project?)

# Based on vendor/k8s.io/code-generator/generate-groups.sh
go run vendor/k8s.io/code-generator/cmd/deepcopy-gen/main.go \
	--input-dirs github.com/vmware-tanzu/carvel-vendir/pkg/vendir/versions/v1alpha1 \
	-O zz_generated.deepcopy \
	--bounding-dirs github.com/vmware-tanzu/carvel-vendir/pkg/vendir \
	--go-header-file ./hack/gen-boilerplate.txt

# To keep things simple for now, vendir is not including protobuf files.
# Projects (e.g. kapp-controller) including vendir should generate it
# for pkg/vendir/versions/v1alpha1 directory.
