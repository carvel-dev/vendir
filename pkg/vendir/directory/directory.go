// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package directory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"

	"github.com/cppforlife/go-cli-ui/ui"
	dircopy "github.com/otiai10/copy"
	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
	ctlcache "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/cache"
	ctlgit "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/git"
	ctlghr "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/githubrelease"
	ctlhelmc "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/helmchart"
	ctlhg "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/hg"
	ctlhttp "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/http"
	ctlimg "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/image"
	ctlimgpkgbundle "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/imgpkgbundle"
	ctlinl "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/inline"
)

type Directory struct {
	opts       ctlconf.Directory
	lockConfig ctlconf.LockConfig
	ui         ui.UI
}

func NewDirectory(opts ctlconf.Directory, lockConfig ctlconf.LockConfig, ui ui.UI) *Directory {
	return &Directory{opts, lockConfig, ui}
}

type SyncOpts struct {
	RefFetcher     ctlfetch.RefFetcher
	GithubAPIToken string
	HelmBinary     string
	Cache          ctlcache.Cache
	Eager          bool
}

func createContentHash(contents ctlconf.DirectoryContents) (string, error) {
	yaml, err := yaml.Marshal(contents)
	if err != nil {
		return "", fmt.Errorf("error during hash creation for path '%s': %s", contents.Path, err)
	}
	hash := sha256.Sum256(yaml)
	hashStr := hex.EncodeToString(hash[:])
	return hashStr, nil
}

