// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
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

func TestWriteToFile(t *testing.T) {
	lockConfig := config.LockConfig{
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
	lockConfigBytes, _ := lockConfig.AsBytes()

	otherLockFile := config.LockConfig{
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
	otherLockFileBytes, _ := otherLockFile.AsBytes()

	tempDir, err := os.MkdirTemp("", "test-vendir-write-to-file")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "lockfile.yml"), lockConfigBytes, 0666))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "lockfile-copy.yml"), lockConfigBytes, 0666))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "other-lockfile.yml"), otherLockFileBytes, 0666))

	t.Run("no prior lock config file will write", func(t *testing.T) {
		lockFilePath := filepath.Join(tempDir, "new-lockfile.yml")
		require.NoError(t, lockConfig.WriteToFile(lockFilePath))
		require.FileExists(t, lockFilePath)
	})

	t.Run("pre-existing identical lock config file does not write", func(t *testing.T) {
		lockFilePath := filepath.Join(tempDir, "lockfile.yml")

		// Remember the original mod time
		beforeStats, err := os.Stat(lockFilePath)
		require.NoError(t, err)

		// Call WriteToFile, which should not update the file
		require.NoError(t, lockConfig.WriteToFile(lockFilePath))

		// Check that mod time is the same
		afterStats, err := os.Stat(lockFilePath)
		require.NoError(t, err)
		require.Equal(t, beforeStats.ModTime(), afterStats.ModTime(), "lock file was modified but it shouldn't have been")
	})

	t.Run("pre-existing but different lock config file will write", func(t *testing.T) {
		lockFilePath := filepath.Join(tempDir, "other-lockfile.yml")

		// Remember the original mod time
		beforeStats, err := os.Stat(lockFilePath)
		require.NoError(t, err)

		// Call WriteToFile, which should update the file
		require.NoError(t, lockConfig.WriteToFile(lockFilePath))

		// Check that mod time has been updated
		afterStats, err := os.Stat(lockFilePath)
		require.NoError(t, err)

		require.GreaterOrEqual(t, afterStats.ModTime(), beforeStats.ModTime(), "lock file was not modified but it should have been")

		// Check that the file contents have been updated
		updatedBytes, err := os.ReadFile(lockFilePath)
		require.NoError(t, err)
		require.Equal(t, updatedBytes, lockConfigBytes)
	})

	require.NoError(t, os.RemoveAll(tempDir))
}
