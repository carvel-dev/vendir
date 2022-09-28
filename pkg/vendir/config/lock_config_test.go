// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	"testing"
)

func TestNewLockConfigFromBytes(t *testing.T) {
	t.Run("invalid yaml returns an error", func(t *testing.T) {
		invalidYaml := "this !== valid yaml"
		_, err := config.NewLockConfigFromBytes([]byte(invalidYaml))
		require.EqualError(t, err, "Unmarshaling lock config: error unmarshaling JSON: while decoding JSON: json: cannot unmarshal string into Go value of type config.LockConfig")
	})

	t.Run("valid yaml, but not valid lock config returns an error", func(t *testing.T) {
		invalidYaml := "apiVersion: not.the.right.apiVersion"
		_, err := config.NewLockConfigFromBytes([]byte(invalidYaml))
		require.EqualError(t, err, "Validating lock config: Validating apiVersion: Unknown version (known: vendir.k14s.io/v1alpha1)")
	})
}

func TestValidate(t *testing.T) {
	t.Run("valid object returns no error", func(t *testing.T) {
		lockConfig := config.LockConfig{
			APIVersion:  "vendir.k14s.io/v1alpha1",
			Kind:        "LockConfig",
			Directories: []config.LockDirectory{},
		}
		require.NoError(t, lockConfig.Validate())
	})
	t.Run("invalid API Version returns an error", func(t *testing.T) {
		lockConfig := config.LockConfig{
			APIVersion:  "what.in.the.world.is.that.thing",
			Kind:        "LockConfig",
			Directories: []config.LockDirectory{},
		}
		require.EqualError(t, lockConfig.Validate(), "Validating apiVersion: Unknown version (known: vendir.k14s.io/v1alpha1)")
	})
	t.Run("invalid kind returns an error", func(t *testing.T) {
		lockConfig := config.LockConfig{
			APIVersion:  "vendir.k14s.io/v1alpha1",
			Kind:        "LockedConfig",
			Directories: []config.LockDirectory{},
		}
		require.EqualError(t, lockConfig.Validate(), "Validating kind: Unknown kind (known: LockConfig)")
	})
}

func TestIsEqualTo(t *testing.T) {
	gitAndDirLockConfig := config.LockConfig{
		APIVersion: "vendir.k14s.io/v1alpha1",
		Kind:       "LockConfig",
		Directories: []config.LockDirectory{
			{
				Path: "lockpath",
				Contents: []config.LockDirectoryContents{
					{
						Path: "gitpath",
						Git: &config.LockDirectoryContentsGit{
							SHA:         "mygitsha",
							Tags:        []string{"main"},
							CommitTitle: "mycommittitle",
						},
					},
					{
						Path:      "dirpath",
						Directory: &config.LockDirectoryContentsDirectory{},
					},
				},
			},
		},
	}
	sameGitAndDirLockConfig := config.LockConfig{
		APIVersion: "vendir.k14s.io/v1alpha1",
		Kind:       "LockConfig",
		Directories: []config.LockDirectory{
			{
				Path: "lockpath",
				Contents: []config.LockDirectoryContents{
					{
						Path: "gitpath",
						Git: &config.LockDirectoryContentsGit{
							SHA:         "mygitsha",
							Tags:        []string{"main"},
							CommitTitle: "mycommittitle",
						},
					},
					{
						Path:      "dirpath",
						Directory: &config.LockDirectoryContentsDirectory{},
					},
				},
			},
		},
	}
	sortedGitAndDirLockConfig := config.LockConfig{
		APIVersion: "vendir.k14s.io/v1alpha1",
		Kind:       "LockConfig",
		Directories: []config.LockDirectory{
			{
				Path: "lockpath",
				Contents: []config.LockDirectoryContents{
					{
						Path:      "dirpath",
						Directory: &config.LockDirectoryContentsDirectory{},
					},
					{
						Path: "gitpath",
						Git: &config.LockDirectoryContentsGit{
							SHA:         "mygitsha",
							Tags:        []string{"main"},
							CommitTitle: "mycommittitle",
						},
					},
				},
			},
		},
	}
	httpLockConfig := config.LockConfig{
		APIVersion: "vendir.k14s.io/v1alpha1",
		Kind:       "LockConfig",
		Directories: []config.LockDirectory{
			{
				Path: "lockpath",
				Contents: []config.LockDirectoryContents{
					{
						Path: "httppath",
						HTTP: &config.LockDirectoryContentsHTTP{},
					},
				},
			},
		},
	}

	t.Run("equal lock configs returns true", func(t *testing.T) {
		require.True(t, gitAndDirLockConfig.IsEqualTo(sameGitAndDirLockConfig))
	})

	t.Run("equal lock configs, but different ordering, returns false", func(t *testing.T) {
		require.False(t, gitAndDirLockConfig.IsEqualTo(sortedGitAndDirLockConfig))
	})

	t.Run("not equal lock configs returns false", func(t *testing.T) {
		require.False(t, gitAndDirLockConfig.IsEqualTo(httpLockConfig))
	})
}
