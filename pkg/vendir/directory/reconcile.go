// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package directory

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// preservedPaths is a tree of path segments naming what vendir does not
// manage within a reconciled dir, e.g. manually managed contents. Those are
// neither deleted nor descended into.
type preservedPaths map[string]preservedPaths

func newPreservedPaths(paths []string) preservedPaths {
	root := preservedPaths{}
	for _, path := range paths {
		root.add(path)
	}

	return root
}

func (p preservedPaths) add(path string) {
	node := p
	for _, segment := range strings.Split(filepath.ToSlash(path), "/") {
		child, found := node[segment]
		if !found {
			child = preservedPaths{}
			node[segment] = child
		}
		node = child
	}
}

// reconcileDir makes the dir at dstPath hold exactly the contents of the dir
// at srcPath, consuming srcPath in the process. Entries that already hold the
// wanted contents are left in place instead of being written again, so that
// their modification times stay stable for tooling that watches them.
func reconcileDir(srcPath, dstPath string, preserved preservedPaths) error {
	srcInfo, err := os.Lstat(srcPath)
	if err != nil {
		return fmt.Errorf("Reading staged dir '%s': %s", srcPath, err)
	}

	dstInfo, err := existingDir(dstPath)
	if err != nil {
		return err
	}

	// Nothing to compare against, and nothing to keep either since preserved
	// paths only ever name contents that are already there, so the staged dir
	// is moved over as a whole
	if dstInfo == nil {
		return moveEntry(srcPath, dstPath, nil)
	}

	err = reconcileDirEntries(srcPath, dstPath, preserved)
	if err != nil {
		return err
	}

	return maybeChmodTo(dstPath, dstInfo, srcInfo)
}

// existingDir returns the info of the dir at path, or nil when there is none.
// Anything else found at path, e.g. a file or a symlink, is deleted, since it
// cannot be updated in place.
func existingDir(path string) (os.FileInfo, error) {
	info, err := lstatOrNil(path)
	if err != nil {
		return nil, fmt.Errorf("Reading '%s': %s", path, err)
	}
	if info == nil {
		return nil, nil
	}
	if info.IsDir() {
		return info, nil
	}

	return nil, removePath(path)
}

func reconcileDirEntries(srcPath, dstPath string,
	preserved preservedPaths) error {
	srcEntries, err := os.ReadDir(srcPath)
	if err != nil {
		return fmt.Errorf("Reading staged dir '%s': %s", srcPath, err)
	}

	err = deleteStaleEntries(dstPath, keepNames(srcEntries, preserved))
	if err != nil {
		return err
	}

	for _, entry := range srcEntries {
		err := reconcileEntry(
			filepath.Join(srcPath, entry.Name()),
			filepath.Join(dstPath, entry.Name()),
			preserved[entry.Name()])
		if err != nil {
			return err
		}
	}

	return nil
}

// deleteStaleEntries drops everything within dstPath that is not to be kept
func deleteStaleEntries(dstPath string, keep nameSet) error {
	dstEntries, err := os.ReadDir(dstPath)
	if err != nil {
		return fmt.Errorf("Reading dir '%s': %s", dstPath, err)
	}

	for _, entry := range dstEntries {
		if _, found := keep[entry.Name()]; found {
			continue
		}

		err := removePath(filepath.Join(dstPath, entry.Name()))
		if err != nil {
			return err
		}
	}

	return nil
}

// nameSet holds directory entry names
type nameSet map[string]struct{}

// keepNames returns the names that have to survive within the reconciled dir:
// the staged ones plus the ones vendir does not manage
func keepNames(staged []os.DirEntry, preserved preservedPaths) nameSet {
	names := nameSet{}
	for _, entry := range staged {
		names[entry.Name()] = struct{}{}
	}
	for name := range preserved {
		names[name] = struct{}{}
	}

	return names
}

