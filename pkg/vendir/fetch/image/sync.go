// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"fmt"
	"regexp"
	"strings"

	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
)

type Sync struct {
	opts       ctlconf.DirectoryContentsImage
	refFetcher ctlfetch.RefFetcher
}

func NewSync(opts ctlconf.DirectoryContentsImage, refFetcher ctlfetch.RefFetcher) *Sync {
	return &Sync{opts, refFetcher}
}

var (
	// Example image ref in imgpkg stdout:
	//   Pulling image 'index.docker.io/dkalinin/consul-helm@sha256:d1cdbd46561a144332f0744302d45f27583fc0d75002cba473d840f46630c9f7'
	imgpkgPulledImageRef = regexp.MustCompile("(?m)^Pulling image '(.+)'$")
)

func (t *Sync) Sync(dstPath string) (ctlconf.LockDirectoryContentsImage, error) {
	lockConf := ctlconf.LockDirectoryContentsImage{}

	if len(t.opts.URL) == 0 {
		return lockConf, fmt.Errorf("Expected non-empty URL")
	}

	imgpkg := NewImgpkg(t.opts.SecretRef, t.refFetcher, nil)

	args := []string{"pull", "-i", t.opts.URL, "-o", dstPath, "--tty=true"}
	args = t.addDangerousArgs(args)

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

	lockConf.URL = matches[1]

	return lockConf, nil
}

func (t *Sync) addDangerousArgs(args []string) []string {
	if t.opts.DangerousSkipTLSVerify {
		args = append(args, "--registry-verify-certs=false")
	}
	return args
}
