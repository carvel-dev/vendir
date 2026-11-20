// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	ctlfetch "carvel.dev/vendir/pkg/vendir/fetch"
	ctlcache "carvel.dev/vendir/pkg/vendir/fetch/cache"
)

const gitCacheType = "git-bundle"

type Sync struct {
	opts         ctlconf.DirectoryContentsGit
	contentPaths ctlconf.ContentPaths
	log          io.Writer
	refFetcher   ctlfetch.RefFetcher
	cache        ctlcache.Cache
}

func NewSync(opts ctlconf.DirectoryContentsGit, contentPaths ctlconf.ContentPaths,
	log io.Writer, refFetcher ctlfetch.RefFetcher, cache ctlcache.Cache,
) Sync {
	return Sync{opts, contentPaths, log, refFetcher, cache}
}

func (d Sync) Desc() string {
	ref := "?"
	switch {
	case len(d.opts.Ref) > 0:
		ref = d.opts.Ref
	case d.opts.RefSelection != nil:
		ref = "ref=" + d.opts.RefSelection.Description()
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

	cacheID := fmt.Sprintf("%x", sha256.Sum256([]byte(d.opts.URL)))

	git := NewGit(d.opts, d.contentPaths, d.log, d.refFetcher)

	var bundle string
	if cacheEntry, hasCache := d.cache.Has(gitCacheType, cacheID); hasCache {
		bundle = filepath.Join(cacheEntry, "bundle")
	}

	info, err := git.Retrieve(incomingTmpPath, tempArea, bundle)
	if err != nil {
		return gitLockConf, fmt.Errorf("Fetching git repository: %s", err)
	}

	gitLockConf.SHA = info.SHA
	gitLockConf.Tags = info.Tags
	gitLockConf.CommitTitle = d.singleLineCommitTitle(info.CommitTitle)

	if _, ok := d.cache.(*ctlcache.NoCache); !ok {
		// attempt to save a bundle to the cache
		bundleDir, err := tempArea.NewTempDir("bundleCache")
		if err != nil {
			return gitLockConf, err
		}
		defer os.RemoveAll(bundleDir)
		bundle := filepath.Join(bundleDir, "bundle")
		// get all refs

		out, _, err := git.cmdRunner.Run([]string{"for-each-ref", "--format=%(refname)"}, nil, incomingTmpPath)
		if err != nil {
			return gitLockConf, err
		}
		var refs []string
		for _, ref := range strings.Split(string(out), "\n") {
			ref = strings.TrimSpace(ref)
			if ref != "" {
				refs = append(refs, ref)
			}
		}
		if _, _, err := git.cmdRunner.Run(append([]string{"bundle", "create", bundle}, refs...), nil, incomingTmpPath); err != nil {
			return gitLockConf, err
		}
		if err := d.cache.Save(gitCacheType, cacheID, bundleDir); err != nil {
			return gitLockConf, err
		}
	}

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
