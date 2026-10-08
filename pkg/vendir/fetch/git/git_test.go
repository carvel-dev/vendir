// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package git_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"carvel.dev/vendir/pkg/vendir/config"
	"carvel.dev/vendir/pkg/vendir/fetch"
	"carvel.dev/vendir/pkg/vendir/fetch/git"
	"github.com/stretchr/testify/require"
)

func TestGit_Retrieve(t *testing.T) {
	t.Run("Force basic auth header when flag is enabled and the user/pass is provided", func(t *testing.T) {
		secretFetcher := &fetch.SingleSecretRefFetcher{Secret: &config.Secret{
			Metadata: config.GenericMetadata{
				Name: "some-secret",
			},
			Data: map[string][]byte{
				"username": []byte("YWRtaW4="),     // admin
				"password": []byte("cGFzc3dvcmQ="), // password
			},
		}}
		runner := &cmdRunnerLocal{commandsToRun: [][]string{}}
		gitRetriever := git.NewGitWithRunner(config.DirectoryContentsGit{
			URL:                "https://some.git/repo",
			Ref:                "origin/main",
			SecretRef:          &config.DirectoryContentsLocalRef{Name: "some-secret"},
			ForceHTTPBasicAuth: true,
		}, os.Stdout, secretFetcher, runner)
		_, err := gitRetriever.Retrieve("", &tmpFolder{t}, "")
		require.NoError(t, err)
		isPresent := false
		// Check that the header was added with the correct values
		for _, args := range runner.commandsToRun {
			if args[0] == "config" && args[1] == "--add" {
				isPresent = true
				require.Equal(t, "config --add http.extraHeader Authorization: Basic WVdSdGFXND06Y0dGemMzZHZjbVE9", strings.Join(args, " "))
			}
		}
		require.True(t, isPresent, "could not find the configuration")
	})

	t.Run("Errors when authenticating with user/pass on http URL", func(t *testing.T) {
		secretFetcher := &fetch.SingleSecretRefFetcher{Secret: &config.Secret{
			Metadata: config.GenericMetadata{
				Name: "some-secret",
			},
			Data: map[string][]byte{
				"username": []byte("YWRtaW4="),     // admin
				"password": []byte("cGFzc3dvcmQ="), // password
			},
		}}
		runner := &cmdRunnerLocal{commandsToRun: [][]string{}}
		gitRetriever := git.NewGitWithRunner(config.DirectoryContentsGit{
			URL:                "http://some.git/repo",
			Ref:                "origin/main",
			SecretRef:          &config.DirectoryContentsLocalRef{Name: "some-secret"},
			ForceHTTPBasicAuth: true,
		}, os.Stdout, secretFetcher, runner)
		_, err := gitRetriever.Retrieve("", &tmpFolder{t}, "")
		require.ErrorContains(t, err, "Username/password authentication is only supported for https remotes")
	})

	t.Run("Records the configured tag when other tags point at the same commit", func(t *testing.T) {
		upstream := t.TempDir()
		runGit := func(args ...string) {
			cmd := exec.Command("git", args...)
			cmd.Dir = upstream
			cmd.Env = append(os.Environ(),
				"GIT_AUTHOR_NAME=vendir", "GIT_AUTHOR_EMAIL=vendir@example.com",
				"GIT_COMMITTER_NAME=vendir", "GIT_COMMITTER_EMAIL=vendir@example.com")
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, string(out))
		}
		runGit("init", "-q")
		runGit("commit", "-q", "--allow-empty", "-m", "release")
		// "stable" sorts before "v3.5.3", so git describe returns "stable"
		runGit("tag", "v3.5.3")
		runGit("tag", "stable")

		gitRetriever := git.NewGit(config.DirectoryContentsGit{
			URL: upstream,
			Ref: "v3.5.3",
		}, os.Stdout, &fetch.SingleSecretRefFetcher{})
		info, err := gitRetriever.Retrieve(t.TempDir(), &tmpFolder{t}, "")
		require.NoError(t, err)
		require.Equal(t, []string{"v3.5.3"}, info.Tags)
	})
}

type cmdRunnerLocal struct {
	commandsToRun [][]string
}

func (c *cmdRunnerLocal) RunMultiple(argss [][]string, _ []string, _ string) error {
	c.commandsToRun = append(c.commandsToRun, argss...)
	return nil
}

func (c *cmdRunnerLocal) Run(args []string, _ []string, _ string) (string, string, error) {
	c.commandsToRun = append(c.commandsToRun, args)
	return "", "", nil
}

type tmpFolder struct {
	t *testing.T
}

func (t tmpFolder) NewTempDir(_ string) (string, error) {
	return t.t.TempDir(), nil
}

func (t tmpFolder) NewTempFile(_ string) (*os.File, error) {
	panic("Not implemented")
}
