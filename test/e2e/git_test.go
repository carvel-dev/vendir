// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitVerification(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	vendir := Vendir{t, env.BinaryPath, logger}

	gitSrcPath, err := ioutil.TempDir("", "vendir-e2e-git-verify-signed-git-repo")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(gitSrcPath)

	out, err := exec.Command("tar", "xzvf", "assets/git-repo-signed/asset.tgz", "-C", gitSrcPath).CombinedOutput()
	if err != nil {
		t.Fatalf("Unpacking git-repo-signed asset: %s (output: '%s')", err, out)
	}

	dstPath, err := ioutil.TempDir("", "vendir-e2e-git-verify-signed-dst")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dstPath)

	trustedPubKey := readFile(t, filepath.Join(gitSrcPath, "keys/trusted.pub"))

	yamlConfigWithPubKeys := func(ref string, pubKeys string) io.Reader {
		encodedPubKeys := base64.StdEncoding.EncodeToString([]byte(pubKeys))
		repoPath := filepath.Join(gitSrcPath, "git-repo")
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
		ref := "signed-trusted-tag"
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

func readFile(t *testing.T, path string) string {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("Reading file %s: %s", path, err)
	}
	return string(contents)
}
