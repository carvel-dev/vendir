// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package hg

import (
	"fmt"
	"io"
	"os"
	"strings"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	ctlfetch "carvel.dev/vendir/pkg/vendir/fetch"
)

type Sync struct {
	opts       ctlconf.DirectoryContentsHg
	log        io.Writer
	refFetcher ctlfetch.RefFetcher
}

func NewSync(opts ctlconf.DirectoryContentsHg,
	log io.Writer, refFetcher ctlfetch.RefFetcher) Sync {

	return Sync{opts, log, refFetcher}
}

func (d Sync) Desc() string {
	ref := "?"
	switch {
	case len(d.opts.Ref) > 0:
		ref = d.opts.Ref
	}
	return fmt.Sprintf("%s@%s", d.opts.URL, ref)
}

func (d Sync) Sync(dstPath string, tempArea ctlfetch.TempArea) (ctlconf.LockDirectoryContentsHg, error) {
	hgLockConf := ctlconf.LockDirectoryContentsHg{}

	incomingTmpPath, err := tempArea.NewTempDir("hg")
	if err != nil {
		return hgLockConf, err
	}

	defer os.RemoveAll(incomingTmpPath)

	hg := NewHg(d.opts, d.log, d.refFetcher)

	info, err := hg.Retrieve(incomingTmpPath, tempArea)
	if err != nil {
		return hgLockConf, fmt.Errorf("Fetching hg repository: %s", err)
	}

	hgLockConf.SHA = info.SHA
	hgLockConf.ChangeSetTitle = d.singleLineChangeSetTitle(info.ChangeSetTitle)

	err = os.RemoveAll(dstPath)
	if err != nil {
		return hgLockConf, fmt.Errorf("Deleting dir %s: %s", dstPath, err)
	}

	err = os.Rename(incomingTmpPath, dstPath)
	if err != nil {
		return hgLockConf, fmt.Errorf("Moving directory '%s' to staging dir: %s", incomingTmpPath, err)
	}

	return hgLockConf, nil
}

func (Sync) singleLineChangeSetTitle(in string) string {
	pieces := strings.SplitN(in, "\n", 2)
	if len(pieces) > 1 {
		return pieces[0] + "..."
	}
	return pieces[0]
}
