package directory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cppforlife/go-cli-ui/ui"
	dircopy "github.com/otiai10/copy"
)

type Directory struct {
	opts Config
	ui   ui.UI
}

func NewDirectory(opts Config, ui ui.UI) *Directory {
	return &Directory{opts, ui}
}

var (
	tmpDir         = ".vendir-tmp"
	stagingTmpDir  = filepath.Join(tmpDir, "staging")
	incomingTmpDir = filepath.Join(tmpDir, "incoming")
)

func (d *Directory) Sync() (LockConfig, error) {
	lockConfig := LockConfig{Path: d.opts.Path}

	err := d.cleanUpTmpDir()
	if err != nil {
		return lockConfig, err
	}

	defer d.cleanUpTmpDir()

	err = os.MkdirAll(stagingTmpDir, 0700)
	if err != nil {
		return lockConfig, fmt.Errorf("Creating staging dir '%s': %s", stagingTmpDir, err)
	}

	err = os.MkdirAll(incomingTmpDir, 0700)
	if err != nil {
		return lockConfig, fmt.Errorf("Creating incoming dir '%s': %s", incomingTmpDir, err)
	}

	for _, contents := range d.opts.Contents {
		stagingDstPath := filepath.Join(stagingTmpDir, contents.Path)
		stagingDstPathParent := filepath.Dir(stagingDstPath)

		err := os.MkdirAll(stagingDstPathParent, 0700)
		if err != nil {
			return lockConfig, fmt.Errorf("Creating directory '%s': %s", stagingDstPathParent, err)
		}

		switch {
		case contents.Git != nil:
			d.ui.PrintLinef("%s + %s (git from %s@%s)",
				d.opts.Path, contents.Path, contents.Git.URL, contents.Git.Ref)

			gitLockConf, err := d.syncGit(*contents.Git, stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Syncing directory '%s' with git contents: %s", contents.Path, err)
			}

			err = FileFilter{contents}.Apply(stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Filtering paths in directory '%s': %s", contents.Path, err)
			}

			lockConfig.Contents = append(lockConfig.Contents, LockConfigContents{
				Path: contents.Path,
				Git:  &gitLockConf,
			})

		case contents.Manual != nil:
			d.ui.PrintLinef("%s + %s (manual)", d.opts.Path, contents.Path)

			srcPath := filepath.Join(d.opts.Path, contents.Path)

			err := os.Rename(srcPath, stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Moving directory '%s' to staging dir: %s", srcPath, err)
			}

			lockConfig.Contents = append(lockConfig.Contents, LockConfigContents{
				Path:   contents.Path,
				Manual: &LockConfigContentsManual{},
			})

		case contents.Directory != nil:
			d.ui.PrintLinef("%s + %s (directory)", d.opts.Path, contents.Path)

			err := dircopy.Copy(contents.Directory.Path, stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Copying another directory contents into directory '%s': %s", contents.Path, err)
			}

			err = FileFilter{contents}.Apply(stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Filtering paths in directory '%s': %s", contents.Path, err)
			}

			lockConfig.Contents = append(lockConfig.Contents, LockConfigContents{
				Path:      contents.Path,
				Directory: &LockConfigContentsDirectory{},
			})

		default:
			return lockConfig, fmt.Errorf("Unknown contents type for directory '%s' (known: git, manual)", contents.Path)
		}
	}

	err = os.RemoveAll(d.opts.Path)
	if err != nil {
		return lockConfig, fmt.Errorf("Deleting dir %s: %s", d.opts.Path, err)
	}

	err = os.Rename(stagingTmpDir, d.opts.Path)
	if err != nil {
		return lockConfig, fmt.Errorf("Moving staging directory '%s' to final location '%s': %s", stagingTmpDir, d.opts.Path, err)
	}

	return lockConfig, nil
}

func (d *Directory) syncGit(opts ConfigContentsGit, dstPath string) (LockConfigContentsGit, error) {
	gitLockConf := LockConfigContentsGit{}
	incomingTmpPath := filepath.Join(incomingTmpDir, "git")

	err := os.MkdirAll(incomingTmpPath, 0700)
	if err != nil {
		return gitLockConf, fmt.Errorf("Creating incoming dir '%s' for git fetching: %s", incomingTmpPath, err)
	}

	defer os.RemoveAll(incomingTmpPath)

	git := NewGit(opts, NewInfoLog(d.ui))

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

func (*Directory) singleLineCommitTitle(in string) string {
	pieces := strings.SplitN(in, "\n", 2)
	if len(pieces) > 1 {
		return pieces[0] + "..."
	}
	return pieces[0]
}

func (d *Directory) cleanUpTmpDir() error {
	err := os.RemoveAll(tmpDir)
	if err != nil {
		return fmt.Errorf("Deleting tmp dir '%s': %s", tmpDir, err)
	}
	return nil
}
