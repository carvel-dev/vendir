// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package helmchart

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
)

type Sync struct {
	opts       ctlconf.DirectoryContentsHelmChart
	helmBinary string
	refFetcher ctlfetch.RefFetcher
}

func NewSync(opts ctlconf.DirectoryContentsHelmChart,
	helmBinary string, refFetcher ctlfetch.RefFetcher) *Sync {

	if helmBinary == "" {
		helmBinary = "helm"
		if opts.HelmVersion == "3" {
			helmBinary = "helm3"
		}
	}
	return &Sync{opts, helmBinary, refFetcher}
}

func (t *Sync) Desc() string {
	desc := ""
	if t.opts.Repository != nil && len(t.opts.Repository.URL) > 0 {
		desc += t.opts.Repository.URL + "@"
	}
	desc += t.opts.Name + ":"
	if len(t.opts.Version) > 0 {
		desc += t.opts.Version
	} else {
		desc += "latest"
	}
	return desc
}

func (t *Sync) Sync(dstPath string, tempArea ctlfetch.TempArea) (ctlconf.LockDirectoryContentsHelmChart, error) {
	lockConf := ctlconf.LockDirectoryContentsHelmChart{}

	if len(t.opts.Name) == 0 {
		return lockConf, fmt.Errorf("Expected non-empty name")
	}

	chartsDir, err := tempArea.NewTempDir("helm-chart")
	if err != nil {
		return lockConf, err
	}

	defer os.RemoveAll(chartsDir)

	helmHomeDir, err := tempArea.NewTempDir("helm-home")
	if err != nil {
		return lockConf, err
	}

	defer os.RemoveAll(helmHomeDir)

	err = t.init(helmHomeDir)
	if err != nil {
		return lockConf, err
	}

	err = t.fetch(helmHomeDir, chartsDir)
	if err != nil {
		return lockConf, err
	}

	chartPath, err := t.findChartDir(chartsDir)
	if err != nil {
		return lockConf, fmt.Errorf("Finding single helm chart: %s", err)
	}

	meta, err := t.retrieveChartMeta(chartPath)
	if err != nil {
		return lockConf, fmt.Errorf("Retrieving helm chart metadata: %s", err)
	}

	err = ctlfetch.MoveDir(chartPath, dstPath)
	if err != nil {
		return lockConf, err
	}

	lockConf.Version = meta.Version
	lockConf.AppVersion = meta.AppVersion

	return lockConf, nil
}

func (t *Sync) init(helmHomeDir string) error {
	args := []string{"init", "--client-only"}

	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command(t.helmBinary, args...)
	cmd.Env = []string{"HOME=" + helmHomeDir}
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

func (t *Sync) fetch(helmHomeDir, chartsPath string) error {
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
			cmd.Env = []string{"HOME=" + helmHomeDir}
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
	cmd.Env = []string{"HOME=" + helmHomeDir}
	cmd.Stdout = &stdoutBs
	cmd.Stderr = &stderrBs

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Fetching helm chart: %s (stderr: %s)", err, stderrBs.String())
	}

	return nil
}

type chartMeta struct {
	AppVersion string
	Version    string
}

func (t *Sync) retrieveChartMeta(chartPath string) (chartMeta, error) {
	var meta chartMeta

	bs, err := ioutil.ReadFile(filepath.Join(chartPath, "Chart.yaml"))
	if err != nil {
		return meta, fmt.Errorf("Reading Chart.yaml: %s", err)
	}

	err = yaml.Unmarshal(bs, &meta)
	if err != nil {
		return meta, err
	}

	if len(meta.Version) == 0 {
		return meta, fmt.Errorf("Expected non-empty chart version")
	}

	return meta, nil
}

func (t *Sync) findChartDir(chartsPath string) (string, error) {
	files, err := ioutil.ReadDir(chartsPath)
	if err != nil {
		return "", err
	}

	var dirNames []string
	for _, file := range files {
		if file.IsDir() && !strings.HasSuffix(file.Name(), ".tgz") {
			dirNames = append(dirNames, file.Name())
		}
	}

	if len(dirNames) != 1 {
		return "", fmt.Errorf("Expected single directory in charts directory, but was: %#v", dirNames)
	}
	return filepath.Join(chartsPath, dirNames[0]), nil
}

func (t *Sync) addAuthArgs(args []string) ([]string, error) {
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
