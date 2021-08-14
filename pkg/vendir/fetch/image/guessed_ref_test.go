// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	ctlimg "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/image"
)

func TestGuessedRefParts(t *testing.T) {
	assert.Equal(t,
		ctlimg.GuessedRefParts{Repo: "localhost", Tag: "8080"},
		ctlimg.NewGuessedRefParts("localhost:8080"))
	assert.Equal(t,
		ctlimg.GuessedRefParts{Repo: "localhost:8080/image", Tag: "tag"},
		ctlimg.NewGuessedRefParts("localhost:8080/image:tag"))
	assert.Equal(t,
		ctlimg.GuessedRefParts{Repo: "foo", Tag: "tag"},
		ctlimg.NewGuessedRefParts("foo:tag"))
	assert.Equal(t,
		ctlimg.GuessedRefParts{Repo: "docker.io/foo", Tag: "tag-0.1.1"},
		ctlimg.NewGuessedRefParts("docker.io/foo:tag-0.1.1"))
	assert.Equal(t,
		ctlimg.GuessedRefParts{Repo: "docker.@io/foo", Tag: "tag"},
		ctlimg.NewGuessedRefParts("docker.@io/foo:tag"))
	assert.Equal(t,
		ctlimg.GuessedRefParts{Repo: "docker.io/foo", Tag: "tag", Digest: "sha256:abc"},
		ctlimg.NewGuessedRefParts("docker.io/foo:tag@sha256:abc"))
	assert.Equal(t,
		ctlimg.GuessedRefParts{Repo: "docker.io/foo", Tag: "", Digest: "sha256:abc"},
		ctlimg.NewGuessedRefParts("docker.io/foo@sha256:abc"))
}