func reconcileEntry(srcPath, dstPath string, preserved preservedPaths) error {
	srcInfo, err := os.Lstat(srcPath)
	if err != nil {
		return fmt.Errorf("Reading staged '%s': %s", srcPath, err)
	}

	if srcInfo.IsDir() {
		return reconcileDir(srcPath, dstPath, preserved)
	}

	dstInfo, err := lstatOrNil(dstPath)
	if err != nil {
		return fmt.Errorf("Reading '%s': %s", dstPath, err)
	}

	upToDate, err := sameEntry(srcPath, dstPath, srcInfo, dstInfo)
	if err != nil {
		return err
	}
	if upToDate {
		return maybeChmodTo(dstPath, dstInfo, srcInfo)
	}

	return moveEntry(srcPath, dstPath, dstInfo)
}

// sameEntry reports whether the entry at dst already holds what the staged
// entry at src has to offer, permissions aside
func sameEntry(src, dst string, srcInfo, dstInfo os.FileInfo) (bool, error) {
	switch {
	case dstInfo == nil:
		return false, nil

	case srcInfo.Mode()&os.ModeSymlink != 0:
		return sameSymlinkTarget(src, dst, dstInfo)

	case srcInfo.Mode().IsRegular():
		if !dstInfo.Mode().IsRegular() {
			return false, nil
		}
		if dstInfo.Size() != srcInfo.Size() {
			return false, nil
		}

		return sameFileContents(src, dst)

	default:
		// Anything else, e.g. a socket, is always moved over
		return false, nil
	}
}

// moveEntry moves the staged entry over the final location, carrying its
// modification time along
func moveEntry(srcPath, dstPath string, dstInfo os.FileInfo) error {
	// Rename cannot replace a dir, and on Windows not an existing file either
	if dstInfo != nil {
		err := removePath(dstPath)
		if err != nil {
			return err
		}
	}

	err := os.Rename(srcPath, dstPath)
	if err != nil {
		return fmt.Errorf("Moving staged '%s' to '%s': %s",
			srcPath, dstPath, err)
	}

	return nil
}

// sameFileContents reports whether both files hold the same bytes.
// Permissions are deliberately left out, since they can be changed without
// writing the file again.
func sameFileContents(src, dst string) (bool, error) {
	srcSum, err := fileSum(src)
	if err != nil {
		return false, err
	}

	dstSum, err := fileSum(dst)
	if err != nil {
		return false, err
	}

	return srcSum == dstSum, nil
}

func fileSum(path string) ([sha256.Size]byte, error) {
	var sum [sha256.Size]byte

	file, err := os.Open(path)
	if err != nil {
		return sum, fmt.Errorf("Opening file '%s': %s", path, err)
	}
	defer file.Close()

	hash := sha256.New()

	_, err = io.Copy(hash, file)
	if err != nil {
		return sum, fmt.Errorf("Reading file '%s': %s", path, err)
	}

	return [sha256.Size]byte(hash.Sum(nil)), nil
}

func sameSymlinkTarget(src, dst string, dstInfo os.FileInfo) (bool, error) {
	if dstInfo.Mode()&os.ModeSymlink == 0 {
		return false, nil
	}

	srcTarget, err := os.Readlink(src)
	if err != nil {
		return false, fmt.Errorf("Reading staged symlink '%s': %s", src, err)
	}

	dstTarget, err := os.Readlink(dst)
	if err != nil {
		return false, fmt.Errorf("Reading symlink '%s': %s", dst, err)
	}

	return srcTarget == dstTarget, nil
}

// maybeChmodTo aligns the permissions of an entry that was left in place with
// the staged ones
func maybeChmodTo(dstPath string, dstInfo, srcInfo os.FileInfo) error {
	// A chmod would follow the link and touch its target instead
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return nil
	}

	if dstInfo.Mode().Perm() == srcInfo.Mode().Perm() {
		return nil
	}

	err := os.Chmod(dstPath, srcInfo.Mode().Perm())
	if err != nil {
		return fmt.Errorf("Chmod on '%s': %s", dstPath, err)
	}

	return nil
}

func removePath(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("Deleting '%s': %s", path, err)
	}

	return nil
}

// lstatOrNil returns a nil FileInfo, and no error, when path does not exist
func lstatOrNil(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return info, nil
}
