// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package inline_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	ctlfetch "carvel.dev/vendir/pkg/vendir/fetch"
	"carvel.dev/vendir/pkg/vendir/fetch/inline"
	"github.com/stretchr/testify/require"
)

type fakeRefFetcher struct{}

func (fakeRefFetcher) GetSecret(string) (ctlconf.Secret, error) {
	return ctlconf.Secret{Data: map[string][]byte{"secret.sh": []byte("#!/bin/sh\n")}}, nil
}

func (fakeRefFetcher) GetConfigMap(string) (ctlconf.ConfigMap, error) {
	return ctlconf.ConfigMap{Data: map[string]string{"config.sh": "#!/bin/sh\n"}}, nil
}

var _ ctlfetch.RefFetcher = fakeRefFetcher{}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	return info.Mode().Perm()
}

func TestSyncDefaultMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes are not supported on windows")
	}

	sources := []ctlconf.DirectoryContentsInlineSource{
		{SecretRef: &ctlconf.DirectoryContentsInlineSourceRef{DirectoryContentsLocalRef: ctlconf.DirectoryContentsLocalRef{Name: "s"}}},
		{ConfigMapRef: &ctlconf.DirectoryContentsInlineSourceRef{DirectoryPath: "cm", DirectoryContentsLocalRef: ctlconf.DirectoryContentsLocalRef{Name: "c"}}},
	}

	t.Run("applies to paths and pathsFrom", func(t *testing.T) {
		mode := int32(0755)
		dst := t.TempDir()
		opts := ctlconf.DirectoryContentsInline{
			Paths:       map[string]string{"script.sh": "#!/bin/sh\n", "dir/other.sh": "#!/bin/sh\n"},
			PathsFrom:   sources,
			DefaultMode: &mode,
		}

		_, err := inline.NewSync(opts, fakeRefFetcher{}).Sync(dst)
		require.NoError(t, err)

		for _, p := range []string{"script.sh", "dir/other.sh", "secret.sh", "cm/config.sh"} {
			require.Equal(t, os.FileMode(0755), fileMode(t, filepath.Join(dst, p)), p)
		}
	})

	t.Run("defaults to 0600", func(t *testing.T) {
		dst := t.TempDir()
		opts := ctlconf.DirectoryContentsInline{
			Paths:     map[string]string{"script.sh": "#!/bin/sh\n"},
			PathsFrom: sources,
		}

		_, err := inline.NewSync(opts, fakeRefFetcher{}).Sync(dst)
		require.NoError(t, err)

		for _, p := range []string{"script.sh", "secret.sh", "cm/config.sh"} {
			require.Equal(t, os.FileMode(0600), fileMode(t, filepath.Join(dst, p)), p)
		}
	})

	t.Run("ignores the umask", func(t *testing.T) {
		mode := int32(0777)
		dst := t.TempDir()
		opts := ctlconf.DirectoryContentsInline{
			Paths:       map[string]string{"script.sh": "#!/bin/sh\n"},
			DefaultMode: &mode,
		}

		_, err := inline.NewSync(opts, fakeRefFetcher{}).Sync(dst)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0777), fileMode(t, filepath.Join(dst, "script.sh")))
	})
}
