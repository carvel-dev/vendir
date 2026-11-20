// Copyright 2025 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package directory

import (
	"path"

	ctlgit "carvel.dev/vendir/pkg/vendir/fetch/git"
	ctlhg "carvel.dev/vendir/pkg/vendir/fetch/hg"
	ctlstatus "carvel.dev/vendir/pkg/vendir/status"
)

func (d *Directory) Status(syncOpts SyncOpts) (ctlstatus.StatusList, error) {
	var res ctlstatus.StatusList

	for _, contents := range d.opts.Contents {
		contentPath := path.Join(d.opts.Path, contents.Path)
		switch {
		case contents.Git != nil:
			gitSync := ctlgit.NewSync(
				*contents.Git, NewInfoLog(d.ui),
				syncOpts.RefFetcher, syncOpts.Cache)

			gitStatus, err := gitSync.Status(contentPath)
			if err != nil {
				return nil, err
			}

			if gitStatus != nil {
				gitStatus.DirectoryPath = d.opts.Path
				gitStatus.ContentPath = contents.Path
				res = append(res, gitStatus)
			}
		case contents.Hg != nil:
			hgSync := ctlhg.NewSync(
				*contents.Hg, NewInfoLog(d.ui),
				syncOpts.RefFetcher, syncOpts.Cache)

			hgStatus, err := hgSync.Status(contentPath)
			if err != nil {
				return nil, err
			}

			if hgStatus != nil {
				hgStatus.DirectoryPath = d.opts.Path
				hgStatus.ContentPath = contents.Path
				res = append(res, hgStatus)
			}
		}
	}

	return res, nil
}
