// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	"github.com/stretchr/testify/require"
)

func TestGitRefSelectionSemverPrefix(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	vendir := Vendir{t, env.BinaryPath, logger}

	repoPath, commitAndTag := newLocalGitRepo(t)
	dstPath := t.TempDir()

	commitAndTag("sf/v1.2.0")
	expectedSHA := commitAndTag("sf/v1.10.0", "-a", "-m", "sf 1.10.0") // annotated
	commitAndTag("sf/v1.11.0-rc.1")
	commitAndTag("sf/v2.0.0")
	commitAndTag("other/v1.99.0")
	commitAndTag("v1.99.0")

	config := func() io.Reader {
		return strings.NewReader(fmt.Sprintf(`
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: vendor
  contents:
  - path: test
    git:
      url: "%s"
      refSelection:
        semver:
          prefix: "sf/"
          constraints: ">=1.0.0 <2.0.0"
`, repoPath))
	}

	sync := func(args ...string) {
		_, err := vendir.RunWithOpts(append([]string{"sync", "-f", "-"}, args...), RunOpts{Dir: dstPath, StdinReader: config()})
		require.NoError(t, err)
	}

	vendoredContent := func() string {
		return readFile(t, filepath.Join(dstPath, "vendor", "test", localGitRepoContentFile))
	}

	lockedGit := func() ctlconf.LockDirectoryContentsGit {
		lockConfig, err := ctlconf.NewLockConfigFromFile(filepath.Join(dstPath, "vendir.lock.yml"))
		require.NoError(t, err)
		require.Len(t, lockConfig.Directories, 1)
		require.Len(t, lockConfig.Directories[0].Contents, 1)
		require.NotNil(t, lockConfig.Directories[0].Contents[0].Git)
		return *lockConfig.Directories[0].Contents[0].Git
	}

	logger.Section("selects highest prefixed tag within constraints", func() {
		sync()

		require.Equal(t, "sf/v1.10.0", vendoredContent())
		lock := lockedGit()
		require.Equal(t, expectedSHA, lock.SHA)
		require.Equal(t, []string{"sf/v1.10.0"}, lock.Tags)
	})

	newerSHA := commitAndTag("sf/v1.12.0")

	logger.Section("locked sync stays pinned after newer eligible tag appears", func() {
		sync("--locked")

		require.Equal(t, "sf/v1.10.0", vendoredContent())
		require.Equal(t, expectedSHA, lockedGit().SHA)
	})

	logger.Section("unlocked sync picks up newer eligible tag", func() {
		sync()

		require.Equal(t, "sf/v1.12.0", vendoredContent())
		require.Equal(t, newerSHA, lockedGit().SHA)
	})
}

const (
	localGitRepoContentFile = "content"
	localGitRepoContentMode = 0644
)

// newLocalGitRepo initializes an empty git repository in a temporary directory and returns
// its path along with a helper that commits the tag name as file content and tags that commit.
// Extra tagArgs (e.g. "-a", "-m", "msg") are passed to git tag.
func newLocalGitRepo(t *testing.T) (string, func(tag string, tagArgs ...string) string) {
	repoPath := t.TempDir()

	git := func(args ...string) string {
		// Avoid depending on the global git identity or signing configuration
		args = append([]string{"-c", "user.name=vendir", "-c", "user.email=vendir@example.com",
			"-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)
		out, _, err := execGit(args, repoPath)
		require.NoError(t, err)
		return strings.TrimSpace(out)
	}

	git("init", "-q")

	commitAndTag := func(tag string, tagArgs ...string) string {
		require.NoError(t, os.WriteFile(filepath.Join(repoPath, localGitRepoContentFile), []byte(tag), localGitRepoContentMode))
		git("add", localGitRepoContentFile)
		git("commit", "-q", "-m", "Add "+tag)
		git(append([]string{"tag"}, append(tagArgs, tag)...)...)
		return git("rev-parse", "HEAD")
	}

	return repoPath, commitAndTag
}
