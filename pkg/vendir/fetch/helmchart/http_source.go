// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package helmchart

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
)

type HTTPSource struct {
	opts       ctlconf.DirectoryContentsHelmChart
	helmBinary string
	refFetcher ctlfetch.RefFetcher
}

func NewHTTPSource(opts ctlconf.DirectoryContentsHelmChart,
	helmBinary string, refFetcher ctlfetch.RefFetcher) *HTTPSource {

	return &HTTPSource{opts, helmBinary, refFetcher}
}

func (t *HTTPSource) Fetch(dstPath string, tempArea ctlfetch.TempArea) error {
	helmHomeDir, err := tempArea.NewTempDir("helm-home")
	if err != nil {
		return err
	}

	defer os.RemoveAll(helmHomeDir)

	err = t.init(helmHomeDir)
	if err != nil {
		return err
	}

	return t.fetch(helmHomeDir, dstPath)
}

func (t *HTTPSource) init(helmHomeDir string) error {
	args := []string{"init", "--client-only", "--stable-repo-url", "https://charts.helm.sh/stable"}

	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command(t.helmBinary, args...)
	cmd.Env = helmEnv(helmHomeDir)
	cmd.Stdout = &stdoutBs
	cmd.Stderr = &stderrBs

	err := cmd.Run()
	if err != nil {
		stderrStr := stderrBs.String()
		// Helm 3 does not have/need init command
		if strings.Contains(stderrStr, "unknown command") {
			return nil
		}

		return fmt.Errorf("Init helm: %s (stderr: %s)", err, stderrStr)
	}

	return nil
}

func (t *HTTPSource) fetch(helmHomeDir, chartsPath string) error {
	const (
		stablePrefix  = "stable/"
		stableRepoURL = "https://kubernetes-charts.storage.googleapis.com"
	)

	var name, repoURL string

	if strings.HasPrefix(t.opts.Name, stablePrefix) {
		name = strings.TrimPrefix(t.opts.Name, stablePrefix)
		repoURL = stableRepoURL
	} else {
		name = t.opts.Name
	}

	fetchArgs := []string{"fetch", name, "--untar", "--untardir", chartsPath}

	if len(t.opts.Version) > 0 {
		fetchArgs = append(fetchArgs, []string{"--version", t.opts.Version}...)
	}

	if t.opts.Repository != nil {
		if len(t.opts.Repository.URL) == 0 {
			return fmt.Errorf("Expected non-empty repository URL")
		}
		repoURL = t.opts.Repository.URL
	}

	if len(repoURL) > 0 {
		// Add repo explicitly for helm to be recognized in fetch command
		{
			repoAddArgs := []string{"repo", "add", "vendir-unused", repoURL}
			repoAddArgs, err := t.addAuthArgs(repoAddArgs)
			if err != nil {
				return fmt.Errorf("Adding helm chart auth info: %s", err)
			}

			var stdoutBs, stderrBs bytes.Buffer

			cmd := exec.Command(t.helmBinary, repoAddArgs...)
			cmd.Env = helmEnv(helmHomeDir)
			cmd.Stdout = &stdoutBs
			cmd.Stderr = &stderrBs

			err = cmd.Run()
			if err != nil {
				return fmt.Errorf("Add helm chart repository: %s (stderr: %s)", err, stderrBs.String())
			}
		}

		fetchArgs = append(fetchArgs, []string{"--repo", repoURL}...)

		var err error

		fetchArgs, err = t.addAuthArgs(fetchArgs)
		if err != nil {
			return fmt.Errorf("Adding helm chart auth info: %s", err)
		}
	}

	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command(t.helmBinary, fetchArgs...)
	cmd.Env = helmEnv(helmHomeDir)
	cmd.Stdout = &stdoutBs
	cmd.Stderr = &stderrBs

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Fetching helm chart: %s (stderr: %s)", err, stderrBs.String())
	}

	return nil
}

func (t *HTTPSource) addAuthArgs(args []string) ([]string, error) {
	var authArgs []string

	if t.opts.Repository != nil && t.opts.Repository.SecretRef != nil {
		secret, err := t.refFetcher.GetSecret(t.opts.Repository.SecretRef.Name)
		if err != nil {
			return nil, err
		}

		for name, val := range secret.Data {
			switch name {
			case ctlconf.SecretK8sCorev1BasicAuthUsernameKey:
				authArgs = append(authArgs, []string{"--username", string(val)}...)
			case ctlconf.SecretK8sCorev1BasicAuthPasswordKey:
				authArgs = append(authArgs, []string{"--password", string(val)}...)
			default:
				return nil, fmt.Errorf("Unknown secret field '%s' in secret '%s'", name, secret.Metadata.Name)
			}
		}
	}

	return append(args, authArgs...), nil
}
