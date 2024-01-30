// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"fmt"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	ctlfetch "carvel.dev/vendir/pkg/vendir/fetch"
	ctlcache "carvel.dev/vendir/pkg/vendir/fetch/cache"
	ctlver "carvel.dev/vendir/pkg/vendir/versions"
)

type Sync struct {
	opts   ctlconf.DirectoryContentsImage
	imgpkg *Imgpkg
}

func NewSync(opts ctlconf.DirectoryContentsImage, refFetcher ctlfetch.RefFetcher, c ctlcache.Cache) *Sync {
	imgpkgOpts := ImgpkgOpts{
		SecretRef:              opts.SecretRef,
		DangerousSkipTLSVerify: opts.DangerousSkipTLSVerify,
		ResponseHeaderTimeout:  opts.ResponseHeaderTimeout,
	}
	return &Sync{opts, NewImgpkg(imgpkgOpts, refFetcher, c)}
}

func (t Sync) Desc() string {
	url := "?"
	if len(t.opts.URL) > 0 {
		url = t.opts.URL
		if t.opts.TagSelection != nil {
			url += ":tag=" + t.opts.TagSelection.Description()
		}
	}
	return url
}

func (t *Sync) Sync(dstPath string) (ctlconf.LockDirectoryContentsImage, error) {
	lockConf := ctlconf.LockDirectoryContentsImage{}

	url, err := t.resolveURL()
	if err != nil {
		return lockConf, err
	}

	imgRef, err := t.imgpkg.FetchImage(url, dstPath)
	if err != nil {
		return lockConf, err
	}

	lockConf.URL = imgRef
	if len(t.opts.PreresolvedTag()) > 0 {
		lockConf.Tag = t.opts.PreresolvedTag()
	} else {
		lockConf.Tag = NewGuessedRefParts(url).Tag
	}

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
