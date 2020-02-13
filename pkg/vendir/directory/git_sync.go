package directory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cppforlife/go-cli-ui/ui"
)

type GitSync struct {
	opts ConfigContentsGit
	ui   ui.UI
}

func (d GitSync) Sync(dstPath string) (LockConfigContentsGit, error) {
	gitLockConf := LockConfigContentsGit{}
	incomingTmpPath := filepath.Join(incomingTmpDir, "git")

	err := os.MkdirAll(incomingTmpPath, 0700)
	if err != nil {
		return gitLockConf, fmt.Errorf("Creating incoming dir '%s' for git fetching: %s", incomingTmpPath, err)
	}

	defer os.RemoveAll(incomingTmpPath)

	git := NewGit(d.opts, NewInfoLog(d.ui))

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

func (GitSync) singleLineCommitTitle(in string) string {
	pieces := strings.SplitN(in, "\n", 2)
	if len(pieces) > 1 {
		return pieces[0] + "..."
	}
	return pieces[0]
}
