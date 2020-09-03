// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package directory

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ghodss/yaml"
)

type HelmChart struct {
	opts       ConfigContentsHelmChart
	refFetcher RefFetcher
}

func NewHelmChart(opts ConfigContentsHelmChart, refFetcher RefFetcher) *HelmChart {
	return &HelmChart{opts, refFetcher}
}

func (t *HelmChart) Desc() string {
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

func (t *HelmChart) Sync(dstPath string) (LockConfigContentsHelmChart, error) {
	lockConf := LockConfigContentsHelmChart{}

	if len(t.opts.Name) == 0 {
		return lockConf, fmt.Errorf("Expected non-empty name")
	}

	chartsDir, err := TempDir("helm-chart")
	if err != nil {
		return lockConf, err
	}

	defer os.RemoveAll(chartsDir)

	err = t.init(chartsDir)
	if err != nil {
		return lockConf, err
	}

	err = t.fetch(chartsDir)
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

	err = MoveDir(chartPath, dstPath)
	if err != nil {
		return lockConf, err
	}

	lockConf.Version = meta.Version
	lockConf.AppVersion = meta.AppVersion

	return lockConf, nil
}

func (t *HelmChart) init(chartsPath string) error {
	args := []string{"init", "--client-only"}

	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command("helm", args...)
	cmd.Env = []string{"HOME=" + chartsPath}
	cmd.Stdout = &stdoutBs
	cmd.Stderr = &stderrBs

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Init helm: %s (stderr: %s)", err, stderrBs.String())
	}

	return nil
}

func (t *HelmChart) fetch(chartsPath string) error {
	args := []string{"fetch", t.opts.Name, "--untar", "--untardir", chartsPath}

	if len(t.opts.Version) > 0 {
		args = append(args, []string{"--version", t.opts.Version}...)
	}

	if t.opts.Repository != nil {
		if len(t.opts.Repository.URL) == 0 {
			return fmt.Errorf("Expected non-empty repository URL")
		}

		// Add repo explicitly for helm to recognize it
		{
			var stdoutBs, stderrBs bytes.Buffer

			cmd := exec.Command("helm", "repo", "add", "bitnami", t.opts.Repository.URL)
			cmd.Env = []string{"HOME=" + chartsPath}
			cmd.Stdout = &stdoutBs
			cmd.Stderr = &stderrBs

			err := cmd.Run()
			if err != nil {
				return fmt.Errorf("Add helm chart repository: %s (stderr: %s)", err, stderrBs.String())
			}
		}

		args = append(args, []string{"--repo", t.opts.Repository.URL}...)

		var err error

		args, err = t.addAuthArgs(args)
		if err != nil {
			return fmt.Errorf("Adding helm chart auth info: %s", err)
		}
	}

	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command("helm", args...)
	cmd.Env = []string{"HOME=" + chartsPath}
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

func (t *HelmChart) retrieveChartMeta(chartPath string) (chartMeta, error) {
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

func (t *HelmChart) findChartDir(chartsPath string) (string, error) {
	files, err := ioutil.ReadDir(chartsPath)
	if err != nil {
		return "", err
	}

	var result []os.FileInfo
	for _, file := range files {
		if file.IsDir() && file.Name() != ".helm" {
			result = append(result, file)
		}
	}

	if len(result) != 1 {
		return "", fmt.Errorf("Expected single directory in charts directory")
	}
	return filepath.Join(chartsPath, result[0].Name()), nil
}

func (t *HelmChart) addAuthArgs(args []string) ([]string, error) {
	var authArgs []string

	if t.opts.Repository.SecretRef != nil {
		secret, err := t.refFetcher.GetSecret(t.opts.Repository.SecretRef.Name)
		if err != nil {
			return nil, err
		}

		for name, val := range secret.Data {
			switch name {
			case k8s_corev1_BasicAuthUsernameKey:
				authArgs = append(authArgs, []string{"--username", string(val)}...)
			case k8s_corev1_BasicAuthPasswordKey:
				authArgs = append(authArgs, []string{"--password", string(val)}...)
			default:
				return nil, fmt.Errorf("Unknown secret field '%s' in secret '%s'", name, secret.Name)
			}
		}
	}

	return append(args, authArgs...), nil
}
