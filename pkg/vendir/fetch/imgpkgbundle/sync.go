// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
package imgpkgbundle

import (
	"fmt"
	"regexp"
	"strings"

	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
	ctlimg "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/image"
	ctlver "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/versions/v1alpha1"
)

type Sync struct {
	opts   ctlconf.DirectoryContentsImgpkgBundle
	imgpkg *ctlimg.Imgpkg
}

func NewSync(opts ctlconf.DirectoryContentsImgpkgBundle, refFetcher ctlfetch.RefFetcher) *Sync {
	imgpkgOpts := ctlimg.ImgpkgOpts{
		SecretRef:              opts.SecretRef,
		DangerousSkipTLSVerify: opts.DangerousSkipTLSVerify,
	}
	return &Sync{opts, ctlimg.NewImgpkg(imgpkgOpts, refFetcher)}
}

var (
	// Example image ref in imgpkg stdout:
	//   Pulling bundle 'index.docker.io/dkalinin/consul-helm@sha256:d1cdbd46561a144332f0744302d45f27583fc0d75002cba473d840f46630c9f7'
	imgpkgPulledImageRef = regexp.MustCompile("(?m)^Pulling bundle '(.+)'$")
)

func (t *Sync) Sync(dstPath string) (ctlconf.LockDirectoryContentsImgpkgBundle, error) {
	lockConf := ctlconf.LockDirectoryContentsImgpkgBundle{}

	image, err := t.resolveImage()
	if err != nil {
		return lockConf, err
	}

	args := []string{"pull", "-b", image, "-o", dstPath, "--tty=true"}
	if t.opts.Recursive {
		args = append(args, "--recursive")
	}

	stdoutStr, err := t.imgpkg.Run(args)
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

	lockConf.Image = matches[1]

	return lockConf, nil
}

func (t *Sync) resolveImage() (string, error) {
	if len(t.opts.Image) == 0 {
		return "", fmt.Errorf("Expected non-empty image")
	}

	if t.opts.TagSelection != nil {
		tags, err := t.imgpkg.Tags(t.opts.Image)
		if err != nil {
			return "", err
		}

		selectedTag, err := ctlver.HighestConstrainedVersion(tags, *t.opts.TagSelection)
		if err != nil {
			return "", fmt.Errorf("Determining tag selection: %s", err)
		}

		// In case image erroneously contains tag or digest,
		// pull operation will fail, so no need to do any checks here.
		return t.opts.Image + ":" + selectedTag, nil
	}

	return t.opts.Image, nil
}
