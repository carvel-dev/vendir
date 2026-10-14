// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package directory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	ctlfetch "carvel.dev/vendir/pkg/vendir/fetch"
	ctlcache "carvel.dev/vendir/pkg/vendir/fetch/cache"
	ctlgit "carvel.dev/vendir/pkg/vendir/fetch/git"
	ctlghr "carvel.dev/vendir/pkg/vendir/fetch/githubrelease"
	ctlhelmc "carvel.dev/vendir/pkg/vendir/fetch/helmchart"
	ctlhg "carvel.dev/vendir/pkg/vendir/fetch/hg"
	ctlhttp "carvel.dev/vendir/pkg/vendir/fetch/http"
	ctlimg "carvel.dev/vendir/pkg/vendir/fetch/image"
	ctlimgpkgbundle "carvel.dev/vendir/pkg/vendir/fetch/imgpkgbundle"
	ctlinl "carvel.dev/vendir/pkg/vendir/fetch/inline"
	"github.com/cppforlife/go-cli-ui/ui"
	dircopy "github.com/otiai10/copy"
)

type Directory struct {
	opts          ctlconf.Directory
	lockDirectory ctlconf.LockDirectory
	ui            ui.UI
}

func NewDirectory(opts ctlconf.Directory, lockDirectory ctlconf.LockDirectory, ui ui.UI) *Directory {
	return &Directory{opts, lockDirectory, ui}
}

type SyncOpts struct {
	RefFetcher     ctlfetch.RefFetcher
	GithubAPIToken string
	HelmBinary     string
	Cache          ctlcache.Cache
	Lazy           bool
	Partial        bool
}

func createConfigDigest(contents ctlconf.DirectoryContents) (string, error) {
	yaml, err := yaml.Marshal(contents)
	if err != nil {
		return "", fmt.Errorf("error during creating for config digest for path '%s': %s", contents.Path, err)
	}
	digest := sha256.Sum256(yaml)
	digestStr := hex.EncodeToString(digest[:])
	return digestStr, nil
}

func (d *Directory) Sync(syncOpts SyncOpts) (ctlconf.LockDirectory, error) {
	lockConfig := ctlconf.LockDirectory{Path: d.opts.Path}

	stagingDir, err := NewStagingDir()
	if err != nil {
		return lockConfig, err
	}

	err = stagingDir.Prepare()
	if err != nil {
		return lockConfig, err
	}

	defer stagingDir.CleanUp()

	for _, contents := range d.opts.Contents {
		stagingDstPath, err := stagingDir.NewChild(contents.Path)
		if err != nil {
			return lockConfig, err
		}

		// creates config digest for current content config
		configDigest, err := createConfigDigest(contents)
		if err != nil {
			return lockConfig, err
		}

		lockDirContents := ctlconf.LockDirectoryContents{
			Path: contents.Path,
		}

		// error is safe to ignore, since it indicates that no lock file entry for the given path exists
		oldLockContents, _ := d.lockDirectory.FindContents(contents.Path)

		// if lazy sync is enabled in both config and sync options and the config was not changed since the last sync,
		// skip fetching
		if syncOpts.Lazy && contents.Lazy && oldLockContents.ConfigDigest == configDigest {
			d.ui.PrintLinef("Skipping fetch: %s + %s (flagged as lazy, config has not changed since last sync)", d.opts.Path, contents.Path)
			lockConfig.Contents = append(lockConfig.Contents, oldLockContents)
			// copy previously fetched contents to staging dir
			err = dircopy.Copy(filepath.Join(d.opts.Path, contents.Path), stagingDstPath)
			if err != nil {
				return lockConfig, fmt.Errorf("Lazy content missing. Run sync with --lazy=false to fix. '%s': %s", d.opts.Path, err)
			}
			continue
		}

		skipFileFilter := false
		skipNewRootPath := false

		switch {
		case contents.Git != nil:
			contentPaths := contents.ContentPaths
			gitSync := ctlgit.NewSync(*contents.Git, contentPaths, NewInfoLog(d.ui), syncOpts.RefFetcher, syncOpts.Cache)

			d.ui.PrintLinef("Fetching: %s + %s (git from %s)", d.opts.Path, contents.Path, gitSync.Desc())

			lock, err := gitSync.Sync(stagingDstPath, stagingDir.TempArea())
			if err != nil {
				return lockConfig, fmt.Errorf("Syncing directory '%s' with git contents: %s", contents.Path, err)
			}
			lockDirContents.Git = &lock

		case contents.Hg != nil:
			hgSync := ctlhg.NewSync(*contents.Hg, NewInfoLog(d.ui), syncOpts.RefFetcher, syncOpts.Cache)

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

			lock, err := helmChartSync.Sync(stagingDstPath, stagingDir.TempArea())
			if err != nil {
				return lockConfig, fmt.Errorf("Syncing directory '%s' with helm chart contents: %s", contents.Path, err)
			}
			lockDirContents.HelmChart = &lock

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

		// config digest is always added if lazy syncing is enabled in the config
		if contents.Lazy {
			lockDirContents.ConfigDigest = configDigest
		}

		lockConfig.Contents = append(lockConfig.Contents, lockDirContents)

		if syncOpts.Partial {
			err = stagingDir.PartialRepace(contents.Path, filepath.Join(d.opts.Path, contents.Path))
			if err != nil {
				return lockConfig, err
			}
		}
	}

	if !syncOpts.Partial {
		err = stagingDir.Replace(d.opts.Path)
		if err != nil {
			return lockConfig, err
		}
	}

	// after everything else is done, ensure the outer dir's access perms are set
	err = maybeChmod(d.opts.Path, d.opts.Permissions)
	if err != nil {
		return lockConfig, fmt.Errorf("chmod on '%s': %s", d.opts.Path, err)
	}

	return lockConfig, nil
}

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
