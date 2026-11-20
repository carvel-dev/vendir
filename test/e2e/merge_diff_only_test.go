// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"carvel.dev/vendir/pkg/vendir/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	vendorPath = "vendor"
	inlinePath = "vendor/inline"
	manualPath = "vendor/manual"

	manualFilePath = "vendor/manual/kept.txt"
	inlineFilePath = "vendor/inline/bar.yml"

	manualFileContents = "manually managed"
)

const (
	permsOwnerOnly     os.FileMode = 0700
	permsOwnerFile     os.FileMode = 0600
	permsOthersRead    os.FileMode = 0744
	permsOthersReadDir os.FileMode = 0755
	permsGroupReadDir  os.FileMode = 0750
	permsTraverseOnly  os.FileMode = 0100
)

// TestMergeDiffOnlyLeavesUpToDateContentsUntouched asserts that a resync which
// has nothing new to offer does not rewrite what is already there
func TestMergeDiffOnlyLeavesUpToDateContentsUntouched(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	tmpDir := setUpMergeDiffOnlyDir(t, mergeDiffOnlyConfig())

	syncMergeDiffOnly(t, vendir, tmpDir)
	firstSync := modTimes(t, tmpDir, vendorPath, inlinePath, manualPath, inlineFilePath, manualFilePath)

	syncMergeDiffOnly(t, vendir, tmpDir)
	secondSync := modTimes(t, tmpDir, vendorPath, inlinePath, manualPath, inlineFilePath, manualFilePath)

	assert.Equal(t, firstSync, secondSync,
		"expected an up to date sync to leave every path untouched")

	assertFileContents(t, tmpDir, manualFilePath, manualFileContents)
	assertFileContents(t, tmpDir, inlineFilePath, "bar")
}

// TestMergeDiffOnlyWritesChangedContents asserts that leaving up to date files
// alone does not keep the changed ones from being written
func TestMergeDiffOnlyWritesChangedContents(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	cfg := mergeDiffOnlyConfig()
	tmpDir := setUpMergeDiffOnlyDir(t, cfg)

	syncMergeDiffOnly(t, vendir, tmpDir)
	firstSync := modTimes(t, tmpDir, inlineFilePath, manualFilePath)

	cfg.Directories[0].Contents[0].Inline.Paths["bar.yml"] = "bar-updated"
	writeConfigFile(t, tmpDir, *cfg)

	syncMergeDiffOnly(t, vendir, tmpDir)
	secondSync := modTimes(t, tmpDir, inlineFilePath, manualFilePath)

	assert.NotEqual(t, firstSync[inlineFilePath], secondSync[inlineFilePath],
		"expected the changed inline file to be written again")
	assert.Equal(t, firstSync[manualFilePath], secondSync[manualFilePath],
		"expected the manual contents to stay untouched")

	assertFileContents(t, tmpDir, inlineFilePath, "bar-updated")
	assertFileContents(t, tmpDir, manualFilePath, manualFileContents)
}

// TestMergeDiffOnlyPreservesManualContents asserts that contents vendir does
// not manage survive a sync that removes stale entries around them
func TestMergeDiffOnlyPreservesManualContents(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	tmpDir := setUpMergeDiffOnlyDir(t, mergeDiffOnlyConfig())

	syncMergeDiffOnly(t, vendir, tmpDir)

	// something vendir does not know about, next to the managed contents
	stalePath := filepath.Join(tmpDir, vendorPath, "stale")
	require.NoError(t, os.MkdirAll(stalePath, permsOwnerOnly), "creating stale dir")

	// and something the user added to the manual contents in the meantime
	addedPath := filepath.Join(tmpDir, manualPath, "added.txt")
	require.NoError(t, os.WriteFile(addedPath, []byte("added"), permsOwnerFile), "writing added file")

	syncMergeDiffOnly(t, vendir, tmpDir)

	assert.NoDirExists(t, stalePath, "expected the stale dir to be dropped")
	assertFileContents(t, tmpDir, manualFilePath, manualFileContents)
	assertFileContents(t, tmpDir, filepath.Join(manualPath, "added.txt"), "added")
}

// TestMergeDiffOnlyAppliesPermissions asserts that permissions still land on
// the final location, including on contents vendir does not manage and on
// contents whose permissions keep them from being read back
func TestMergeDiffOnlyAppliesPermissions(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	tCases := map[string]struct {
		updateConfig  func(cfg *config.Config)
		expectedPerms filePerms
	}{
		"no permissions defined": {
			expectedPerms: filePerms{vendorPath: permsOwnerOnly, inlinePath: permsOwnerOnly, manualPath: permsOwnerOnly},
		},
		"outer dir permissions apply to manual contents too": {
			updateConfig: func(c *config.Config) {
				c.Directories[0].Permissions = p(permsOthersRead)
			},
			expectedPerms: filePerms{vendorPath: permsOthersRead, inlinePath: permsOthersRead, manualPath: permsOthersRead},
		},
		"inner dir permissions can be configured": {
			updateConfig: func(c *config.Config) {
				c.Directories[0].Contents[0].Permissions = p(permsOthersReadDir)
				c.Directories[0].Contents[1].Permissions = p(permsGroupReadDir)
			},
			expectedPerms: filePerms{vendorPath: permsOwnerOnly, inlinePath: permsOthersReadDir, manualPath: permsGroupReadDir},
		},
		"blocking reads in inner dirs still works": {
			updateConfig: func(c *config.Config) {
				c.Directories[0].Contents[0].Permissions = p(permsTraverseOnly)
				c.Directories[0].Contents[1].Permissions = p(permsTraverseOnly)
			},
			expectedPerms: filePerms{vendorPath: permsOwnerOnly, inlinePath: permsTraverseOnly, manualPath: permsTraverseOnly},
		},
	}

	for tName, tCase := range tCases {
		t.Run(tName, func(t *testing.T) {
			cfg := mergeDiffOnlyConfig()

			if u := tCase.updateConfig; u != nil {
				u(cfg)
			}

			tmpDir := setUpMergeDiffOnlyDir(t, cfg)
			syncMergeDiffOnly(t, vendir, tmpDir)

			actual := filePerms{}
			for path := range tCase.expectedPerms {
				actual[path] = getPerms(t, filepath.Join(tmpDir, path))
			}

			tCase.expectedPerms.validate(t, actual)
		})
	}
}

