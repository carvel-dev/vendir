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
