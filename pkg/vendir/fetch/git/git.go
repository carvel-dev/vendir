// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	ctlfetch "carvel.dev/vendir/pkg/vendir/fetch"
	ctlver "carvel.dev/vendir/pkg/vendir/versions"
)

type Git struct {
	opts       ctlconf.DirectoryContentsGit
	infoLog    io.Writer
	refFetcher ctlfetch.RefFetcher
	cmdRunner  CommandRunner
}

func NewGit(opts ctlconf.DirectoryContentsGit,
	infoLog io.Writer, refFetcher ctlfetch.RefFetcher) *Git {

	return &Git{opts, infoLog, refFetcher, &runner{infoLog}}
}

// NewGitWithRunner creates a Git retriever with a provided runner
func NewGitWithRunner(opts ctlconf.DirectoryContentsGit,
	infoLog io.Writer, refFetcher ctlfetch.RefFetcher, cmdRunner CommandRunner) *Git {

	return &Git{opts, infoLog, refFetcher, cmdRunner}
}

//nolint:revive
type GitInfo struct {
	SHA         string
	Tags        []string
	CommitTitle string
}

func (t *Git) Retrieve(dstPath string, tempArea ctlfetch.TempArea, bundle string) (GitInfo, error) {
	if len(t.opts.URL) == 0 {
		return GitInfo{}, fmt.Errorf("Expected non-empty URL")
	}

	err := t.fetch(dstPath, tempArea, bundle)
	if err != nil {
		return GitInfo{}, err
	}

	info := GitInfo{}

	out, _, err := t.cmdRunner.Run([]string{"rev-parse", "HEAD"}, nil, dstPath)
	if err != nil {
		return GitInfo{}, err
	}

	info.SHA = strings.TrimSpace(out)

	out, _, err = t.cmdRunner.Run([]string{"describe", "--tags", info.SHA}, nil, dstPath)
	if err == nil {
		info.Tags = strings.Split(strings.TrimSpace(out), "\n")
	}

	out, _, err = t.cmdRunner.Run([]string{"log", "-n", "1", "--pretty=%B", info.SHA}, nil, dstPath)
	if err != nil {
		return GitInfo{}, err
	}

	info.CommitTitle = strings.TrimSpace(out)

	return info, nil
}

