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
)

// Cache functionality
type Cache interface {
	Has(artifactType string, id string) (string, bool)
	Save(artifactType string, id string, src string) error
	CopyFrom(artifactType string, id string, dst string) error
}

// FolderCache cache storing the information into a folder in the OS
type FolderCache struct {
	folder string
}

// NewCache creates a new cache
// When cacheFolder is empty this constructor will provide a noop cache
func NewCache(cacheFolder string) Cache {
	if cacheFolder == "" {
		return &NoCache{}
	}
	return &FolderCache{folder: cacheFolder}
}

// Has checks if a particular entry in the cache is present
// Returns the path to the entry and a flag information if the entry was found or not
func (c FolderCache) Has(artifactType string, id string) (string, bool) {
	folder := filepath.Join(c.folder, c.idToFolder(artifactType, id))
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
func (c FolderCache) Save(artifactType string, id string, src string) error {
	cachedContent, hit := c.Has(artifactType, id)
	if hit {
		err := os.RemoveAll(cachedContent)
		if err != nil {
			return err
		}
	}

	folder := filepath.Join(c.folder, c.idToFolder(artifactType, id))
	err := os.MkdirAll(folder, 0700)
	if err != nil {
		return err
	}
	return c.copyFolder(src, folder)
}

// CopyFrom the cache into a particular destination
func (c FolderCache) CopyFrom(artifactType string, id string, dst string) error {
	src, hit := c.Has(artifactType, id)
	if !hit {
		return fmt.Errorf("There is no cache entry for '%s'", id)
	}

	return c.copyFolder(src, dst)
}

func (c FolderCache) copyFolder(src string, dst string) error {
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
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

func (c FolderCache) idToFolder(artifactType string, id string) string {
	normalizedID := strings.ReplaceAll(id, ":", "-")
	return filepath.Join(".vendir-cache", "fetcher", artifactType, normalizedID)

}

// NoCache is a noop cache
type NoCache struct{}

// Has always returns false
func (c *NoCache) Has(_, _ string) (string, bool) { return "", false }

// Save does nothing
func (c *NoCache) Save(_, _, _ string) error { return nil }

// CopyFrom does nothing
func (c *NoCache) CopyFrom(_, _, _ string) error { return nil }
