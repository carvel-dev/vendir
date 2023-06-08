// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
	ctlver "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/versions"
)

type Git struct {
	opts       ctlconf.DirectoryContentsGit
	infoLog    io.Writer
	refFetcher ctlfetch.RefFetcher
}

func NewGit(opts ctlconf.DirectoryContentsGit,
	infoLog io.Writer, refFetcher ctlfetch.RefFetcher) *Git {

	return &Git{opts, infoLog, refFetcher}
}

//nolint:revive
type GitInfo struct {
	SHA         string
	Tags        []string
	CommitTitle string
}

func (t *Git) Retrieve(dstPath string, tempArea ctlfetch.TempArea) (GitInfo, error) {
	if len(t.opts.URL) == 0 {
		return GitInfo{}, fmt.Errorf("Expected non-empty URL")
	}

	err := t.fetch(dstPath, tempArea)
	if err != nil {
		return GitInfo{}, err
	}

	info := GitInfo{}

	out, _, err := t.run([]string{"rev-parse", "HEAD"}, nil, dstPath)
	if err != nil {
		return GitInfo{}, err
	}

	info.SHA = strings.TrimSpace(out)

	out, _, err = t.run([]string{"describe", "--tags", info.SHA}, nil, dstPath)
	if err == nil {
		info.Tags = strings.Split(strings.TrimSpace(out), "\n")
	}

	out, _, err = t.run([]string{"log", "-n", "1", "--pretty=%B", info.SHA}, nil, dstPath)
	if err != nil {
		return GitInfo{}, err
	}

	info.CommitTitle = strings.TrimSpace(out)

	return info, nil
}

func (t *Git) fetch(dstPath string, tempArea ctlfetch.TempArea) error {
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

			err = os.WriteFile(path, []byte(*authOpts.PrivateKey), 0600)
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

	gitURL := t.opts.URL
	gitCredsPath := filepath.Join(authDir, ".git-credentials")

	if authOpts.Username != nil && authOpts.Password != nil {
		if !strings.HasPrefix(gitURL, "https://") {
			return fmt.Errorf("Username/password authentication is only supported for https remotes")
		}

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

	argss := [][]string{
		{"init"},
		{"config", "credential.helper", "store --file " + gitCredsPath},
		{"remote", "add", "origin", gitURL},
	}

	if t.opts.RefSelection != nil || t.opts.Ref != "" {
		// fetch tags for selection
		argss = append(argss, []string{"config", "remote.origin.tagOpt", "--tags"})
	}

	{
		fetchArgs := []string{"fetch", "origin"}
		if strings.HasPrefix(t.opts.Ref, "origin/") {
			// only fetch the exact ref we're seeking
			fetchArgs = append(fetchArgs, t.opts.Ref[7:])
		}
		if t.opts.Depth > 0 {
			fetchArgs = append(fetchArgs, "--depth", strconv.Itoa(t.opts.Depth))
		}
		argss = append(argss, fetchArgs)
	}

	err = t.runMultiple(argss, env, dstPath)
	if err != nil {
		return err
	}

	ref, err := t.resolveRef(dstPath)
	if err != nil {
		return err
	}

	if t.opts.Verification != nil {
		err := Verification{dstPath, *t.opts.Verification, t.refFetcher}.Verify(ref)
		if err != nil {
			return err
		}
	}

	_, _, err = t.run([]string{"-c", "advice.detachedHead=false", "checkout", ref}, env, dstPath)
	if err != nil {
		return err
	}

	if !t.opts.SkipInitSubmodules {
		_, _, err = t.run([]string{"submodule", "update", "--init", "--recursive"}, env, dstPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Git) resolveRef(dstPath string) (string, error) {
	switch {
	case len(t.opts.Ref) > 0:
		return t.opts.Ref, nil

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
	out, _, err := t.run([]string{"tag", "-l"}, nil, dstPath)
	if err != nil {
		return nil, err
	}

	return strings.Split(out, "\n"), nil
}

func (t *Git) runMultiple(argss [][]string, env []string, dstPath string) error {
	for _, args := range argss {
		_, _, err := t.run(args, env, dstPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Git) run(args []string, env []string, dstPath string) (string, string, error) {
	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command("git", args...)
	cmd.Env = env
	cmd.Dir = dstPath
	cmd.Stdout = io.MultiWriter(t.infoLog, &stdoutBs)
	cmd.Stderr = io.MultiWriter(t.infoLog, &stderrBs)

	t.infoLog.Write([]byte(fmt.Sprintf("--> git %s\n", strings.Join(args, " "))))

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
