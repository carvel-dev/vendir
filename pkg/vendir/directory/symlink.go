// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package directory

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// RemoveDanglingSymlinks removes symlinks within path whose targets do not exist.
func RemoveDanglingSymlinks(path string) error {
	return filepath.WalkDir(path, func(entryPath string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info.Type()&os.ModeSymlink == os.ModeSymlink {
			_, statErr := os.Stat(entryPath)
			if statErr != nil && os.IsNotExist(statErr) {
				return os.Remove(entryPath)
			}
		}
		return nil
	})
}

// ValidateSymlinks enforces that symlinks inside the given path resolve to inside the path
func ValidateSymlinks(path string) error {
	absRoot, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rootSegments := strings.Split(absRoot, string(os.PathSeparator))
	return filepath.WalkDir(path, func(path string, info fs.DirEntry, _ error) error {
		if info.Type()&os.ModeSymlink == os.ModeSymlink {
			resolvedPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return fmt.Errorf("Unable to resolve symlink: %w", err)
			}
			absPath, err := filepath.Abs(resolvedPath)
			if err != nil {
				return err
			}
			pathSegments := strings.Split(absPath, string(os.PathSeparator))

			if len(rootSegments) > len(pathSegments) {
				return fmt.Errorf("Invalid symlink found to outside parent directory: %q", absPath)
			}
			for i, segment := range rootSegments {
				if pathSegments[i] != segment {
					return fmt.Errorf("Invalid symlink found to outside parent directory: %q", absPath)
				}
			}
		}
		return nil
	})

}
