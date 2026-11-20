// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"carvel.dev/vendir/pkg/vendir/config"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

// TestSymlinkWithIncludePaths verifies that a sync with includePaths succeeds even when
// the filtered content contains symlinks pointing to paths that are excluded by includePaths.
// The dangling symlinks must be silently removed rather than causing an error.
func TestSymlinkWithIncludePaths(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	tmpDir, err := os.MkdirTemp("", "vendir-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	// Source layout:
	//   src/
	//     dir1/
	//       kept.txt        <- included by includePaths
	//       link_to_other   -> ../dir2/other.txt  (will dangle after filtering)
	//     dir2/
	//       other.txt       <- NOT included by includePaths
	srcDir := filepath.Join(tmpDir, "src")
	dir1 := filepath.Join(srcDir, "dir1")
	dir2 := filepath.Join(srcDir, "dir2")
	require.NoError(t, os.MkdirAll(dir1, os.ModePerm))
	require.NoError(t, os.MkdirAll(dir2, os.ModePerm))

	keptFile, err := os.Create(filepath.Join(dir1, "kept.txt"))
	require.NoError(t, err)
	keptFile.Close()

	otherFile, err := os.Create(filepath.Join(dir2, "other.txt"))
	require.NoError(t, err)
	otherFile.Close()

	// Symlink uses a relative path (../dir2/other.txt) so it is internal to the bundle
	require.NoError(t, os.Symlink("../dir2/other.txt", filepath.Join(dir1, "link_to_other")))

	cfg := config.Config{
		APIVersion: "vendir.k14s.io/v1alpha1",
		Kind:       "Config",
		Directories: []config.Directory{{
			Path: "result",
			Contents: []config.DirectoryContents{{
				Path: "out",
				Directory: &config.DirectoryContentsDirectory{
					Path: "src",
				},
				IncludePaths: []string{"dir1/**"},
			}},
		}},
	}

	cfgBytes, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "vendir.yml"), cfgBytes, 0666))

	_, err = vendir.RunWithOpts([]string{"sync"}, RunOpts{Dir: tmpDir, AllowError: true})
	require.NoError(t, err, "sync should succeed even though includePaths causes a dangling symlink")

	// The dangling symlink must have been removed
	_, err = os.Lstat(filepath.Join(tmpDir, "result", "out", "dir1", "link_to_other"))
	require.True(t, os.IsNotExist(err), "dangling symlink should have been removed after filtering")

	// The kept file must still be present
	_, err = os.Stat(filepath.Join(tmpDir, "result", "out", "dir1", "kept.txt"))
	require.NoError(t, err, "kept.txt should be present after filtering")
}

func TestInvalidSymlink(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	tmpDir, err := os.MkdirTemp("", "vendir-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	symlinkDir := filepath.Join(tmpDir, "symlink-dir")
	err = os.Mkdir(symlinkDir, os.ModePerm)
	require.NoError(t, err)

	// valid since it is in the symlink-dir
	validFilePath := filepath.Join(symlinkDir, "a_valid_file.txt")
	validFile, err := os.Create(validFilePath)
	require.NoError(t, err)
	validFile.Close()

	//invalid since it is outside the symlink-dir
	invalidFilePath := filepath.Join(tmpDir, "invalid_file.txt")
	invalidFile, err := os.Create(invalidFilePath)
	require.NoError(t, err)
	invalidFile.Close()

	baseCfg := config.Config{
		APIVersion: "vendir.k14s.io/v1alpha1",
		Kind:       "Config",
		Directories: []config.Directory{{
			Path: "result",
			Contents: []config.DirectoryContents{{
				Path: "bad",
				Directory: &config.DirectoryContentsDirectory{
					Path: "symlink-dir",
				},
			}},
		}},
	}

	baseCfgBytes, err := yaml.Marshal(baseCfg)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "vendir.yml"), baseCfgBytes, 0666)
	require.NoError(t, err)

	tests := []struct {
		description     string
		symlinkLocation string
		valid           bool
		expectedErr     string
	}{
		{description: "valid symlink", symlinkLocation: "a_valid_file.txt", valid: true},
		{description: "symlink to outside the parent directory", symlinkLocation: invalidFilePath, valid: false, expectedErr: "Invalid symlink found to outside parent directory"},
		{description: "symlink target does not exist", symlinkLocation: "non_existent_file.txt", valid: false, expectedErr: "Unable to resolve symlink"},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			symlinkPath := filepath.Join(symlinkDir, "file")
			err = os.Symlink(tc.symlinkLocation, symlinkPath)
			require.NoError(t, err)
			defer os.Remove(symlinkPath)

			_, err = vendir.RunWithOpts([]string{"sync"}, RunOpts{Dir: tmpDir, AllowError: true})
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr)
			}
		})
	}
}
