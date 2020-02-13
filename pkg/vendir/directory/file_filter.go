package directory

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar"
)

type FileFilter struct {
	contents ConfigContents
}

func (d FileFilter) Apply(dirPath string) error {
	includePaths := d.scopePatterns(append([]string{}, d.contents.IncludePaths...), dirPath)
	excludePaths := d.scopePatterns(append([]string{}, d.contents.ExcludePaths...), dirPath)
	legalPaths := d.scopePatterns(append([]string{}, d.contents.LegalPathsWithDefaults()...), dirPath)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		var matched bool

		if len(includePaths) == 0 {
			matched = true
		}

		ok, err := d.matchAgainstPatterns(path, includePaths)
		if err != nil {
			return err
		}
		if ok {
			matched = true
		}

		ok, err = d.matchAgainstPatterns(path, excludePaths)
		if err != nil {
			return err
		}
		if ok {
			matched = false
		}

		ok, err = d.matchAgainstPatterns(path, legalPaths)
		if err != nil {
			return err
		}
		if ok {
			matched = true
		}

		if !matched {
			err := os.RemoveAll(path)
			if err != nil {
				return fmt.Errorf("Deleting file %s: %s", path, err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	_, err = d.deleteEmptyDirs(dirPath)
	return err
}

func (d FileFilter) scopePatterns(patterns []string, dirPath string) []string {
	for i, pattern := range patterns {
		patterns[i] = filepath.Join(dirPath, pattern)
	}
	return patterns
}

func (d FileFilter) matchAgainstPatterns(path string, patterns []string) (bool, error) {
	for _, pattern := range patterns {
		ok, err := doublestar.PathMatch(pattern, path)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func (d FileFilter) deleteEmptyDirs(dirPath string) (bool, error) {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return false, err
	}

	var hasFiles bool

	for _, file := range files {
		if file.IsDir() {
			hasFilesInside, err := d.deleteEmptyDirs(filepath.Join(dirPath, file.Name()))
			if err != nil {
				return false, err
			}
			if hasFilesInside {
				hasFiles = true
			}
		} else {
			hasFiles = true
		}
	}

	if !hasFiles {
		// not RemoveAll to double check directory is empty
		return false, os.Remove(dirPath)
	}

	return true, nil
}
