// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGithubRelease(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	vendir := Vendir{t, env.BinaryPath, logger}

	dstPath, err := os.MkdirTemp("", "vendir-e2e-github-release-dst")
	require.NoError(t, err)

	defer os.RemoveAll(dstPath)

	yaml := `
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: vendor
  contents:
  # unpacks archive included in the release
  - path: github.com/cloudfoundry-incubator/eirini-release
    githubRelease:
      slug: cloudfoundry-incubator/eirini-release
      tag: v1.2.0
      checksums:
        eirini-cf.tgz: 819b37126f81ad479acc8dcd7e61e8b0e55153d8fa27aa9a04692c38d0c310fe
        eirini-uaa.tgz: efe8a498c67368fac1c46fa52c484261cbf4e78b3291cff69e660cd863342674
        eirini.tgz: b535d9434300e79d11d42acc417148d09054aa32808dab5f12264d3af59ad548
      unpackArchive:
        path: thisfiledoesnotexist.tgz
`
	yaml1 := `
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: vendor
  contents:
  # unpacks archive included in the release
  - path: github.com/cloudfoundry-incubator/eirini-release
    githubRelease:
      slug: cloudfoundry-incubator/eirini-release
      tag: v1.2.0
      checksums:
        eirini-cf.tgz: 819b37126f81ad479acc8dcd7e61e8b0e55153d8fa27aa9a04692c38d0c310fe
        eirini-uaa.tgz: efe8a498c67368fac1c46fa52c484261cbf4e78b3291cff69e660cd863342674
        eirini.tgz: b535d9434300e79d11d42acc417148d09054aa32808dab5f12264d3af59ad548
      unpackArchive:
        path: eirini.tgz
`

	expectedErr := `vendir: Error: Syncing directory 'vendor':
  Syncing directory 'github.com/cloudfoundry-incubator/eirini-release' with github release contents:
    Unpacking archive 'thisfiledoesnotexist.tgz' is not part of the github release
`

	logger.Section("sync with mismatch github release path", func() {
		_, err := vendir.RunWithOpts([]string{"sync", "-f", "-"}, RunOpts{Dir: dstPath, StdinReader: strings.NewReader(yaml), AllowError: true})
		require.Error(t, err, "Expected to err while syncing with github release")
		assert.ErrorContains(t, err, expectedErr)
	})

	logger.Section("sync again with github release", func() {
		_, err := vendir.RunWithOpts([]string{"sync", "-f", "-"}, RunOpts{Dir: dstPath, StdinReader: strings.NewReader(yaml1)})
		require.NoError(t, err)
	})
}
