// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"carvel.dev/vendir/pkg/vendir/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyLineStartsConfig(t *testing.T) {
	t.Run("Empty document is ignored", func(t *testing.T) {
		tempConfigPath := filepath.Join(t.TempDir(), "config.yml")
		configWithWhitespace := []byte(`

---
apiVersion: vendir.k14s.io/v1alpha1
kind: Config`)

		require.NoError(t, os.WriteFile(tempConfigPath, configWithWhitespace, 0666))

		_, _, _, err := config.NewConfigFromFiles([]string{tempConfigPath})
		require.NoError(t, err)
	})
}

func TestCleanPaths(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		checkFn func(config.Config)
	}{
		{
			name: "single directory and single contents",
			input: `
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: vendor/foo/..//
  contents:
    - path: bar//baz///../
      inline:
        paths:
          file.txt: File contents
`,
			checkFn: func(cfg config.Config) {
				require.Equal(t, "vendor", cfg.Directories[0].Path)
				require.Equal(t, "bar", cfg.Directories[0].Contents[0].Path)
			},
		},
		{
			name: "multiple directories and multiple contents",
			input: `
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: vendor/foo/
  contents:
  - path: bar///baz/.
    inline:
      paths:
        file.txt: File contents
  - path: lorem/ipsum
    inline:
      paths:
        file.txt: File contents
- path: vendor//../vendor/bar
  contents:
  - path: baz
    inline:
      paths:
        file.txt: File contents
`,
			checkFn: func(cfg config.Config) {
				require.Equal(t, "vendor/foo", cfg.Directories[0].Path)
				require.Equal(t, "bar/baz", cfg.Directories[0].Contents[0].Path)
				require.Equal(t, "lorem/ipsum", cfg.Directories[0].Contents[1].Path)
				require.Equal(t, "vendor/bar", cfg.Directories[1].Path)
				require.Equal(t, "baz", cfg.Directories[1].Contents[0].Path)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "config.yml")
			require.NoError(t, os.WriteFile(tmpFile, []byte(tc.input), 0600))
			cfg, _, _, err := config.NewConfigFromFiles([]string{tmpFile})
			require.NoError(t, err)
			tc.checkFn(cfg)
		})
	}
}

func TestSecretsForNewConfigFromFiles(t *testing.T) {
	t.Run("Config with single secret", func(t *testing.T) {
		tempConfigPath := filepath.Join(t.TempDir(), "config.yml")
		configWithWhitespace := []byte(`
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: "repo"
  contents:
  - path: "folder-1"
    git:
      url: git@my-git-server.com:my-user/my-repo.git
      secretRef:
        name: ssh-key-secret
      ref: origin/main
    includePaths:
    - folder-1/**/*
  - path: "folder-2"
    git:
      url: git@my-git-server.com:my-user/my-repo.git
      secretRef:
        name: ssh-key-secret
      ref: origin/main
    includePaths:
    - folder-2/**/*
---
apiVersion: v1
data:
  ssh-privatekey: FOO=
kind: Secret
metadata:
  name: ssh-key-secret
`)

		require.NoError(t, os.WriteFile(tempConfigPath, configWithWhitespace, 0666))

		_, _, _, err := config.NewConfigFromFiles([]string{tempConfigPath})
		require.NoError(t, err)
	})

	t.Run("Config with same secret", func(t *testing.T) {
		tempConfigPath := filepath.Join(t.TempDir(), "config.yml")
		configWithWhitespace := []byte(`
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: "repo"
  contents:
  - path: "folder-1"
    git:
      url: git@my-git-server.com:my-user/my-repo.git
      secretRef:
        name: ssh-key-secret
      ref: origin/main
    includePaths:
    - folder-1/**/*
  - path: "folder-2"
    git:
      url: git@my-git-server.com:my-user/my-repo.git
      secretRef:
        name: ssh-key-secret
      ref: origin/main
    includePaths:
    - folder-2/**/*
---
apiVersion: v1
data:
  ssh-privatekey: FOO=
kind: Secret
metadata:
  name: ssh-key-secret
---
apiVersion: v1
data:
  ssh-privatekey: FOO=
kind: Secret
metadata:
  name: ssh-key-secret
---
apiVersion: v1
data:
  ssh-privatekey: FOO=
kind: Secret
metadata:
  name: ssh-key-secret
`)

		require.NoError(t, os.WriteFile(tempConfigPath, configWithWhitespace, 0666))

		_, _, _, err := config.NewConfigFromFiles([]string{tempConfigPath})
		require.NoError(t, err)
	})

	t.Run("Config with multiple secret", func(t *testing.T) {
		tempConfigPath := filepath.Join(t.TempDir(), "config.yml")
		configWithWhitespace := []byte(`
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: "repo"
  contents:
  - path: "folder-1"
    git:
      url: git@my-git-server.com:my-user/my-repo.git
      secretRef:
        name: ssh-key-secret
      ref: origin/main
    includePaths:
    - folder-1/**/*
  - path: "folder-2"
    git:
      url: git@my-git-server.com:my-user/my-repo.git
      secretRef:
        name: ssh-key-secret
      ref: origin/main
    includePaths:
    - folder-2/**/*
---
apiVersion: v1
data:
  ssh-privatekey: FOO=
kind: Secret
metadata:
  name: ssh-key-secret
---
apiVersion: v1
data:
  ssh-privatekey: FOO=
kind: Secret
metadata:
  name: ssh-key-secret
---
apiVersion: v1
data:
  ssh-privatekey: FOO=
kind: Secret
metadata:
  name: ssh-key-secret
---
---
apiVersion: v1
data:
  ssh-privatekey: BAR=
kind: Secret
metadata:
  name: another-secret
---
apiVersion: v1
data:
  ssh-privatekey: BAR=
kind: Secret
metadata:
  name: another-secret
`)

		require.NoError(t, os.WriteFile(tempConfigPath, configWithWhitespace, 0666))

		_, s, _, err := config.NewConfigFromFiles([]string{tempConfigPath})
		assert.Equal(t, len(s), 2)
		require.NoError(t, err)
	})

	t.Run("Config with same secrets name but different data", func(t *testing.T) {
		tempConfigPath := filepath.Join(t.TempDir(), "config.yml")
		configWithWhitespace := []byte(`
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: "repo"
  contents:
  - path: "folder-1"
    git:
      url: git@my-git-server.com:my-user/my-repo.git
      secretRef:
        name: ssh-key-secret
      ref: origin/main
    includePaths:
    - folder-1/**/*
  - path: "folder-2"
    git:
      url: git@my-git-server.com:my-user/my-repo.git
      secretRef:
        name: ssh-key-secret
      ref: origin/main
    includePaths:
    - folder-2/**/*
---
apiVersion: v1
data:
  ssh-privatekey: FOO=
kind: Secret
metadata:
  name: ssh-key-secret
---
apiVersion: v1
data:
  ssh-privatekey: FOO=
kind: Secret
metadata:
  name: ssh-key-secret
---
apiVersion: v1
data:
  ssh-privatekey: BAR=
kind: Secret
metadata:
  name: ssh-key-secret
---
apiVersion: v1
data:
  ssh-privatekey: BAR=
kind: Secret
metadata:
  name: another-secret
---
apiVersion: v1
data:
  ssh-privatekey: BAZ=
kind: Secret
metadata:
  name: another-secret
`)

		require.NoError(t, os.WriteFile(tempConfigPath, configWithWhitespace, 0666))

		_, _, _, err := config.NewConfigFromFiles([]string{tempConfigPath})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Expected to find one secret 'ssh-key-secret', but found multiple")
	})
}
