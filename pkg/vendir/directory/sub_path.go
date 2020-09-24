// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package directory

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	ctlfetch "github.com/k14s/vendir/pkg/vendir/fetch"
)

type SubPath struct {
	subPath string
}

func NewSubPath(subPath string) SubPath {
	return SubPath{subPath}
}

func (s SubPath) Extract(srcPath, dstPath string, tempArea ctlfetch.TempArea) error {
	newPath, err := ctlfetch.ScopedPath(srcPath, s.subPath)
	if err != nil {
		return err
	}

	err = s.checkDirExists(newPath, srcPath)
	if err != nil {
		return err
	}

	interDstPath, err := tempArea.NewTempDir("sub-path-extract")
	if err != nil {
		return err
	}

	defer os.RemoveAll(interDstPath)

	safeNewPath := filepath.Join(interDstPath, "dst")

	// Move out new path into safe dst to avoid cases
	// when newPath is contained within dstPath
	// (since dstPath gets deleted before newPath is moved to it)
	err = ctlfetch.MoveDir(newPath, safeNewPath)
	if err != nil {
		return err
	}

	return ctlfetch.MoveDir(safeNewPath, dstPath)
}

func (s SubPath) checkDirExists(path, srcPath string) error {
	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		return nil
	}

	hintMsg := ""

	missingPath, _ := s.findMissingDir(srcPath)
	if len(missingPath) > 0 {
		altDirs, _ := s.findAltDirs(srcPath, missingPath)
		if len(altDirs) > 0 {
			hintMsg = fmt.Sprintf(" (found other directories: %s)", strings.Join(altDirs, ", "))
		}
	}

	return fmt.Errorf("Expected directory '%s' (subpath) to exist%s", s.subPath, hintMsg)
}

func (s SubPath) findMissingDir(srcPath string) (string, error) {
	var pieces []string

	for _, piece := range filepath.SplitList(s.subPath) {
		pieces = append(pieces, piece)

		newPath, err := ctlfetch.ScopedPath(srcPath, filepath.Join(pieces...))
		if err != nil {
			return "", err
		}

		_, err = os.Stat(newPath)
		if os.IsNotExist(err) {
			return filepath.Join(pieces...), nil
		}
	}

	return "", nil
}

func (s SubPath) findAltDirs(srcPath, subPath string) ([]string, error) {
	parentDirOfSubPath := filepath.Dir(subPath)

	newPath, err := ctlfetch.ScopedPath(srcPath, parentDirOfSubPath)
	if err != nil {
		return nil, err
	}

	fileInfos, err := ioutil.ReadDir(newPath)
	if err != nil {
		return nil, err
	}

	var result []string

	for _, info := range fileInfos {
		if info.IsDir() {
			result = append(result, filepath.Join(parentDirOfSubPath, info.Name()))
		}
	}

	return result, nil
}