func (d *Directory) configUnchanged(path string, contents ctlconf.DirectoryContents) (bool, error) {
	hash, err := createContentHash(contents)
	if err != nil {
		return false, err
	}
	lockContents, _ := d.lockConfig.FindContents(path, contents.Path)
	if hash == lockContents.Hash {
		return true, nil
	}
	return false, nil
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
		// adds hash to lockfile of current content
		hash, err := createContentHash(contents)
		if err != nil {
			return lockConfig, err
		}

		lockDirContents := ctlconf.LockDirectoryContents{
			Path: contents.Path,
			Hash: hash,
		}
		// check if vendir config has changed. If not, skip syncing
		unchanged := false
		var oldLock ctlconf.LockDirectoryContents
		if contents.Lazy && syncOpts.Eager == false {
			unchanged, err = d.configUnchanged(d.opts.Path, contents)
			if err != nil {
				return lockConfig, err
			}
			if unchanged {
				d.ui.PrintLinef("Skipping fetch since config has not changed: %s%s", d.opts.Path, contents.Path)
				oldLock, _ = d.lockConfig.FindContents(d.opts.Path, contents.Path)
			}
		}

		skipFileFilter := false
		skipNewRootPath := false

		switch {
		case contents.Git != nil:
			gitSync := ctlgit.NewSync(*contents.Git, NewInfoLog(d.ui), syncOpts.RefFetcher)

			d.ui.PrintLinef("Fetching: %s + %s (git from %s)", d.opts.Path, contents.Path, gitSync.Desc())

			if unchanged == false {
				lock, err := gitSync.Sync(stagingDstPath, stagingDir.TempArea())
				if err != nil {
					return lockConfig, fmt.Errorf("Syncing directory '%s' with git contents: %s", contents.Path, err)
				}
				lockDirContents.Git = &lock
			} else {
				lockDirContents.Git = oldLock.Git
			}

		case contents.Hg != nil:
			hgSync := ctlhg.NewSync(*contents.Hg, NewInfoLog(d.ui), syncOpts.RefFetcher)

			d.ui.PrintLinef("Fetching: %s + %s (hg from %s)", d.opts.Path, contents.Path, hgSync.Desc())

			lock, err := hgSync.Sync(stagingDstPath, stagingDir.TempArea())
			if err != nil {
				return lockConfig, fmt.Errorf("Syncing directory '%s' with hg contents: %s", contents.Path, err)
			}

			lockDirContents.Hg = &lock

		case contents.HTTP != nil:
			d.ui.PrintLinef("Fetching: %s + %s (http from %s)", d.opts.Path, contents.Path, contents.HTTP.URL)

			lock, err := ctlhttp.NewSync(*contents.HTTP, syncOpts.RefFetcher).Sync(stagingDstPath, stagingDir.TempArea())
			if err != nil {
				return lockConfig, fmt.Errorf("Syncing directory '%s' with HTTP contents: %s", contents.Path, err)
			}

			lockDirContents.HTTP = &lock

		case contents.Image != nil:
			imageSync := ctlimg.NewSync(*contents.Image, syncOpts.RefFetcher, syncOpts.Cache)

			d.ui.PrintLinef("Fetching: %s + %s (image from %s)", d.opts.Path, contents.Path, imageSync.Desc())

			lock, err := imageSync.Sync(stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Syncing directory '%s' with image contents: %s", contents.Path, err)
			}

			lockDirContents.Image = &lock

		case contents.ImgpkgBundle != nil:
			imgpkgBundleSync := ctlimgpkgbundle.NewSync(*contents.ImgpkgBundle, syncOpts.RefFetcher, syncOpts.Cache)

			d.ui.PrintLinef("Fetching: %s + %s (imgpkgBundle from %s)", d.opts.Path, contents.Path, imgpkgBundleSync.Desc())

			lock, err := imgpkgBundleSync.Sync(stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Syncing directory '%s' with imgpkgBundle contents: %s", contents.Path, err)
			}

			lockDirContents.ImgpkgBundle = &lock

		case contents.GithubRelease != nil:
			sync, err := ctlghr.NewSync(*contents.GithubRelease, syncOpts.GithubAPIToken, syncOpts.RefFetcher)
			if err != nil {
				return lockConfig, err
			}

			desc, _ := sync.Desc()
			d.ui.PrintLinef("Fetching: %s + %s (github release %s)", d.opts.Path, contents.Path, desc)

			lock, err := sync.Sync(stagingDstPath, stagingDir.TempArea())
			if err != nil {
				return lockConfig, fmt.Errorf("Syncing directory '%s' with github release contents: %s", contents.Path, err)
			}

			lockDirContents.GithubRelease = &lock

		case contents.HelmChart != nil:
			helmChartSync := ctlhelmc.NewSync(*contents.HelmChart, syncOpts.HelmBinary, syncOpts.RefFetcher)

			d.ui.PrintLinef("Fetching: %s + %s (helm chart from %s)",
				d.opts.Path, contents.Path, helmChartSync.Desc())

			if unchanged == false {
				lock, err := helmChartSync.Sync(stagingDstPath, stagingDir.TempArea())
				if err != nil {
					return lockConfig, fmt.Errorf("Syncing directory '%s' with helm chart contents: %s", contents.Path, err)
				}
				lockDirContents.HelmChart = &lock
			} else {
				lockDirContents.HelmChart = oldLock.HelmChart
			}

		case contents.Manual != nil:
			d.ui.PrintLinef("Fetching: %s + %s (manual)", d.opts.Path, contents.Path)

			srcPath := filepath.Join(d.opts.Path, contents.Path)

			err := os.Rename(srcPath, stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Moving directory '%s' to staging dir: %s", srcPath, err)
			}

			lockDirContents.Manual = &ctlconf.LockDirectoryContentsManual{}
			skipFileFilter = true
			skipNewRootPath = true

		case contents.Directory != nil:
			d.ui.PrintLinef("Fetching: %s + %s (directory)", d.opts.Path, contents.Path)

			err := dircopy.Copy(contents.Directory.Path, stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Copying another directory contents into directory '%s': %s", contents.Path, err)
			}

			lockDirContents.Directory = &ctlconf.LockDirectoryContentsDirectory{}

		case contents.Inline != nil:
			d.ui.PrintLinef("Fetching: %s + %s (inline)", d.opts.Path, contents.Path)

			lock, err := ctlinl.NewSync(*contents.Inline, syncOpts.RefFetcher).Sync(stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Syncing directory '%s' with inline contents: %s", contents.Path, err)
			}

			lockDirContents.Inline = &lock

		default:
			return lockConfig, fmt.Errorf("Unknown contents type for directory '%s'", contents.Path)
		}

		if !unchanged {
			if !skipFileFilter {
				err = FileFilter{contents}.Apply(stagingDstPath)
				if err != nil {
					return lockConfig, fmt.Errorf("Filtering paths in directory '%s': %s", contents.Path, err)
				}
			}

			if !skipNewRootPath && len(contents.NewRootPath) > 0 {
				err = NewSubPath(contents.NewRootPath).Extract(stagingDstPath, stagingDstPath, stagingDir.TempArea())
				if err != nil {
					return lockConfig, fmt.Errorf("Changing to new root path '%s': %s", contents.Path, err)
				}
			}

			// Copy files from current source if values are supposed to be ignored
			err = stagingDir.CopyExistingFiles(d.opts.Path, stagingDstPath, contents.IgnorePaths)
			if err != nil {
				return lockConfig, fmt.Errorf("Copying existing content to staging '%s': %s", d.opts.Path, err)
			}

			// after everything else is done, ensure the inner dir's access perms are set
			// chmod to the content's permission, fall back to the directory's
			err = maybeChmod(stagingDstPath, contents.Permissions, d.opts.Permissions)
			if err != nil {
				return lockConfig, fmt.Errorf("chmod on '%s': %s", stagingDstPath, err)
			}
		}

		lockConfig.Contents = append(lockConfig.Contents, lockDirContents)
	}

	err = stagingDir.Replace(d.opts.Path)
	if err != nil {
		return lockConfig, err
	}

	// after everything else is done, ensure the outer dir's access perms are set
	err = maybeChmod(d.opts.Path, d.opts.Permissions)
	if err != nil {
		return lockConfig, fmt.Errorf("chmod on '%s': %s", d.opts.Path, err)
	}

	return lockConfig, nil
}

//

// maybeChmod will chmod the path with the first non-nil permission provided.
// If no permission is handed in or all of them are nil, no chmod will be done.
func maybeChmod(path string, potentialPerms ...*os.FileMode) error {
	for _, p := range potentialPerms {
		if p != nil {
			return os.Chmod(path, *p)
		}
	}

	return nil
}
