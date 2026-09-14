// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const gitRepoAssetDir = "git-repo"
const signedTrustedTag = "signed-trusted-tag"

func TestGitVerification(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	vendir := Vendir{t, env.BinaryPath, logger}

	gitSrcPath, err := os.MkdirTemp("", "vendir-e2e-git-verify-signed-git-repo")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(gitSrcPath)

	out, err := exec.Command("tar", "xzvf", "assets/git-repo-signed/asset.tgz", "-C", gitSrcPath).CombinedOutput()
	if err != nil {
		t.Fatalf("Unpacking git-repo-signed asset: %s (output: '%s')", err, out)
	}

	dstPath, err := os.MkdirTemp("", "vendir-e2e-git-verify-signed-dst")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dstPath)

	trustedPubKey := readFile(t, filepath.Join(gitSrcPath, "keys/trusted.pub"))

	yamlConfigWithPubKeys := func(ref string, pubKeys string) io.Reader {
		encodedPubKeys := base64.StdEncoding.EncodeToString([]byte(pubKeys))
		repoPath := filepath.Join(gitSrcPath, gitRepoAssetDir)
		return strings.NewReader(fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: git-pubs
data:
  valid.pub: "%s"
---
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: vendor
  contents:
  - path: test
    git:
      url: "%s"
      ref: "%s"
      verification:
        publicKeysSecretRef:
          name: git-pubs
`, encodedPubKeys, repoPath, ref))
	}

	yamlConfig := func(ref string) io.Reader {
		return yamlConfigWithPubKeys(ref, trustedPubKey)
	}

	logger.Section("signed trusted commit", func() {
		ref := strings.TrimSpace(readFile(t, filepath.Join(gitSrcPath, "git-meta/signed-trusted-commit.txt")))
		vendir.RunWithOpts([]string{"sync", "-f", "-"}, RunOpts{Dir: dstPath, StdinReader: yamlConfig(ref)})
	})

	logger.Section("signed trusted commit verified against stranger and trusted public keys", func() {
		ref := strings.TrimSpace(readFile(t, filepath.Join(gitSrcPath, "git-meta/signed-trusted-commit.txt")))
		strangerPubKey := readFile(t, filepath.Join(gitSrcPath, "keys/stranger.pub"))
		// trusted key is after stranger key on purpose
		config := yamlConfigWithPubKeys(ref, strangerPubKey+"\n\n"+trustedPubKey)
		vendir.RunWithOpts([]string{"sync", "-f", "-"}, RunOpts{Dir: dstPath, StdinReader: config})
	})

	logger.Section("signed trusted tag", func() {
		ref := signedTrustedTag
		vendir.RunWithOpts([]string{"sync", "-f", "-"}, RunOpts{Dir: dstPath, StdinReader: yamlConfig(ref)})
	})

	logger.Section("signed trusted tag for unsigned commit", func() {
		ref := "signed-trusted-tag-for-unsigned-commit"
		vendir.RunWithOpts([]string{"sync", "-f", "-"}, RunOpts{Dir: dstPath, StdinReader: yamlConfig(ref)})
	})

	logger.Section("signed stranger commit", func() {
		ref := strings.TrimSpace(readFile(t, filepath.Join(gitSrcPath, "git-meta/signed-stranger-commit.txt")))
		_, err := vendir.RunWithOpts([]string{"sync", "-f", "-"}, RunOpts{Dir: dstPath, StdinReader: yamlConfig(ref), AllowError: true})
		assert.Error(t, err, "Expected to err when commit is signed by stranger")
		assert.ErrorContains(t, err, "openpgp: signature made by unknown entity", "Expected err to indicate stranger signing failure")
	})

	logger.Section("signed stranger tag", func() {
		ref := "signed-stranger-tag"
		_, err := vendir.RunWithOpts([]string{"sync", "-f", "-"}, RunOpts{Dir: dstPath, StdinReader: yamlConfig(ref), AllowError: true})
		assert.Error(t, err, "Expected to err when commit is signed by stranger")
		assert.ErrorContains(t, err, "openpgp: signature made by unknown entity", "Expected err to indicate stranger signing failure")
	})

	logger.Section("unsigned commit", func() {
		ref := strings.TrimSpace(readFile(t, filepath.Join(gitSrcPath, "git-meta/unsigned-commit.txt")))
		_, err := vendir.RunWithOpts([]string{"sync", "-f", "-"}, RunOpts{Dir: dstPath, StdinReader: yamlConfig(ref), AllowError: true})
		assert.Error(t, err, "Expected to err when commit is signed by stranger")
		assert.ErrorContains(t, err, "Expected to find commit signature:", "Expected err to indicate stranger signing failure")
		assert.ErrorContains(t, err, "Expected to find section 'PGP SIGNATURE', but did not", "Expected err to indicate stranger signing failure")
	})

	logger.Section("unsigned tag", func() {
		ref := "unsigned-tag"
		_, err := vendir.RunWithOpts([]string{"sync", "-f", "-"}, RunOpts{Dir: dstPath, StdinReader: yamlConfig(ref), AllowError: true})
		assert.Error(t, err, "Expected to err when commit is signed by stranger")
		assert.ErrorContains(t, err, "Expected to find tag signature:", "Expected err to indicate stranger signing failure")
		assert.ErrorContains(t, err, "Expected to find section 'PGP SIGNATURE', but did not", "Expected err to indicate stranger signing failure")
	})

	logger.Section("clones submodule by default", func() {
		ref := "git-submodule"
		_, err := vendir.RunWithOpts([]string{"sync", "-f", "-"}, RunOpts{Dir: dstPath, StdinReader: yamlConfig(ref), AllowError: true})
		assert.NoError(t, err)

		_, err = os.Stat(filepath.Join(dstPath, "vendor", "test", "carvel-vendir"))
		assert.NoError(t, err)
	})
}

func TestGitCache(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	vendir := Vendir{t, env.BinaryPath, logger}

	tmpDir, err := os.MkdirTemp("", "vendir-e2e-git-verify-signed-git-repo")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	gitSrcPath, err := os.MkdirTemp("", "vendir-e2e-git-verify-signed-git-repo")
	require.NoError(t, err)
	defer os.RemoveAll(gitSrcPath)

	out, err := exec.Command("tar", "xzvf", "assets/git-repo-signed/asset.tgz", "-C", gitSrcPath).CombinedOutput()
	require.NoErrorf(t, err, "Unpacking git-repo-signed asset: %s (output: '%s')", err, out)

	dstPath, err := os.MkdirTemp("", "vendir-e2e-git-verify-signed-dst")
	require.NoError(t, err)
	defer os.RemoveAll(dstPath)

	trustedPubKey := readFile(t, filepath.Join(gitSrcPath, "keys/trusted.pub"))

	yamlConfigWithPubKeys := func(ref string, pubKeys string) io.Reader {
		encodedPubKeys := base64.StdEncoding.EncodeToString([]byte(pubKeys))
		repoPath := filepath.Join(gitSrcPath, gitRepoAssetDir)
		return strings.NewReader(fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: git-pubs
data:
  valid.pub: "%s"
---
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: vendor
  contents:
  - path: test
    git:
      url: "%s"
      ref: "%s"
      verification:
        publicKeysSecretRef:
          name: git-pubs
`, encodedPubKeys, repoPath, ref))
	}

	yamlConfig := func(ref string) io.Reader {
		return yamlConfigWithPubKeys(ref, trustedPubKey)
	}

	var stdout bytes.Buffer
	stdoutDec := json.NewDecoder(&stdout)

	vendir.RunWithOpts(
		[]string{"sync", "-f", "-", "--json"},
		RunOpts{
			Dir:          dstPath,
			StdinReader:  yamlConfig(signedTrustedTag),
			StdoutWriter: &stdout,
			Env: []string{
				"VENDIR_CACHE_DIR=" + tmpDir,
				"VENDIR_CACHE_MAX_SIZE=10M",
			},
		})

	var vendirOutput VendirOutput
	require.NoError(t, stdoutDec.Decode(&vendirOutput))

	for _, l := range vendirOutput.Lines {
		assert.NotContains(t, l, "unbundle")
	}

	stdout.Truncate(0)

	vendir.RunWithOpts(
		[]string{"sync", "-f", "-", "--json"},
		RunOpts{
			Dir:          dstPath,
			StdinReader:  yamlConfig(signedTrustedTag),
			StdoutWriter: &stdout,
			Env: []string{
				"VENDIR_CACHE_DIR=" + tmpDir,
				"VENDIR_CACHE_MAX_SIZE=10M",
			},
		})

	require.NoError(t, stdoutDec.Decode(&vendirOutput))

	var unbundled bool
	for _, l := range vendirOutput.Lines {
		if strings.Contains(l, "unbundle") {
			unbundled = true
		}
	}
	require.True(t, unbundled, "git did not use the cached bundle")
}

