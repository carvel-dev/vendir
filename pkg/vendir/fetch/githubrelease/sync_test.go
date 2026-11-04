// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package githubrelease

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	"github.com/stretchr/testify/require"
)

type testTempArea struct {
	dir string
}

func (t testTempArea) NewTempDir(name string) (string, error) {
	return os.MkdirTemp(t.dir, name)
}

func (t testTempArea) NewTempFile(name string) (*os.File, error) {
	return os.CreateTemp(t.dir, name)
}

func TestSyncReleaseWithoutMatchingAssets(t *testing.T) {
	var assets string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := fmt.Fprintf(w, `{"url":"https://api.github.com/repos/org/repo/releases/1","tag_name":"v1.0.0","body":"","assets":[%s]}`, assets)
		require.NoError(t, err)
	}))
	defer server.Close()

	sync := func(opts ctlconf.DirectoryContentsGithubRelease) error {
		opts.URL = server.URL
		opts.DisableAutoChecksumValidation = true
		s, err := NewSync(opts, "", nil)
		require.NoError(t, err)
		tmpDir := t.TempDir()
		_, err = s.Sync(filepath.Join(tmpDir, "dst"), testTempArea{tmpDir})
		return err
	}

	t.Run("release has no assets", func(t *testing.T) {
		assets = ""
		err := sync(ctlconf.DirectoryContentsGithubRelease{})
		require.ErrorContains(t, err, "Expected release 'v1.0.0' to have at least one asset, but it has none")
	})

	t.Run("no asset matches assetNames", func(t *testing.T) {
		assets = `{"name":"tool-linux-amd64","url":"https://example.com/tool","size":1}`
		err := sync(ctlconf.DirectoryContentsGithubRelease{AssetNames: []string{"*.tgz"}})
		require.ErrorContains(t, err, "Expected at least one asset of release 'v1.0.0' to match assetNames [*.tgz]")
	})
}
