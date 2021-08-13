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
)

type Sync struct {
	opts       ctlconf.DirectoryContentsImgpkgBundle
	refFetcher ctlfetch.RefFetcher
}

func NewSync(opts ctlconf.DirectoryContentsImgpkgBundle, refFetcher ctlfetch.RefFetcher) *Sync {
	return &Sync{opts, refFetcher}
}

var (
	// Example image ref in imgpkg stdout:
	//   Pulling bundle 'index.docker.io/dkalinin/consul-helm@sha256:d1cdbd46561a144332f0744302d45f27583fc0d75002cba473d840f46630c9f7'
	imgpkgPulledImageRef = regexp.MustCompile("(?m)^Pulling bundle '(.+)'$")
)

func (t *Sync) Sync(dstPath string) (ctlconf.LockDirectoryContentsImgpkgBundle, error) {
	lockConf := ctlconf.LockDirectoryContentsImgpkgBundle{}

	if len(t.opts.Image) == 0 {
		return lockConf, fmt.Errorf("Expected non-empty Image")
	}

	imgpkg := ctlimg.NewImgpkg(t.opts.SecretRef, t.refFetcher, nil)

	args := []string{"pull", "-b", t.opts.Image, "-o", dstPath, "--tty=true"}
	args = t.addDangerousArgs(args)
	args = t.addGeneralArgs(args)

	stdoutStr, err := imgpkg.Run(args)
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

func (t *Sync) addDangerousArgs(args []string) []string {
	if t.opts.DangerousSkipTLSVerify {
		args = append(args, "--registry-verify-certs=false")
	}
	return args
}

func (t *Sync) addGeneralArgs(args []string) []string {
	if t.opts.Recursive {
		args = append(args, "--recursive")
	}
	return args
}