func (t *Git) fetch(dstPath string, tempArea ctlfetch.TempArea, bundle string) error {
	authOpts, err := t.getAuthOpts()
	if err != nil {
		return err
	}

	authDir, err := tempArea.NewTempDir("git-auth")
	if err != nil {
		return err
	}

	defer os.RemoveAll(authDir)

	env := os.Environ()

	if authOpts.IsPresent() {
		sshCmd := []string{"ssh", "-o", "ServerAliveInterval=30", "-o", "ForwardAgent=no", "-F", "/dev/null"}

		if authOpts.PrivateKey != nil {
			path := filepath.Join(authDir, "private-key")

			// Ensure the private key ends with a newline character, as git requires it to work. (https://github.com/carvel-dev/vendir/issues/350)
			err = os.WriteFile(path, []byte(*authOpts.PrivateKey+"\n"), 0600)
			if err != nil {
				return fmt.Errorf("Writing private key: %s", err)
			}

			sshCmd = append(sshCmd, "-i", path, "-o", "IdentitiesOnly=yes")
		}

		if authOpts.KnownHosts != nil {
			path := filepath.Join(authDir, "known-hosts")

			err = os.WriteFile(path, []byte(*authOpts.KnownHosts), 0600)
			if err != nil {
				return fmt.Errorf("Writing known hosts: %s", err)
			}

			sshCmd = append(sshCmd, "-o", "StrictHostKeyChecking=yes", "-o", "UserKnownHostsFile="+path)
		} else {
			sshCmd = append(sshCmd, "-o", "StrictHostKeyChecking=no")
		}

		env = append(env, "GIT_SSH_COMMAND="+strings.Join(sshCmd, " "))
	}

	if t.opts.LFSSkipSmudge {
		env = append(env, "GIT_LFS_SKIP_SMUDGE=1")
	}
	if t.opts.DangerousSkipTLSVerify {
		env = append(env, "GIT_SSL_NO_VERIFY=true")
	}
	gitURL := t.opts.URL
	gitCredsPath := filepath.Join(authDir, ".git-credentials")

	argss := [][]string{
		{"init"},
		{"config", "credential.helper", "store --file " + gitCredsPath},
		{"remote", "add", "origin", gitURL},
	}

	if authOpts.Username != nil && authOpts.Password != nil {
		if !strings.HasPrefix(gitURL, "https://") {
			return fmt.Errorf("Username/password authentication is only supported for https remotes")
		}

		if t.opts.ForceHTTPBasicAuth {
			encodedAuth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", *authOpts.Username, *authOpts.Password)))
			argss = append(argss, []string{"config", "--add", "http.extraHeader", fmt.Sprintf("Authorization: Basic %s", encodedAuth)})
		} else {
			gitCredsURL, err := url.Parse(gitURL)
			if err != nil {
				return fmt.Errorf("Parsing git remote url: %s", err)
			}

			gitCredsURL.User = url.UserPassword(*authOpts.Username, *authOpts.Password)
			gitCredsURL.Path = ""

			err = os.WriteFile(gitCredsPath, []byte(gitCredsURL.String()+"\n"), 0600)
			if err != nil {
				return fmt.Errorf("Writing %s: %s", gitCredsPath, err)
			}
		}
	}

	if bundle != "" {
		argss = append(argss, []string{"bundle", "unbundle", bundle})
	}

	err = t.cmdRunner.RunMultiple(argss, env, dstPath)
	if err != nil {
		return err
	}

	// Resolve an ambiguous ref (which may be either a branch or a tag) before
	// constructing the refspec. This lets us preserve tags without fetching
	// every tag in the repository.
	refType := remoteRefUnknown
	fetchRef := t.opts.Ref
	if t.isLockedSHAWithNamedOriginal() {
		fetchRef = t.opts.OriginalRef
	}
	if t.canTargetFetch() {
		var err error
		refType, err = t.remoteRefType(dstPath, fetchRef)
		if err != nil {
			return err
		}
	}

	// RefSelection needs all tags for semver matching, and a bare SHA has no
	// named ref to target. A targeted tag fetch explicitly creates just that
	// tag; a targeted branch fetch uses Git's default auto-follow behavior,
	// which fetches only tags reachable from the branch tip.
	if !t.canTargetFetch() || refType == remoteRefUnknown ||
		refType == remoteRefTag {
		tagOpt := "--tags"
		if t.canTargetFetch() {
			tagOpt = noTagsFlag
		}
		_, _, err = t.cmdRunner.Run(
			[]string{"config", "remote.origin.tagOpt", tagOpt}, nil, dstPath)
		if err != nil {
			return err
		}
	}

	// When fetching a single named ref, track the result in FETCH_HEAD for
	// checkout. Branches/tags also get local refs to support bundle caching.
	fetchArgs, useFetchHead := t.targetedFetchArgs(fetchRef, refType)
	if t.opts.Depth > 0 {
		fetchArgs = append(fetchArgs, "--depth", strconv.Itoa(t.opts.Depth))
	}

	_, _, err = t.cmdRunner.Run(fetchArgs, env, dstPath)
	lockedFetchByOriginal := t.isLockedSHAWithNamedOriginal() &&
		fetchRef != t.opts.Ref
	if err != nil && lockedFetchByOriginal {
		// The original ref may have been deleted or renamed since the config
		// was locked. Fetching the locked object directly retains locked-sync
		// behavior without requiring the old name to remain available.
		fallbackArgs := []string{"fetch", "origin", t.opts.Ref, noTagsFlag}
		if t.opts.Depth > 0 {
			fallbackArgs = append(
				fallbackArgs, "--depth", strconv.Itoa(t.opts.Depth),
			)
		}
		_, _, err = t.cmdRunner.Run(fallbackArgs, env, dstPath)
	}
	if err != nil {
		return err
	}

	if useFetchHead {
		_, _, err = t.cmdRunner.Run([]string{
			"update-ref", "refs/vendir/fetched", "FETCH_HEAD",
		}, nil, dstPath)
		if err != nil {
			return err
		}
	}

	ref, err := t.resolveRef(dstPath)
	if err != nil {
		return err
	}

	// In locked mode we fetch by the original named ref for speed, but must
	// check out the exact locked SHA. Verify FETCH_HEAD resolves to that SHA
	// before checkout so a force-pushed tag is caught early.
	if useFetchHead && isHexSHA(t.opts.Ref) {
		rp := []string{"rev-parse", fetchHeadCommit}
		out, _, runErr := t.cmdRunner.Run(rp, nil, dstPath)
		if runErr != nil {
			return runErr
		}
		fetchedSHA := strings.TrimSpace(out)
		if fetchedSHA != t.opts.Ref {
			return fmt.Errorf("Locked SHA %s does not match fetched "+
				"commit %s — tag may have been moved", t.opts.Ref, fetchedSHA)
		}
		// Use the locked SHA for checkout so mutable refs (e.g. origin/main)
		// don't cause locked syncs to track the branch tip, not the pin.
		ref = t.opts.Ref
	} else if useFetchHead {
		ref = "FETCH_HEAD"
	}

	if t.opts.Verification != nil {
		verification := Verification{
			dstPath, *t.opts.Verification, t.refFetcher, t.infoLog,
		}
		err := verification.Verify(ref)
		if err != nil {
			return err
		}
	}

	_, _, err = t.cmdRunner.Run([]string{"-c", "advice.detachedHead=false", "checkout", ref}, env, dstPath)
	if err != nil {
		return err
	}

	if !t.opts.SkipInitSubmodules {
		_, _, err = t.cmdRunner.Run([]string{"submodule", "update", "--init", "--recursive"}, env, dstPath)
		if err != nil {
			return err
		}
	}

	return nil
}

