// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package githubrelease

import (
	"testing"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	"github.com/stretchr/testify/require"
)

func TestURLEscapesTag(t *testing.T) {
	for tag, expected := range map[string]string{
		"v1.2.3":           "v1.2.3",
		"sf/v1.10.0":       "sf%2Fv1.10.0",
		"component#v1.2.3": "component%23v1.2.3",
		"50%/v1.0.0":       "50%25%2Fv1.0.0",
	} {
		sync := Sync{opts: ctlconf.DirectoryContentsGithubRelease{Slug: "owner/repo", Tag: tag}}
		url, err := sync.url()
		require.NoError(t, err)
		require.Equal(t, "https://api.github.com/repos/owner/repo/releases/tags/"+expected, url)
	}
}
