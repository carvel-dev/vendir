package directory

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
)

type StagingDir struct {
	rootDir     string
	stagingDir  string
	incomingDir string
}

func NewStagingDir() StagingDir {
	rootDir := ".vendir-tmp"
	return StagingDir{
		rootDir:     rootDir,
		stagingDir:  filepath.Join(rootDir, "staging"),
		incomingDir: filepath.Join(rootDir, "incoming"),
	}
}

func (d StagingDir) Prepare() error {
	err := d.cleanUpAll()
	if err != nil {
		return err
	}

	err = os.MkdirAll(d.stagingDir, 0700)
	if err != nil {
		return fmt.Errorf("Creating staging dir '%s': %s", d.stagingDir, err)
	}

	err = os.MkdirAll(d.incomingDir, 0700)
	if err != nil {
		return fmt.Errorf("Creating incoming dir '%s': %s", d.incomingDir, err)
	}

	return nil
}

func (d StagingDir) NewChild(path string) (string, error) {
	childPath := filepath.Join(d.stagingDir, path)
	childPathParent := filepath.Dir(childPath)

	err := os.MkdirAll(childPathParent, 0700)
	if err != nil {
		return "", fmt.Errorf("Creating directory '%s': %s", childPathParent, err)
	}

	return childPath, nil
}

func (d StagingDir) Replace(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("Deleting dir %s: %s", path, err)
	}

	// Clean to avoid getting 'out/in/' from 'out/in/' instead of just 'out'
	parentPath := filepath.Dir(filepath.Clean(path))

	err = os.MkdirAll(parentPath, 0700)
	if err != nil {
		return fmt.Errorf("Creating final location parent dir %s: %s", parentPath, err)
	}

	err = os.Rename(d.stagingDir, path)
	if err != nil {
		return fmt.Errorf("Moving staging directory '%s' to final location '%s': %s", d.stagingDir, path, err)
	}

	return nil
}

func (d StagingDir) TempArea() StagingTempArea {
	return StagingTempArea{d.incomingDir}
}

func (d StagingDir) CleanUp() error {
	return d.cleanUpAll()
}

func (d StagingDir) cleanUpAll() error {
	err := os.RemoveAll(d.rootDir)
	if err != nil {
		return fmt.Errorf("Deleting tmp dir '%s': %s", d.rootDir, err)
	}
	return nil
}

type StagingTempArea struct {
	path string
}

var _ ctlfetch.TempArea = StagingTempArea{}

func (d StagingTempArea) NewTempDir(name string) (string, error) {
	tmpDir := filepath.Join(d.path, name)

	absTmpDir, err := filepath.Abs(tmpDir)
	if err != nil {
		return "", fmt.Errorf("Abs path '%s': %s", tmpDir, err)
	}

	err = os.Mkdir(absTmpDir, 0700)
	if err != nil {
		return "", fmt.Errorf("Creating incoming dir '%s' for %s: %s", absTmpDir, name, err)
	}

	return absTmpDir, nil
}

func (d StagingTempArea) NewTempFile(pattern string) (*os.File, error) {
	return ioutil.TempFile(d.path, pattern)
}
