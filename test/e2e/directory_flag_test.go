// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestUseDirectory(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	dir := "examples/git-and-manual"
	path := "../../" + dir

	reset := func() {
		// Make sure state is reset
		_, err := vendir.RunWithOpts([]string{"sync"}, RunOpts{Dir: path})
		require.NoError(t, err)
	}
	checkFileContent := func(filePath string, expectedContent string) {
		file, err := os.ReadFile(path + filePath)
		require.NoError(t, err)
		require.EqualValues(t, string(file), expectedContent)
	}

	// Sync with directory flag
	_, err := vendir.RunWithOpts([]string{"sync", "--directory", "vendor/local-dir", "--directory", "vendor/local-dir-2"}, RunOpts{Dir: path})
	if err != nil {
		t.Fatalf("Expected no err")
	}
	checkFileContent("/vendor/local-dir/file.txt", "file\n")
	checkFileContent("/vendor/local-dir-2/file.txt", "file-dir-2\n")

	// Sync with directory flag and local-dir-dev
	_, err = vendir.RunWithOpts([]string{"sync", "--directory", "vendor/local-dir=local-dir-dev"}, RunOpts{Dir: path})
	require.NoError(t, err)
	checkFileContent("/vendor/local-dir/file.txt", "file-dir-dev\n")

	// Lock file check
	require.FileExists(t, path+"/vendir.lock.yml")

	// Final regular sync to make sure nothing in vendir was deleted (manual syncs would fail).
	reset()
}
