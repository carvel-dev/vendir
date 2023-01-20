// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package helmchart

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
	"sigs.k8s.io/yaml"
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

	if t.opts.Repository != nil && strings.HasPrefix(t.opts.Repository.URL, "oci://") {
		err = NewOCISource(t.opts, t.helmBinary, t.refFetcher).Fetch(chartsDir, tempArea)
		if err != nil {
			return lockConf, err
		}
	} else {
		err = NewHTTPSource(t.opts, t.helmBinary, t.refFetcher).Fetch(chartsDir, tempArea)
		if err != nil {
			return lockConf, err
		}
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
		return "", fmt.Errorf("Expected single directory in charts directory, but was: %s", dirNames)
	}
	return filepath.Join(chartsPath, dirNames[0]), nil
}

func helmEnv(helmHomeDir string) []string {
	// Previous discussion around env vars propagation:
	//   https://github.com/carvel-dev/vendir/issues/164
	// Example: without propagating few env vars (e.g. $HOME),
	//          asdf pkg mgr cannot execute helm binary
	// Helm env vars: https://helm.sh/docs/helm/helm/ and https://v2.helm.sh/docs/helm/
	return append(os.Environ(), []string{
		"HELM_HOME=" + helmHomeDir, // for helm2
		"TEMP=" + helmHomeDir,
		"HELM_CACHE_HOME=" + helmHomeDir,
		"HELM_CONFIG_HOME=" + helmHomeDir,
		"HELM_DATA_HOME=" + helmHomeDir,
	}...)
}
