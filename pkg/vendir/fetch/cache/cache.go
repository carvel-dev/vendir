// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/cache/resources"
)

// Cache functionality
type Cache interface {
	Hit(id string) (string, bool)
	Save(id string, src string) error
	CopyFrom(id string, dst string) error
}

// FolderCache cache storing the information into a folder in the OS
type FolderCache struct {
	folder  string
	maxSize resources.Quantity
}

// NewCache creates a new cache
// When cacheFolder is empty this constructor will provide a noop cache
func NewCache(cacheFolder string, maxContentCacheableSize string) (Cache, error) {
	if cacheFolder == "" {
		return &NoCache{}, nil
	}
	q, err := resources.ParseQuantity(maxContentCacheableSize)
	if err != nil {
		return nil, fmt.Errorf("Unable to process maximum amount allowed to cache: %s", err)
	}
	return &FolderCache{folder: cacheFolder, maxSize: q}, nil
}

// Hit checks if a particular entry in the cache is present
// Returns the path to the entry and a flag information if the entry was found or not
func (c FolderCache) Hit(id string) (string, bool) {
	folder := filepath.Join(c.folder, c.idToFolder(id))
	f, err := os.Stat(folder)
	if err != nil {
		if os.IsExist(err) {
			return "", false
		}
		return "", false
	}

	if !f.IsDir() {
		return "", false
	}

	return folder, true
}

// Save the folder from src in the cache using id
// If the cache entry exists it will remove it and create a new one
func (c FolderCache) Save(id string, src string) error {
	contentSize, err := c.dirSize(src)
	if err != nil {
		return fmt.Errorf("Unable to find size of folder to be cached: %s", err)
	}

	// When the content size is bigger than the maximum allowed amount it should not try to save into the cache
	if contentSize > c.maxSize.Value() {
		return nil
	}

	cachedContent, hit := c.Hit(id)
	if hit {
		err := os.RemoveAll(cachedContent)
		if err != nil {
			return err
		}
	}

	folder := filepath.Join(c.folder, c.idToFolder(id))
	return c.copyFolder(src, folder)
}

// CopyFrom the cache into a particular destination
func (c FolderCache) CopyFrom(id string, dst string) error {
	src, hit := c.Hit(id)
	if !hit {
		return fmt.Errorf("There is no cache entry for '%s'", id)
	}

	return c.copyFolder(src, dst)
}

func (c FolderCache) dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func (c FolderCache) copyFolder(src string, dst string) error {
	err := os.MkdirAll(dst, 0700)
	if err != nil {
		return err
	}

	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == src {
			return nil
		}
		fileName := strings.ReplaceAll(path, src, "")
		if len(fileName) > 0 && fileName[0] == filepath.Separator {
			fileName = fileName[1:]
		}

		p := filepath.Join(dst, fileName)
		if info.IsDir() {
			return os.Mkdir(p, info.Mode())
		}

		f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer f.Close()

		if runtime.GOOS != "windows" {
			err := os.Chmod(p, info.Mode())
			if err != nil {
				return err
			}
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		_, err = io.Copy(f, srcFile)
		return err
	})
}

func (c FolderCache) idToFolder(id string) string {
	return strings.ReplaceAll(id, ":", "-")
}

// NoCache is a noop cache
type NoCache struct{}

// Hit always returns false
func (c *NoCache) Hit(_ string) (string, bool) { return "", false }

// Save does nothing
func (c *NoCache) Save(_ string, _ string) error { return nil }

// CopyFrom does nothing
func (c *NoCache) CopyFrom(_ string, _ string) error { return nil }
