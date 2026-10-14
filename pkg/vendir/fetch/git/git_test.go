// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package git_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	"carvel.dev/vendir/pkg/vendir/config"
	"carvel.dev/vendir/pkg/vendir/fetch"
	"carvel.dev/vendir/pkg/vendir/fetch/git"
	"github.com/stretchr/testify/assert"
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

	t.Run("Named tag ref uses targeted fetch with --no-tags", func(t *testing.T) {
		runner := &cmdRunnerLocal{commandsToRun: [][]string{}}
		gitRetriever := git.NewGitWithRunner(config.DirectoryContentsGit{
			URL: "https://some.git/repo",
			Ref: "v1.2.3",
		}, os.Stdout, &fetch.SingleSecretRefFetcher{}, runner)
		_, err := gitRetriever.Retrieve("", &tmpFolder{t}, "")
		require.NoError(t, err)

		fetchArgs := findCommandArgs(runner.commandsToRun, "fetch")
		require.NotNil(t, fetchArgs, "expected a fetch command")
		assert.Contains(t, fetchArgs, "v1.2.3", "expected specific ref in fetch")
		assert.Contains(t, fetchArgs, "--no-tags", "expected --no-tags for named ref fetch")

		tagOptArgs := findConfigArgs(runner.commandsToRun, "remote.origin.tagOpt")
		require.NotNil(t, tagOptArgs, "expected tagOpt config command")
		assert.Equal(t, "--no-tags", tagOptArgs[len(tagOptArgs)-1], "expected --no-tags tagOpt for named ref")

		checkoutArgs := findCheckoutArgs(runner.commandsToRun)
		require.NotNil(t, checkoutArgs, "expected a checkout command")
		assert.Contains(t, checkoutArgs, "FETCH_HEAD", "expected FETCH_HEAD checkout for targeted fetch")
	})

	t.Run("origin/ branch ref uses targeted fetch with --no-tags", func(t *testing.T) {
		runner := &cmdRunnerLocal{commandsToRun: [][]string{}}
		gitRetriever := git.NewGitWithRunner(config.DirectoryContentsGit{
			URL: "https://some.git/repo",
			Ref: "origin/main",
		}, os.Stdout, &fetch.SingleSecretRefFetcher{}, runner)
		_, err := gitRetriever.Retrieve("", &tmpFolder{t}, "")
		require.NoError(t, err)

		fetchArgs := findCommandArgs(runner.commandsToRun, "fetch")
		require.NotNil(t, fetchArgs, "expected a fetch command")
		assert.Contains(t, fetchArgs, "main", "expected branch name in fetch for origin/ ref")
		assert.Contains(t, fetchArgs, "--no-tags", "expected --no-tags on the refspec for origin/ branch")

		checkoutArgs := findCheckoutArgs(runner.commandsToRun)
		require.NotNil(t, checkoutArgs, "expected a checkout command")
		assert.Contains(t, checkoutArgs, "FETCH_HEAD", "expected FETCH_HEAD checkout for targeted fetch")
	})

	t.Run("Locked SHA with OriginalRef uses OriginalRef for targeted fetch", func(t *testing.T) {
		lockedSHA := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
		runner := &cmdRunnerLocal{
			commandsToRun: [][]string{},
			// rev-parse FETCH_HEAD^{commit} must return the locked SHA for verification to pass
			responses: map[string]string{"rev-parse FETCH_HEAD^{commit}": lockedSHA},
		}
		gitRetriever := git.NewGitWithRunner(config.DirectoryContentsGit{
			URL:         "https://some.git/repo",
			Ref:         lockedSHA,
			OriginalRef: "v1.2.3",
		}, os.Stdout, &fetch.SingleSecretRefFetcher{}, runner)
		_, err := gitRetriever.Retrieve("", &tmpFolder{t}, "")
		require.NoError(t, err)

		fetchArgs := findCommandArgs(runner.commandsToRun, "fetch")
		require.NotNil(t, fetchArgs, "expected a fetch command")
		assert.Contains(t, fetchArgs, "v1.2.3", "expected OriginalRef in fetch for locked SHA mode")
		assert.Contains(t, fetchArgs, "--no-tags", "expected --no-tags for locked SHA with OriginalRef")

		tagOptArgs := findConfigArgs(runner.commandsToRun, "remote.origin.tagOpt")
		require.NotNil(t, tagOptArgs)
		assert.Equal(t, "--no-tags", tagOptArgs[len(tagOptArgs)-1], "expected --no-tags tagOpt for locked SHA with OriginalRef")

		// Checkout must use the locked SHA directly, not FETCH_HEAD, so mutable
		// refs (e.g. origin/main) don't cause locked sync to track the branch tip.
		checkoutArgs := findCheckoutArgs(runner.commandsToRun)
		require.NotNil(t, checkoutArgs)
		assert.Contains(t, checkoutArgs, lockedSHA, "expected locked SHA checkout in locked SHA mode")
		assert.NotContains(t, checkoutArgs, "FETCH_HEAD", "expected locked SHA, not FETCH_HEAD, for checkout")

		// Verification: rev-parse FETCH_HEAD^{commit} must be called before checkout
		revParseArgs := findCommandArgs(runner.commandsToRun, "rev-parse")
		require.NotNil(t, revParseArgs, "expected rev-parse for SHA verification")
		assert.Contains(t, revParseArgs, "FETCH_HEAD^{commit}")
	})

	t.Run("Locked SHA with origin/ OriginalRef strips prefix before fetch", func(t *testing.T) {
		lockedSHA := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
		runner := &cmdRunnerLocal{
			commandsToRun: [][]string{},
			responses:     map[string]string{"rev-parse FETCH_HEAD^{commit}": lockedSHA},
		}
		gitRetriever := git.NewGitWithRunner(config.DirectoryContentsGit{
			URL:         "https://some.git/repo",
			Ref:         lockedSHA,
			OriginalRef: "origin/main", // common in vendir configs
		}, os.Stdout, &fetch.SingleSecretRefFetcher{}, runner)
		_, err := gitRetriever.Retrieve("", &tmpFolder{t}, "")
		require.NoError(t, err)

		fetchArgs := findCommandArgs(runner.commandsToRun, "fetch")
		require.NotNil(t, fetchArgs)
		assert.Contains(t, fetchArgs, "main", "expected origin/ prefix stripped from OriginalRef")
		assert.NotContains(t, fetchArgs, "origin/main", "expected origin/ prefix stripped before fetch refspec")
	})

	t.Run("Plain SHA ref without OriginalRef uses full fetch with --tags", func(t *testing.T) {
		runner := &cmdRunnerLocal{commandsToRun: [][]string{}}
		gitRetriever := git.NewGitWithRunner(config.DirectoryContentsGit{
			URL: "https://some.git/repo",
			Ref: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", // 40-char hex SHA, no OriginalRef
		}, os.Stdout, &fetch.SingleSecretRefFetcher{}, runner)
		_, err := gitRetriever.Retrieve("", &tmpFolder{t}, "")
		require.NoError(t, err)

		fetchArgs := findCommandArgs(runner.commandsToRun, "fetch")
		require.NotNil(t, fetchArgs, "expected a fetch command")
		// full fetch: only "fetch" and "origin", no extra refspec
		assert.Equal(t, []string{"fetch", "origin"}, fetchArgs, "expected bare full fetch for plain SHA")

		tagOptArgs := findConfigArgs(runner.commandsToRun, "remote.origin.tagOpt")
		require.NotNil(t, tagOptArgs)
		assert.Equal(t, "--tags", tagOptArgs[len(tagOptArgs)-1], "expected --tags tagOpt for plain SHA ref")
	})

	t.Run("Depth flag is appended after refspec for named ref targeted fetch", func(t *testing.T) {
		runner := &cmdRunnerLocal{commandsToRun: [][]string{}}
		gitRetriever := git.NewGitWithRunner(config.DirectoryContentsGit{
			URL:   "https://some.git/repo",
			Ref:   "v1.2.3",
			Depth: 1,
		}, os.Stdout, &fetch.SingleSecretRefFetcher{}, runner)
		_, err := gitRetriever.Retrieve("", &tmpFolder{t}, "")
		require.NoError(t, err)

		fetchArgs := findCommandArgs(runner.commandsToRun, "fetch")
		require.NotNil(t, fetchArgs)
		fetchStr := strings.Join(fetchArgs, " ")
		assert.Contains(t, fetchStr, "v1.2.3 --no-tags --depth 1", "expected refspec, --no-tags, then --depth in order")
	})
}

