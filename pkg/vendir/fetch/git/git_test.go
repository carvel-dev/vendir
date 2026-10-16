// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package git_test

import (
	"os"
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

	t.Run("Lowercases the host in the stored credentials URL", func(t *testing.T) {
		secretFetcher := &fetch.SingleSecretRefFetcher{Secret: &config.Secret{
			Metadata: config.GenericMetadata{Name: "some-secret"},
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("password"),
			},
		}}
		runner := &credsReadingRunner{}
		gitRetriever := git.NewGitWithRunner(config.DirectoryContentsGit{
			URL:       "https://Some.GIT.example.com/Repo",
			Ref:       "origin/main",
			SecretRef: &config.DirectoryContentsLocalRef{Name: "some-secret"},
		}, os.Stdout, secretFetcher, runner)
		_, err := gitRetriever.Retrieve("", &tmpFolder{t}, "")
		require.NoError(t, err)
		require.Equal(t, "https://admin:password@some.git.example.com\n", runner.credsFileContents)
	})

	t.Run("Normalizes remote hostname to match credentials file", func(t *testing.T) {
		secretFetcher := &fetch.SingleSecretRefFetcher{Secret: &config.Secret{
			Metadata: config.GenericMetadata{Name: "some-secret"},
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("password"),
			},
		}}
		runner := &cmdRunnerLocal{commandsToRun: [][]string{}}
		gitRetriever := git.NewGitWithRunner(config.DirectoryContentsGit{
			URL:       "https://Some.GIT.example.com/Repo",
			Ref:       "origin/main",
			SecretRef: &config.DirectoryContentsLocalRef{Name: "some-secret"},
		}, os.Stdout, secretFetcher, runner)
		_, err := gitRetriever.Retrieve("", &tmpFolder{t}, "")
		require.NoError(t, err)

		var remoteURL string
		for _, args := range runner.commandsToRun {
			if len(args) == 4 &&
				args[0] == "remote" &&
				args[1] == "add" &&
				args[2] == "origin" {
				remoteURL = args[3]
				break
			}
		}

		require.NotEmpty(t, remoteURL, "expected a git remote add command")
		require.Equal(
			t,
			"https://some.git.example.com/Repo",
			remoteURL,
			"normalize the remote hostname to match the credentials "+
				"while preserving repository path case",
		)
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

// noOutput is the empty stdout/stderr returned by the fake runner
const noOutput = ""

// credentialHelperArgCount is the number of args in "config credential.helper <value>"
const credentialHelperArgCount = 3

// credsReadingRunner captures the contents of the git credential store file
// before it is cleaned up at the end of the fetch
type credsReadingRunner struct {
	credsFileContents string
}

func (c *credsReadingRunner) RunMultiple(argss [][]string, _ []string, _ string) error {
	for _, args := range argss {
		if len(args) == credentialHelperArgCount && args[0] == "config" && args[1] == "credential.helper" {
			path := strings.TrimPrefix(args[2], "store --file ")
			bs, err := os.ReadFile(path)
			if err == nil {
				c.credsFileContents = string(bs)
			}
		}
	}
	return nil
}

func (*credsReadingRunner) Run(_ []string, _ []string, _ string) (stdout, stderr string, err error) {
	return noOutput, noOutput, nil
}