const (
	noTagsFlag         = "--no-tags"
	fetchHeadCommit    = "FETCH_HEAD^{commit}"
	originName         = "origin"
	originPrefix       = originName + "/"
	remoteHeadsPrefix  = "refs/heads/"
	remoteTagsPrefix   = "refs/tags/"
	minRemoteRefFields = 2
)

// isLockedSHAWithNamedOriginal reports whether Ref is a locked SHA with a
// known named OriginalRef, i.e. a locked sync that can still target a fetch
// by name instead of resolving the bare SHA via a full fetch.
func (t *Git) isLockedSHAWithNamedOriginal() bool {
	return isHexSHA(t.opts.Ref) &&
		len(t.opts.OriginalRef) > 0 &&
		!isHexSHA(t.opts.OriginalRef)
}

// canTargetFetch reports whether fetch can target a single named ref
// instead of fetching all branches/tags.
func (t *Git) canTargetFetch() bool {
	hasNamedRef := len(t.opts.Ref) > 0 && !isHexSHA(t.opts.Ref)
	namedRef := hasNamedRef && t.opts.RefSelection == nil
	return namedRef || t.isLockedSHAWithNamedOriginal()
}

// targetedFetchArgs builds the "fetch origin ..." args for t.opts.Ref,
// targeting a single named ref where possible. It reports whether the
// result lands only in FETCH_HEAD (true) or creates a local ref (false).
func (t *Git) targetedFetchArgs(
	fetchRef string, refType remoteRefType,
) ([]string, bool) {
	fetchArgs := []string{"fetch", originName}
	if !t.canTargetFetch() {
		// no targeted ref available (e.g. RefSelection, or a bare SHA with
		// no OriginalRef) — fall back to a full fetch of all branches/tags.
		return fetchArgs, false
	}

	ref := strings.TrimPrefix(fetchRef, originPrefix)
	ref = strings.TrimPrefix(ref, remoteHeadsPrefix)
	ref = strings.TrimPrefix(ref, remoteTagsPrefix)
	switch refType {
	case remoteRefBranch:
		return append(fetchArgs, ref+":refs/remotes/origin/"+ref), true
	case remoteRefTag:
		return append(
			fetchArgs, remoteTagsPrefix+ref+":"+remoteTagsPrefix+ref,
			noTagsFlag,
		), true
	default:
		return append(fetchArgs, ref, noTagsFlag), true
	}
}

type remoteRefType int

const (
	remoteRefUnknown remoteRefType = iota
	remoteRefBranch
	remoteRefTag
)

func (t *Git) remoteRefType(dstPath, ref string) (remoteRefType, error) {
	normalizedRef := strings.TrimPrefix(ref, originPrefix)
	switch {
	case strings.HasPrefix(normalizedRef, remoteHeadsPrefix):
		return remoteRefBranch, nil
	case strings.HasPrefix(normalizedRef, remoteTagsPrefix):
		return remoteRefTag, nil
	}

	out, _, err := t.cmdRunner.Run(
		[]string{"ls-remote", "origin", normalizedRef}, nil, dstPath)
	if err != nil {
		return remoteRefUnknown, err
	}

	branchRef := remoteHeadsPrefix + normalizedRef
	tagRef := remoteTagsPrefix + normalizedRef
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < minRemoteRefFields {
			continue
		}
		switch fields[1] {
		case branchRef:
			return remoteRefBranch, nil
		case tagRef:
			return remoteRefTag, nil
		}
	}

	return remoteRefUnknown, nil
}