// TestSyncWithoutMergeDiffOnlyReplacesDirectory asserts that the default keeps
// swapping the whole directory over, staging the manual contents along the way
func TestSyncWithoutMergeDiffOnlyReplacesDirectory(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	cfg := mergeDiffOnlyConfig()
	cfg.Directories[0].Contents[0].Permissions = p(permsOthersReadDir)
	cfg.Directories[0].Contents[1].Permissions = p(permsGroupReadDir)

	tmpDir := setUpMergeDiffOnlyDir(t, cfg)

	syncDefault(t, vendir, tmpDir)
	firstSync := modTimes(t, tmpDir, inlineFilePath)

	// the manual contents have to survive the round trip through staging
	assertFileContents(t, tmpDir, manualFilePath, manualFileContents)

	syncDefault(t, vendir, tmpDir)
	secondSync := modTimes(t, tmpDir, inlineFilePath)

	assert.NotEqual(t, firstSync[inlineFilePath], secondSync[inlineFilePath],
		"expected the default to write every file again")

	assertFileContents(t, tmpDir, manualFilePath, manualFileContents)

	filePerms{inlinePath: permsOthersReadDir, manualPath: permsGroupReadDir}.validate(t, filePerms{
		inlinePath: getPerms(t, filepath.Join(tmpDir, inlinePath)),
		manualPath: getPerms(t, filepath.Join(tmpDir, manualPath)),
	})
}

// mergeDiffOnlyConfig returns a config with one vendir managed content and one
// manually managed one, so that both branches are exercised side by side
func mergeDiffOnlyConfig() *config.Config {
	return &config.Config{
		APIVersion: "vendir.k14s.io/v1alpha1",
		Kind:       "Config",
		Directories: []config.Directory{
			{
				Path: vendorPath,
				Contents: []config.DirectoryContents{
					{
						Path: "inline",
						Inline: &config.DirectoryContentsInline{
							Paths: map[string]string{"bar.yml": "bar"},
						},
					},
					{
						Path:   "manual",
						Manual: &config.DirectoryContentsManual{},
					},
				},
			},
		},
	}
}

// setUpMergeDiffOnlyDir returns a directory holding the given config plus the
// manually managed contents it refers to
func setUpMergeDiffOnlyDir(t *testing.T, cfg *config.Config) string {
	tmpDir := t.TempDir()

	writeConfigFile(t, tmpDir, *cfg)

	err := os.MkdirAll(filepath.Join(tmpDir, manualPath), permsOwnerOnly)
	require.NoError(t, err, "creating manual contents dir")

	err = os.WriteFile(filepath.Join(tmpDir, manualFilePath),
		[]byte(manualFileContents), permsOwnerFile)
	require.NoError(t, err, "writing manual contents")

	// configured permissions may keep the tmp dir from being cleaned up again
	t.Cleanup(func() { chmodTreeAccessible(filepath.Join(tmpDir, vendorPath)) })

	return tmpDir
}

// chmodTreeAccessible makes path and every dir below it readable again. Each
// dir is chmodded before it is read, so that restrictive permissions do not
// keep the tree from being descended into.
func chmodTreeAccessible(path string) {
	os.Chmod(path, permsOwnerOnly)

	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			chmodTreeAccessible(filepath.Join(path, entry.Name()))
		}
	}
}

func syncDefault(t *testing.T, vendir Vendir, dir string) {
	t.Helper()
	_, err := vendir.RunWithOpts([]string{"sync"}, RunOpts{Dir: dir, AllowError: true})
	require.NoError(t, err, "running vendir sync")
}

func syncMergeDiffOnly(t *testing.T, vendir Vendir, dir string) {
	t.Helper()
	_, err := vendir.RunWithOpts([]string{"sync", "--merge-diff-only"},
		RunOpts{Dir: dir, AllowError: true})
	require.NoError(t, err, "running vendir sync --merge-diff-only")
}

// modTimes maps the given paths, relative to dir, to their modification time
func modTimes(t *testing.T, dir string, paths ...string) map[string]time.Time {
	t.Helper()

	times := map[string]time.Time{}
	for _, path := range paths {
		stat, err := os.Stat(filepath.Join(dir, path))
		require.NoError(t, err, "getting stats for %s", path)
		times[path] = stat.ModTime()
	}

	return times
}

func assertFileContents(t *testing.T, dir, path, expected string) {
	t.Helper()

	bytes, err := os.ReadFile(filepath.Join(dir, path))
	require.NoError(t, err, "reading %s", path)
	assert.Equal(t, expected, string(bytes), "contents of %s", path)
}
