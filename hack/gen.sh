#!/bin/bash

set -e

# Note if you are not seeing generated code, most likely it's being placed into a different folder
# (e.g. Do you have GOPATH directory structure correctly named for this project?)

rm -rf pkg/client

gen_groups_path=./vendor/k8s.io/code-generator/generate-groups.sh

chmod +x $gen_groups_path

$gen_groups_path \
	deepcopy ""  github.com/vmware-tanzu/carvel-vendir/pkg/vendir versions:v1alpha1 \
	--go-header-file ./hack/gen-boilerplate.txt

chmod -x $gen_groups_path

