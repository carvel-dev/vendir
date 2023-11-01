// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"carvel.dev/vendir/pkg/vendir/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

type filePerms map[string]os.FileMode

func TestDirectoryPermissions(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	tCases := map[string]struct {
		updateConfig  func(cfg *config.Config)
		expectedPerms filePerms
	}{
		"no permissions defined": {
			expectedPerms: filePerms{
				"dir-0": 0700, filepath.Join("dir-0", "subdir-0-0"): 0700, filepath.Join("dir-0", "subdir-0-1"): 0700,
				"dir-1": 0700, filepath.Join("dir-1", "subdir-1-0"): 0700, filepath.Join("dir-1", "subdir-1-1"): 0700,
			},
		},
		"outer dir permissions can be configured": {
			updateConfig: func(c *config.Config) {
				c.Directories[0].Permissions = p(0744)
			},
			expectedPerms: filePerms{
				"dir-0": 0744, filepath.Join("dir-0", "subdir-0-0"): 0744, filepath.Join("dir-0", "subdir-0-1"): 0744,
				"dir-1": 0700, filepath.Join("dir-1", "subdir-1-0"): 0700, filepath.Join("dir-1", "subdir-1-1"): 0700,
			},
		},
		"inner dir permissions can be configured": {
			updateConfig: func(c *config.Config) {
				c.Directories[0].Contents[0].Permissions = p(0755)
				c.Directories[0].Contents[1].Permissions = p(0744)
			},
			expectedPerms: filePerms{
				"dir-0": 0700, filepath.Join("dir-0", "subdir-0-0"): 0755, filepath.Join("dir-0", "subdir-0-1"): 0744,
				"dir-1": 0700, filepath.Join("dir-1", "subdir-1-0"): 0700, filepath.Join("dir-1", "subdir-1-1"): 0700,
			},
		},
		"blocking write or execute in (sub)dirs still works": {
			updateConfig: func(c *config.Config) {
				c.Directories[0].Permissions = p(0100) // we still need exec permissions here, so that we can stat its subdirectories
				c.Directories[0].Contents[0].Permissions = p(0000)
				c.Directories[0].Contents[1].Permissions = p(0004)
			},
			expectedPerms: filePerms{
				"dir-0": 0100, filepath.Join("dir-0", "subdir-0-0"): 0000, filepath.Join("dir-0", "subdir-0-1"): 0004,
				"dir-1": 0700, filepath.Join("dir-1", "subdir-1-0"): 0700, filepath.Join("dir-1", "subdir-1-1"): 0700,
			},
		},
	}

	for tName, tCase := range tCases {
		t.Run(tName, func(t *testing.T) {
			cfg := defaultConfig()

			if u := tCase.updateConfig; u != nil {
				u(cfg)
			}

			actualPerms := runAndGetActualPerms(t, vendir, *cfg)
			tCase.expectedPerms.validate(t, actualPerms)
		})
	}
}

func p(p os.FileMode) *os.FileMode { return &p }

func (expected filePerms) validate(t *testing.T, actual filePerms) {
	for path, expectedPerms := range expected {
		actualPerms, ok := actual[path]
		assert.True(t, ok, "no actual permissions for path %s found", path)
		assert.Equal(t, expectedPerms, actualPerms, "expected permissions for '%s' to be '%s', but got '%s'", path, expectedPerms, actualPerms)
	}
}

func writeConfigFile(t *testing.T, tmpDir string, config config.Config) {
	configPath := filepath.Join(tmpDir, "vendir.yml")
	bytes, err := yaml.Marshal(config)
	require.NoError(t, err, "marshalling vendir config")
	err = os.WriteFile(configPath, bytes, 0600)
	require.NoError(t, err, "writing vendir config")
}

func runAndGetActualPerms(t *testing.T, vendir Vendir, config config.Config) filePerms {
	tmpDir, err := os.MkdirTemp("", "vendir-test-")
	require.NoError(t, err, "creating tmpdir")
	defer os.RemoveAll(tmpDir)

	writeConfigFile(t, tmpDir, config)

	_, err = vendir.RunWithOpts([]string{"sync"}, RunOpts{Dir: tmpDir, AllowError: true})
	require.NoError(t, err, "running vendir")

	paths := []string{
		"dir-0",
		filepath.Join("dir-0", "subdir-0-0"),
		filepath.Join("dir-0", "subdir-0-1"),
		"dir-1",
		filepath.Join("dir-1", "subdir-1-0"),
		filepath.Join("dir-1", "subdir-1-1"),
	}

	perms := filePerms{}

	for _, p := range paths {
		perms[p] = getPerms(t, filepath.Join(tmpDir, p))
	}

	return perms
}

func getPerms(t *testing.T, fileOrDir string) os.FileMode {
	stat, err := os.Stat(fileOrDir)
	require.NoError(t, err, "getting stats for %s", fileOrDir)
	return stat.Mode().Perm()
}

// defaultConfig returns a vendir config which configures 2 directories, with 2
// subdirectories each. We use 2 directories, so that we can assert that
// changes on one does not effect the other.
func defaultConfig() *config.Config {
	return &config.Config{
		APIVersion: "vendir.k14s.io/v1alpha1",
		Kind:       "Config",
		Directories: []config.Directory{
			{
				Path: "dir-0",
				Contents: []config.DirectoryContents{
					{Path: "subdir-0-0",
						Inline: &config.DirectoryContentsInline{
							Paths: map[string]string{"bar.yml": "bar-0-0"},
						},
					},
					{Path: "subdir-0-1",
						Inline: &config.DirectoryContentsInline{
							Paths: map[string]string{"bar.yml": "bar-0-1"},
						},
					},
				},
			},
			{
				Path: "dir-1",
				Contents: []config.DirectoryContents{
					{Path: "subdir-1-0",
						Inline: &config.DirectoryContentsInline{
							Paths: map[string]string{"bar.yml": "bar-1-0"},
						},
					},
					{Path: "subdir-1-1",
						Inline: &config.DirectoryContentsInline{
							Paths: map[string]string{"bar.yml": "bar-1-1"},
						},
					},
				},
			},
		},
	}
}
