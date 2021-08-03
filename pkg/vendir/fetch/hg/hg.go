// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package hg

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
)

type Hg struct {
	opts       ctlconf.DirectoryContentsHg
	infoLog    io.Writer
	refFetcher ctlfetch.RefFetcher
}

func NewHg(opts ctlconf.DirectoryContentsHg,
	infoLog io.Writer, refFetcher ctlfetch.RefFetcher) *Hg {

	return &Hg{opts, infoLog, refFetcher}
}

// nolint:golint
type HgInfo struct {
	SHA            string
	ChangeSetTitle string
}

func (t *Hg) Retrieve(dstPath string, tempArea ctlfetch.TempArea) (HgInfo, error) {
	if len(t.opts.URL) == 0 {
		return HgInfo{}, fmt.Errorf("Expected non-empty URL")
	}

	err := t.fetch(dstPath, tempArea)
	if err != nil {
		return HgInfo{}, err
	}

	info := HgInfo{}

	out, _, err := t.run([]string{"id", "-i"}, nil, dstPath)
	if err != nil {
		return HgInfo{}, err
	}

	info.SHA = strings.TrimSpace(out)

	out, _, err = t.run([]string{"log", "-l", "1", "-T", "{desc|firstline|strip}", "-r", info.SHA}, nil, dstPath)
	if err != nil {
		return HgInfo{}, err
	}

	info.ChangeSetTitle = strings.TrimSpace(out)

	return info, nil
}

func (t *Hg) fetch(dstPath string, tempArea ctlfetch.TempArea) error {
	authOpts, err := t.getAuthOpts()
	if err != nil {
		return err
	}

	authDir, err := tempArea.NewTempDir("hg-auth")
	if err != nil {
		return err
	}

	defer os.RemoveAll(authDir)

	env := os.Environ()

	hgURL := t.opts.URL

	_, _, err = t.run([]string{"init"}, env, dstPath)
	if err != nil {
		return err
	}

	var authRc string

	if authOpts.Username != nil && authOpts.Password != nil {
		if !strings.HasPrefix(hgURL, "https://") {
			return fmt.Errorf("Username/password authentication is only supported for https remotes")
		}
		hgCredsURL, err := url.Parse(hgURL)
		if err != nil {
			return fmt.Errorf("Parsing hg remote url: %s", err)
		}

		authRc = fmt.Sprintf(`[auth]
hgauth.prefix = https://%s
hgauth.username = %s
hgauth.password = %s
`, hgCredsURL.Host, *authOpts.Username, *authOpts.Password)

	}

	if authOpts.IsPresent() {
		sshCmd := []string{"ssh", "-o", "ServerAliveInterval=30", "-o", "ForwardAgent=no", "-F", "/dev/null"}

		if authOpts.PrivateKey != nil {
			path := filepath.Join(authDir, "private-key")

			err = ioutil.WriteFile(path, []byte(*authOpts.PrivateKey), 0600)
			if err != nil {
				return fmt.Errorf("Writing private key: %s", err)
			}

			sshCmd = append(sshCmd, "-i", path, "-o", "IdentitiesOnly=yes")
		}

		if authOpts.KnownHosts != nil {
			path := filepath.Join(authDir, "known-hosts")

			err = ioutil.WriteFile(path, []byte(*authOpts.KnownHosts), 0600)
			if err != nil {
				return fmt.Errorf("Writing known hosts: %s", err)
			}

			sshCmd = append(sshCmd, "-o", "StrictHostKeyChecking=yes", "-o", "UserKnownHostsFile="+path)
		} else {
			sshCmd = append(sshCmd, "-o", "StrictHostKeyChecking=no")
		}

		authRc = fmt.Sprintf("%s\n[ui]\nssh = %s\n", authRc, strings.Join(sshCmd, " "))
	}

	if (authOpts.Username != nil && authOpts.Password != nil) || authOpts.IsPresent() {
		credsHgRcPath := filepath.Join(authDir, "hgrc")
		err = ioutil.WriteFile(credsHgRcPath, []byte(authRc), 0600)
		if err != nil {
			return fmt.Errorf("Writing %s: %s", credsHgRcPath, err)
		}
		env = append(env, "HGRCPATH="+credsHgRcPath)
	}

	hgrcPath := filepath.Join(dstPath, ".hg", "hgrc")

	hgRc := fmt.Sprintf("[paths]\ndefault = %s\n", hgURL)

	err = ioutil.WriteFile(hgrcPath, []byte(hgRc), 0600)
	if err != nil {
		return fmt.Errorf("Writing %s: %s", hgrcPath, err)
	}

	return t.runMultiple([][]string{
		{"pull"},
		{"checkout", t.opts.Ref},
	}, env, dstPath)
}

func (t *Hg) runMultiple(argss [][]string, env []string, dstPath string) error {
	for _, args := range argss {
		_, _, err := t.run(args, env, dstPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Hg) run(args []string, env []string, dstPath string) (string, string, error) {
	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command("hg", args...)
	cmd.Env = env
	cmd.Dir = dstPath
	cmd.Stdout = io.MultiWriter(t.infoLog, &stdoutBs)
	cmd.Stderr = io.MultiWriter(t.infoLog, &stderrBs)

	t.infoLog.Write([]byte(fmt.Sprintf("--> hg %s\n", strings.Join(args, " "))))

	err := cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("Hg %s: %s (stderr: %s)", args, err, stderrBs.String())
	}

	return stdoutBs.String(), stderrBs.String(), nil
}

type hgAuthOpts struct {
	PrivateKey *string
	KnownHosts *string
	Username   *string
	Password   *string
}

func (o hgAuthOpts) IsPresent() bool {
	return o.PrivateKey != nil || o.KnownHosts != nil || o.Username != nil || o.Password != nil
}

func (t *Hg) getAuthOpts() (hgAuthOpts, error) {
	var opts hgAuthOpts

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