// masterHeadSHA is stable — baked into the git-repo-signed test asset tarball.
const masterHeadSHA = "f0076929d229f0819eabd2d1937bdb6aa148f19f"

const ownerReadWrite = 0600

// gitFetchOptTestSetup unpacks the shared test git repo and returns a vendir
// runner plus a yamlConfig builder pointing at it.
func gitFetchOptTestSetup(t *testing.T) (Vendir, func(ref string) string) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	gitSrcPath, err := os.MkdirTemp("", "vendir-e2e-git-fetch-opt")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(gitSrcPath))
	})

	out, err := exec.Command("tar", "xzvf", "assets/git-repo-signed/asset.tgz", "-C", gitSrcPath).CombinedOutput()
	require.NoErrorf(t, err, "Unpacking git-repo-signed asset (output: '%s')", out)

	repoPath := filepath.Join(gitSrcPath, gitRepoAssetDir)

	yamlConfig := func(ref string) string {
		return fmt.Sprintf(`
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: vendor
  contents:
  - path: test
    git:
      url: "%s"
      ref: "%s"
`, repoPath, ref)
	}

	return vendir, yamlConfig
}

func TestGitFetchOptimizationsNamedTag(t *testing.T) {
	vendir, yamlConfig := gitFetchOptTestSetup(t)

	dstPath, err := os.MkdirTemp("", "vendir-e2e-git-fetch-opt-dst")
	require.NoError(t, err)
	defer os.RemoveAll(dstPath)

	var stdout bytes.Buffer
	_, err = vendir.RunWithOpts(
		[]string{"sync", "-f", "-", "--json"},
		RunOpts{Dir: dstPath, StdinReader: strings.NewReader(yamlConfig(signedTrustedTag)), StdoutWriter: &stdout},
	)
	require.NoError(t, err)

	var vendirOutput VendirOutput
	require.NoError(t, json.NewDecoder(&stdout).Decode(&vendirOutput))

	var foundTargetedFetch, foundBareFetch bool
	for _, l := range vendirOutput.Lines {
		if strings.Contains(l, "fetch origin") &&
			strings.Contains(l, signedTrustedTag) &&
			strings.Contains(l, "--no-tags") {
			foundTargetedFetch = true
		}
		// "--> git fetch origin" followed by a depth/no refspec is the old slow path
		if strings.Contains(l, "--> git fetch origin") &&
			!strings.Contains(l, signedTrustedTag) {
			foundBareFetch = true
		}
	}
	assert.True(t, foundTargetedFetch, "expected targeted fetch line 'fetch origin signed-trusted-tag --no-tags'")
	assert.False(t, foundBareFetch, "expected no bare full fetch for named tag ref")

	_, err = os.Stat(filepath.Join(dstPath, "vendor", "test"))
	assert.NoError(t, err, "expected vendored directory to exist after sync")
}

