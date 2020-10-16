package git

import (
	"fmt"
	"io"
	"os"
	"strings"

	ctlconf "github.com/k14s/vendir/pkg/vendir/config"
	ctlfetch "github.com/k14s/vendir/pkg/vendir/fetch"
)

type Sync struct {
	opts       ctlconf.DirectoryContentsGit
	log        io.Writer
	refFetcher ctlfetch.RefFetcher
}

func NewSync(opts ctlconf.DirectoryContentsGit,
	log io.Writer, refFetcher ctlfetch.RefFetcher) Sync {

	return Sync{opts, log, refFetcher}
}

func (d Sync) Desc() string {
	ref := "?"
	switch {
	case len(d.opts.Ref) > 0:
		ref = d.opts.Ref
	case d.opts.RefSelection != nil:
		switch {
		case d.opts.RefSelection.Semver != nil:
			ref = fmt.Sprintf("[%s]", d.opts.RefSelection.Semver.Constraints)
		}
	}
	return fmt.Sprintf("%s@%s", d.opts.URL, ref)
}

func (d Sync) Sync(dstPath string, tempArea ctlfetch.TempArea) (ctlconf.LockDirectoryContentsGit, error) {
	gitLockConf := ctlconf.LockDirectoryContentsGit{}

	incomingTmpPath, err := tempArea.NewTempDir("git")
	if err != nil {
		return gitLockConf, err
	}

	defer os.RemoveAll(incomingTmpPath)

	git := NewGit(d.opts, d.log, d.refFetcher)

	info, err := git.Retrieve(incomingTmpPath, tempArea)
	if err != nil {
		return gitLockConf, fmt.Errorf("Fetching git repository: %s", err)
	}

	gitLockConf.SHA = info.SHA
	gitLockConf.Tags = info.Tags
	gitLockConf.CommitTitle = d.singleLineCommitTitle(info.CommitTitle)

	err = os.RemoveAll(dstPath)
	if err != nil {
		return gitLockConf, fmt.Errorf("Deleting dir %s: %s", dstPath, err)
	}

	err = os.Rename(incomingTmpPath, dstPath)
	if err != nil {
		return gitLockConf, fmt.Errorf("Moving directory '%s' to staging dir: %s", incomingTmpPath, err)
	}

	return gitLockConf, nil
}

func (Sync) singleLineCommitTitle(in string) string {
	pieces := strings.SplitN(in, "\n", 2)
	if len(pieces) > 1 {
		return pieces[0] + "..."
	}
	return pieces[0]
}