// findCommandArgs finds the first command in commandsToRun whose first arg matches firstArg.
func findCommandArgs(commands [][]string, firstArg string) []string {
	for _, args := range commands {
		if len(args) > 0 && args[0] == firstArg {
			return args
		}
	}
	return nil
}

// findCheckoutArgs finds the git checkout command (issued as "-c advice.detachedHead=false checkout ...").
func findCheckoutArgs(commands [][]string) []string {
	for _, args := range commands {
		for i, a := range args {
			if a == "checkout" {
				return args[i:]
			}
		}
	}
	return nil
}

// minConfigArgs is "config" plus at least the key being configured.
const minConfigArgs = 2

// findConfigArgs finds the git config command that sets the given key.
func findConfigArgs(commands [][]string, key string) []string {
	for _, args := range commands {
		if len(args) >= minConfigArgs && args[0] == "config" && slices.Contains(args[1:], key) {
			return args
		}
	}
	return nil
}

type cmdRunnerLocal struct {
	commandsToRun [][]string
	// responses maps "arg0 arg1 ..." to the stdout value Run should return.
	responses map[string]string
}

func (c *cmdRunnerLocal) RunMultiple(argss [][]string, _ []string, _ string) error {
	c.commandsToRun = append(c.commandsToRun, argss...)
	return nil
}

func (c *cmdRunnerLocal) Run(args []string, _ []string, _ string) (string, string, error) {
	c.commandsToRun = append(c.commandsToRun, args)
	if c.responses != nil {
		if out, ok := c.responses[strings.Join(args, " ")]; ok {
			return out, "", nil
		}
	}
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
