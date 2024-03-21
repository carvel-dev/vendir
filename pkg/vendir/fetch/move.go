// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package fetch

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func MoveDir(path, dstPath string) error {
	err := os.RemoveAll(dstPath)
	if err != nil {
		return fmt.Errorf("Deleting dir %s: %s", dstPath, err)
	}

	err = os.Rename(path, dstPath)
	if err != nil {
		return fmt.Errorf("Moving directory '%s' to staging dir: %s", path, err)
	}

	return nil
}

// MoveFile moves a file to the destination, before that it removes all the contents of the destination folder
// If folder already existed it will keep the permissions if not it will use 0700
func MoveFile(path, dstPath string) error {
	folderPermission := os.FileMode(0700)
	_, err := os.Stat(dstPath)
	if !errors.Is(err, &os.PathError{}) {
		err := os.RemoveAll(dstPath)
		if err != nil {
			return fmt.Errorf("Deleting dir %s: %s", dstPath, err)
		}
	}

	err = os.Mkdir(dstPath, folderPermission)
	if err != nil {
		return fmt.Errorf("Creating dir %s: %s", dstPath, err)
	}

	err = os.Rename(path, filepath.Join(dstPath, filepath.Base(path)))
	if err != nil {
		return fmt.Errorf("Moving file '%s' to staging dir: %s", path, err)
	}

	return nil
}

func ScopedPath(path, subPath string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("Abs path: %s", err)
	}

	newPath, err := filepath.Abs(filepath.Join(path, subPath))
	if err != nil {
		return "", fmt.Errorf("Abs path: %s", err)
	}

	// Check that subPath is contained within path (disallow this scenario):
	//   ScopedPath("/root", "../root-trick/file1")
	//   "/root-trick/file1" == "/root" + "../root-trick/file1"
	if newPath != path && !strings.HasPrefix(newPath, path+string(filepath.Separator)) {
		return "", fmt.Errorf("Invalid path: %s", subPath)
	}

	return newPath, nil
}
