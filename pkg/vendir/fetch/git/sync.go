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
	opts ctlconf.DirectoryContentsGit
	log  io.Writer
}

func NewSync(opts ctlconf.DirectoryContentsGit, log io.Writer) Sync {
	return Sync{opts, log}
}

func (d Sync) Sync(dstPath string, tempArea ctlfetch.TempArea) (ctlconf.LockDirectoryContentsGit, error) {
	gitLockConf := ctlconf.LockDirectoryContentsGit{}

	incomingTmpPath, err := tempArea.NewTempDir("git")
	if err != nil {
		return gitLockConf, err
	}

	defer os.RemoveAll(incomingTmpPath)

	git := NewGit(d.opts, d.log)

	info, err := git.Retrieve(incomingTmpPath)
	if err != nil {
		return gitLockConf, fmt.Errorf("Fetching git repository: %s", err)
	}

	gitLockConf.SHA = info.SHA
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