func TestGitFetchOptimizationsLockedSync(t *testing.T) {
	vendir, yamlConfig := gitFetchOptTestSetup(t)

	dstPath, err := os.MkdirTemp("", "vendir-e2e-git-fetch-opt-locked-dst")
	require.NoError(t, err)
	defer os.RemoveAll(dstPath)

	// First sync to produce a lock file
	_, err = vendir.RunWithOpts(
		[]string{"sync", "-f", "-"},
		RunOpts{Dir: dstPath, StdinReader: strings.NewReader(yamlConfig(signedTrustedTag))},
	)
	require.NoError(t, err)

	// Locked sync: should use OriginalRef ("signed-trusted-tag") not the SHA
	var stdout bytes.Buffer
	_, err = vendir.RunWithOpts(
		[]string{"sync", "--locked", "-f", "-", "--json"},
		RunOpts{Dir: dstPath, StdinReader: strings.NewReader(yamlConfig(signedTrustedTag)), StdoutWriter: &stdout},
	)
	require.NoError(t, err)

	var vendirOutput VendirOutput
	require.NoError(t, json.NewDecoder(&stdout).Decode(&vendirOutput))

	var foundTargetedFetch bool
	for _, l := range vendirOutput.Lines {
		if strings.Contains(l, "fetch origin") &&
			strings.Contains(l, signedTrustedTag) &&
			strings.Contains(l, "--no-tags") {
			foundTargetedFetch = true
		}
	}
	assert.True(t, foundTargetedFetch, "expected locked sync to use OriginalRef for targeted fetch")
}

func TestGitFetchOptimizationsTamperedLockSHA(t *testing.T) {
	vendir, yamlConfig := gitFetchOptTestSetup(t)

	dstPath, err := os.MkdirTemp("", "vendir-e2e-git-fetch-opt-tampered-dst")
	require.NoError(t, err)
	defer os.RemoveAll(dstPath)

	// Write a vendir.yml pointing at the tag
	err = os.WriteFile(filepath.Join(dstPath, "vendir.yml"), []byte(yamlConfig(signedTrustedTag)), ownerReadWrite)
	require.NoError(t, err)

	// Write a lock file with a different (valid) SHA — masterHeadSHA ≠ the tag's SHA
	lockContent := fmt.Sprintf(`apiVersion: vendir.k14s.io/v1alpha1
kind: LockConfig
directories:
- path: vendor
  contents:
  - path: test
    git:
      sha: "%s"
      commitTitle: tampered
`, masterHeadSHA)
	err = os.WriteFile(filepath.Join(dstPath, "vendir.lock.yml"), []byte(lockContent), ownerReadWrite)
	require.NoError(t, err)

	_, err = vendir.RunWithOpts(
		[]string{"sync", "--locked"},
		RunOpts{Dir: dstPath, AllowError: true},
	)
	require.Error(t, err, "expected error when lock SHA does not match fetched commit")
	assert.Contains(t, err.Error(), "Locked SHA", "expected error to mention locked SHA")
	assert.Contains(t, err.Error(), "does not match fetched commit", "expected error to describe the mismatch")
}

func readFile(t *testing.T, path string) string {
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Reading file %s: %s", path, err)
	}
	return string(contents)
}
