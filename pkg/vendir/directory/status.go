package directory

import (
	"path"

	ctlgit "carvel.dev/vendir/pkg/vendir/fetch/git"
	ctlhg "carvel.dev/vendir/pkg/vendir/fetch/hg"
	ctlstatus "carvel.dev/vendir/pkg/vendir/status"
)

func (d *Directory) Status(syncOpts SyncOpts) (map[string]*ctlstatus.Status, error) {
	res := map[string]*ctlstatus.Status{}

	for _, contents := range d.opts.Contents {
		path := path.Join(d.opts.Path, contents.Path)
		switch {
		case contents.Git != nil:
			gitSync := ctlgit.NewSync(*contents.Git, NewInfoLog(d.ui), syncOpts.RefFetcher, syncOpts.Cache)

			gitStatus, err := gitSync.Status(path)
			if err != nil {
				return nil, err
			}

			if gitStatus != nil {
				res[path] = gitStatus
			}
		case contents.Hg != nil:
			hgSync := ctlhg.NewSync(
				*contents.Hg, NewInfoLog(d.ui), syncOpts.RefFetcher, syncOpts.Cache)

			hgStatus, err := hgSync.Status(path)
			if err != nil {
				return nil, err
			}

			if hgStatus != nil {
				res[path] = hgStatus
			}
		}
	}

	return res, nil
}
