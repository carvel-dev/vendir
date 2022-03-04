// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package directory

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ValidateSymlinks enforces that symlinks inside the given path resolve to inside the path
func ValidateSymlinks(path string) error {
	absRoot, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	return filepath.WalkDir(path, func(path string, info fs.DirEntry, err error) error {
		if info.Type()&os.ModeSymlink == os.ModeSymlink {
			resolvedPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return fmt.Errorf("Unable to resolve symlink: %w", err)
			}
			absPath, err := filepath.Abs(resolvedPath)
			if err != nil {
				return err
			}
			if !strings.HasPrefix(absPath, absRoot) {
				return fmt.Errorf("Invalid symlink found to outside parent directory: %q", absPath)
			}
		}
		return nil
	})

}
