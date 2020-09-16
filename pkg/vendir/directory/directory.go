package directory

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cppforlife/go-cli-ui/ui"
	ctlconf "github.com/k14s/vendir/pkg/vendir/config"
	ctlfetch "github.com/k14s/vendir/pkg/vendir/fetch"
	ctlgit "github.com/k14s/vendir/pkg/vendir/fetch/git"
	ctlghr "github.com/k14s/vendir/pkg/vendir/fetch/githubrelease"
	ctlhelmc "github.com/k14s/vendir/pkg/vendir/fetch/helmchart"
	ctlhttp "github.com/k14s/vendir/pkg/vendir/fetch/http"
	ctlimg "github.com/k14s/vendir/pkg/vendir/fetch/image"
	dircopy "github.com/otiai10/copy"
)

type Directory struct {
	opts ctlconf.Directory
	ui   ui.UI
}

func NewDirectory(opts ctlconf.Directory, ui ui.UI) *Directory {
	return &Directory{opts, ui}
}

type SyncOpts struct {
	RefFetcher     ctlfetch.RefFetcher
	GithubAPIToken string
	HelmBinary     string
}

func (d *Directory) Sync(syncOpts SyncOpts) (ctlconf.LockDirectory, error) {
	lockConfig := ctlconf.LockDirectory{Path: d.opts.Path}

	stagingDir := NewStagingDir()

	err := stagingDir.Prepare()
	if err != nil {
		return lockConfig, err
	}

	defer stagingDir.CleanUp()

	for _, contents := range d.opts.Contents {
		stagingDstPath, err := stagingDir.NewChild(contents.Path)
		if err != nil {
			return lockConfig, err
		}

		switch {
		case contents.Git != nil:
			d.ui.PrintLinef("%s + %s (git from %s@%s)",
				d.opts.Path, contents.Path, contents.Git.URL, contents.Git.Ref)

			gitLockConf, err := ctlgit.NewSync(*contents.Git, NewInfoLog(d.ui)).Sync(stagingDstPath, stagingDir.TempArea())
			if err != nil {
				return lockConfig, fmt.Errorf("Syncing directory '%s' with git contents: %s", contents.Path, err)
			}

			err = FileFilter{contents}.Apply(stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Filtering paths in directory '%s': %s", contents.Path, err)
			}

			lockConfig.Contents = append(lockConfig.Contents, ctlconf.LockDirectoryContents{
				Path: contents.Path,
				Git:  &gitLockConf,
			})

		case contents.HTTP != nil:
			d.ui.PrintLinef("%s + %s (http from %s)", d.opts.Path, contents.Path, contents.HTTP.URL)

			httpLockConf, err := ctlhttp.NewSync(*contents.HTTP, syncOpts.RefFetcher).Sync(stagingDstPath, stagingDir.TempArea())
			if err != nil {
				return lockConfig, fmt.Errorf("Syncing directory '%s' with HTTP contents: %s", contents.Path, err)
			}

			err = FileFilter{contents}.Apply(stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Filtering paths in directory '%s': %s", contents.Path, err)
			}

			lockConfig.Contents = append(lockConfig.Contents, ctlconf.LockDirectoryContents{
				Path: contents.Path,
				HTTP: &httpLockConf,
			})

		case contents.Image != nil:
			d.ui.PrintLinef("%s + %s (image from %s)", d.opts.Path, contents.Path, contents.Image.URL)

			imageLockConf, err := ctlimg.NewSync(*contents.Image, syncOpts.RefFetcher).Sync(stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Syncing directory '%s' with image contents: %s", contents.Path, err)
			}

			err = FileFilter{contents}.Apply(stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Filtering paths in directory '%s': %s", contents.Path, err)
			}

			lockConfig.Contents = append(lockConfig.Contents, ctlconf.LockDirectoryContents{
				Path:  contents.Path,
				Image: &imageLockConf,
			})

		case contents.GithubRelease != nil:
			sync := ctlghr.NewSync(*contents.GithubRelease, syncOpts.GithubAPIToken)

			desc, _, _ := sync.DescAndURL()
			d.ui.PrintLinef("%s + %s (github release %s)", d.opts.Path, contents.Path, desc)

			lockConf, err := sync.Sync(stagingDstPath, stagingDir.TempArea())
			if err != nil {
				return lockConfig, fmt.Errorf("Syncing directory '%s' with github release contents: %s", contents.Path, err)
			}

			err = FileFilter{contents}.Apply(stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Filtering paths in directory '%s': %s", contents.Path, err)
			}

			lockConfig.Contents = append(lockConfig.Contents, ctlconf.LockDirectoryContents{
				Path:          contents.Path,
				GithubRelease: &lockConf,
			})

		case contents.HelmChart != nil:
			helmChartSync := ctlhelmc.NewSync(*contents.HelmChart, syncOpts.HelmBinary, syncOpts.RefFetcher)

			d.ui.PrintLinef("%s + %s (helm chart from %s)",
				d.opts.Path, contents.Path, helmChartSync.Desc())

			chartLockConf, err := helmChartSync.Sync(stagingDstPath, stagingDir.TempArea())
			if err != nil {
				return lockConfig, fmt.Errorf("Syncing directory '%s' with helm chart contents: %s", contents.Path, err)
			}

			err = FileFilter{contents}.Apply(stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Filtering paths in directory '%s': %s", contents.Path, err)
			}

			lockConfig.Contents = append(lockConfig.Contents, ctlconf.LockDirectoryContents{
				Path:      contents.Path,
				HelmChart: &chartLockConf,
			})

		case contents.Manual != nil:
			d.ui.PrintLinef("%s + %s (manual)", d.opts.Path, contents.Path)

			srcPath := filepath.Join(d.opts.Path, contents.Path)

			err := os.Rename(srcPath, stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Moving directory '%s' to staging dir: %s", srcPath, err)
			}

			lockConfig.Contents = append(lockConfig.Contents, ctlconf.LockDirectoryContents{
				Path:   contents.Path,
				Manual: &ctlconf.LockDirectoryContentsManual{},
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

			lockConfig.Contents = append(lockConfig.Contents, ctlconf.LockDirectoryContents{
				Path:      contents.Path,
				Directory: &ctlconf.LockDirectoryContentsDirectory{},
			})

		default:
			return lockConfig, fmt.Errorf("Unknown contents type for directory '%s' (known: git, manual)", contents.Path)
		}
	}

	err = stagingDir.Replace(d.opts.Path)
	if err != nil {
		return lockConfig, err
	}

	return lockConfig, nil
}
