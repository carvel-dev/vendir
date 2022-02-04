#!/bin/bash

set -e -x -u

function get_latest_git_tag {
  git describe --tags | grep -Eo 'v[0-9]+\.[0-9]+\.[0-9]+'
}

VERSION="${1:-`get_latest_git_tag`}"

./hack/build.sh "$VERSION"
./hack/test.sh
./hack/test-e2e.sh

echo ALL SUCCESS
