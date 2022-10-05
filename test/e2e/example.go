// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type example struct {
	Description           string
	Name                  string
	Env                   []string
	OnlyLocked            bool
	SkipRemove            bool
	VendirYamlReplaceVals []string
	EnableCaching         bool
}

func (et example) Check(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	vendir := Vendir{t, env.BinaryPath, logger}

	logger.Section(et.Name, func() {
		err := et.check(t, vendir)
		if err != nil {
			t.Fatalf("[example: %s] %s", et.Name, err)
		}
	})
}

func (et example) check(t *testing.T, vendir Vendir) error {
	tmpDir, err := os.MkdirTemp("", "vendir-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	defer os.RemoveAll(tmpDir)
	_, _, err = execGit([]string{"clone", ".", tmpDir}, "../..")
	if err != nil {
		t.Fatalf("failed to copy repo to temp dir: %v", err)
	}

	dir := "examples/" + et.Name
	path := tmpDir + "/" + dir

	if len(et.VendirYamlReplaceVals) > 0 {
		vendirYaml := filepath.Join(path, "vendir.yml")
		abs, err := filepath.Abs(vendirYaml)
		require.NoError(t, err)

		yamlContents, err := os.ReadFile(abs)
		require.NoError(t, err)
		for _, replaceVal := range et.VendirYamlReplaceVals {
			splitReplaceVal := bytes.Split([]byte(replaceVal), []byte{','})
			yamlContents = bytes.ReplaceAll(yamlContents, splitReplaceVal[0], splitReplaceVal[1])
		}

		err = os.WriteFile(abs, yamlContents, os.ModePerm)
		require.NoError(t, err)

		// Do not use --global to avoid creation of $HOME/.gitconfig
		_, _, err = execGit([]string{"config", "user.email", "you@example.com"}, tmpDir)
		require.NoError(t, err)
		_, _, err = execGit([]string{"config", "user.name", "Your Name"}, tmpDir)
		require.NoError(t, err)
		_, _, err = execGit([]string{"commit", "-am", "render vendir.yaml"}, tmpDir)
		require.NoError(t, err)
	}

	vendorPath := path + "/vendor"

	vendorDir, err := os.Stat(vendorPath)
	if err != nil {
		return fmt.Errorf("Expected no err for stat: %v", err)
	}
	if !vendorDir.IsDir() {
		return fmt.Errorf("Expected to be dir")
	}

	// remove all vendored bits
	if !et.SkipRemove {
		err = os.RemoveAll(vendorPath)
		if err != nil {
			return fmt.Errorf("Expected no err for remove all")
		}
	}

	if et.EnableCaching {
		cachingPath, err := os.MkdirTemp("", "vendir-caching-test")
		require.NoError(t, err)
		et.Env = append(et.Env, "VENDIR_CACHE_DIR="+cachingPath)
	}

	if !et.OnlyLocked {
		_, err = vendir.RunWithOpts([]string{"sync"}, RunOpts{Dir: path, Env: et.Env})
		if err != nil {
			return fmt.Errorf("Expected no err for sync")
		}

		// This assumes that example's vendor directory is committed to git
		gitOut := gitDiffExamplesDir(t, dir, "../../")
		if gitOut != "" {
			return fmt.Errorf("Expected no diff, but was: >>>%s<<<", gitOut)
		}
	}

	lockFileStat, err := os.Stat(filepath.Join(path, "vendir.lock.yml"))
	require.NoError(t, err, "Expected no err for getting lock file stats")

	_, err = vendir.RunWithOpts([]string{"sync", "--locked"}, RunOpts{Dir: path, Env: et.Env})
	if err != nil {
		return fmt.Errorf("Expected no err for sync locked")
	}

	newLockFileStat, err := os.Stat(filepath.Join(path, "vendir.lock.yml"))
	require.NoError(t, err, "Expected no err for getting new lock file stats")
	require.Equal(t, lockFileStat.ModTime(), newLockFileStat.ModTime(), "Expected lock file to not be updated")

	gitOut := gitDiffExamplesDir(t, path, tmpDir)
	if gitOut != "" {
		return fmt.Errorf("Expected no diff, but was: >>>%s<<<", gitOut)
	}

	return nil
}

func execHelm3(args []string) (string, string, error) {
	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command("helm3", args...)
	cmd.Stdout = &stdoutBs
	cmd.Stderr = &stderrBs

	err := cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("helm3 %s: %s (stderr: %s)", args, err, stderrBs.String())
	}

	return stdoutBs.String(), stderrBs.String(), nil
}