func (t *Git) resolveRef(dstPath string) (string, error) {
	switch {
	case len(t.opts.Ref) > 0:
		return strings.TrimPrefix(t.opts.Ref, originPrefix), nil

	case t.opts.RefSelection != nil:
		tags, err := t.tags(dstPath)
		if err != nil {
			return "", err
		}
		return ctlver.HighestConstrainedVersion(tags, *t.opts.RefSelection)

	default:
		return "", fmt.Errorf("Expected either ref or ref selection to be specified")
	}
}

func (t *Git) tags(dstPath string) ([]string, error) {
	out, _, err := t.cmdRunner.Run([]string{"tag", "-l"}, nil, dstPath)
	if err != nil {
		return nil, err
	}

	return strings.Split(out, "\n"), nil
}

const (
	minAbbreviatedSHALen = 7
	fullSHALen           = 40
)

// isHexSHA reports whether s looks like a full or abbreviated git commit SHA
// (7–40 hex characters, case-insensitive). SHAs cannot be fetched by refspec
// name and require a full fetch to be resolved.
func isHexSHA(s string) bool {
	if len(s) < minAbbreviatedSHALen || len(s) > fullSHALen {
		return false
	}
	return isHex(s)
}

func isHex(s string) bool {
	for _, c := range s {
		if !isHexDigit(c) {
			return false
		}
	}
	return true
}

func isHexDigit(c rune) bool {
	isDigit := c >= '0' && c <= '9'
	isLower := c >= 'a' && c <= 'f'
	isUpper := c >= 'A' && c <= 'F'
	return isDigit || isLower || isUpper
}

type CommandRunner interface {
	RunMultiple(argss [][]string, env []string, dstPath string) error
	Run(args []string, env []string, dstPath string) (string, string, error)
}

type runner struct {
	infoLog io.Writer
}

func (r *runner) RunMultiple(argss [][]string, env []string, dstPath string) error {
	for _, args := range argss {
		_, _, err := r.Run(args, env, dstPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *runner) Run(args []string, env []string, dstPath string) (string, string, error) {
	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command("git", args...)
	cmd.Env = env
	cmd.Dir = dstPath
	cmd.Stdout = io.MultiWriter(r.infoLog, &stdoutBs)
	cmd.Stderr = io.MultiWriter(r.infoLog, &stderrBs)

	r.infoLog.Write([]byte(fmt.Sprintf("--> git %s\n", strings.Join(args, " "))))

	err := cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("Git %s: %s (stderr: %s)", args, err, stderrBs.String())
	}

	return stdoutBs.String(), stderrBs.String(), nil
}

type gitAuthOpts struct {
	PrivateKey *string
	KnownHosts *string
	Username   *string
	Password   *string
}

func (o gitAuthOpts) IsPresent() bool {
	return o.PrivateKey != nil || o.KnownHosts != nil || o.Username != nil || o.Password != nil
}

func (t *Git) getAuthOpts() (gitAuthOpts, error) {
	var opts gitAuthOpts

	if t.opts.SecretRef != nil {
		secret, err := t.refFetcher.GetSecret(t.opts.SecretRef.Name)
		if err != nil {
			return opts, err
		}

		for name, val := range secret.Data {
			switch name {
			case ctlconf.SecretK8sCoreV1SSHAuthPrivateKey:
				key := string(val)
				opts.PrivateKey = &key
			case ctlconf.SecretSSHAuthKnownHosts:
				hosts := string(val)
				opts.KnownHosts = &hosts
			case ctlconf.SecretK8sCorev1BasicAuthUsernameKey:
				username := string(val)
				opts.Username = &username
			case ctlconf.SecretK8sCorev1BasicAuthPasswordKey:
				password := string(val)
				opts.Password = &password
			default:
				return opts, fmt.Errorf("Unknown secret field '%s' in secret '%s'", name, t.opts.SecretRef.Name)
			}
		}
	}

	return opts, nil
}
