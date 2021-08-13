// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"fmt"
	"regexp"
	"strings"

	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
	ctlver "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/versions/v1alpha1"
)

type Sync struct {
	opts   ctlconf.DirectoryContentsImage
	imgpkg *Imgpkg
}

func NewSync(opts ctlconf.DirectoryContentsImage, refFetcher ctlfetch.RefFetcher) *Sync {
	imgpkgOpts := ImgpkgOpts{
		SecretRef:              opts.SecretRef,
		DangerousSkipTLSVerify: opts.DangerousSkipTLSVerify,
	}
	return &Sync{opts, NewImgpkg(imgpkgOpts, refFetcher)}
}

var (
	// Example image ref in imgpkg stdout:
	//   Pulling image 'index.docker.io/dkalinin/consul-helm@sha256:d1cdbd46561a144332f0744302d45f27583fc0d75002cba473d840f46630c9f7'
	imgpkgPulledImageRef = regexp.MustCompile("(?m)^Pulling image '(.+)'$")
)

func (t *Sync) Sync(dstPath string) (ctlconf.LockDirectoryContentsImage, error) {
	lockConf := ctlconf.LockDirectoryContentsImage{}

	url, err := t.resolveURL()
	if err != nil {
		return lockConf, err
	}

	stdoutStr, err := t.imgpkg.Run([]string{"pull", "-i", url, "-o", dstPath, "--tty=true"})
	if err != nil {
		return lockConf, err
	}

	matches := imgpkgPulledImageRef.FindStringSubmatch(stdoutStr)
	if len(matches) != 2 {
		return lockConf, fmt.Errorf("Expected to find pulled image ref in stdout, but did not (stdout: '%s')", stdoutStr)
	}
	if !strings.Contains(matches[1], "@") {
		return lockConf, fmt.Errorf("Expected ref '%s' to be in digest form, but was not", matches[1])
	}

	lockConf.URL = matches[1]

	return lockConf, nil
}

func (t *Sync) resolveURL() (string, error) {
	if len(t.opts.URL) == 0 {
		return "", fmt.Errorf("Expected non-empty URL")
	}

	if t.opts.TagSelection != nil {
		tags, err := t.imgpkg.Tags(t.opts.URL)
		if err != nil {
			return "", err
		}

		selectedTag, err := ctlver.HighestConstrainedVersion(tags, *t.opts.TagSelection)
		if err != nil {
			return "", fmt.Errorf("Determining tag selection: %s", err)
		}

		// In case URL erroneously contains tag or digest,
		// pull operation will fail, so no need to do any checks here.
		return t.opts.URL + ":" + selectedTag, nil
	}

	return t.opts.URL, nil
}
