// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestOverlappingDirPath(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	path := "../../examples/overlapping-dir"

	_, err := vendir.RunWithOpts([]string{"sync", "-f", "vendir-directory-path-overlap-1.yml"}, RunOpts{Dir: path, AllowError: true})
	require.NoError(t, err)

	expectedOutputDirs := []string{
		"dir1",
		"dir2",
	}

	filepath.WalkDir(filepath.Join(path, "vendor"), func(path string, d os.DirEntry, err error) error {
		if d.Name() == "vendor" {
			return nil
		}
		require.Contains(t, expectedOutputDirs, d.Name())
		return filepath.SkipAll
	})

	_, err = vendir.RunWithOpts([]string{"sync", "-f", "vendir-directory-path-overlap-2.yml"}, RunOpts{Dir: path, AllowError: true})
	require.NoError(t, err)

	expectedOutputDirs = []string{
		"dir1",
		"dir3",
	}

	filepath.WalkDir(filepath.Join(path, "vendor"), func(path string, d os.DirEntry, err error) error {
		if d.Name() == "vendor" {
			return nil
		}
		require.Contains(t, expectedOutputDirs, d.Name())
		return filepath.SkipAll
	})
}
