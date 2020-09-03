package directory

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func TempDir(name string) (string, error) {
	tmpDir := filepath.Join(incomingTmpDir, name)

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

func TempFile(pattern string) (*os.File, error) {
	return ioutil.TempFile(incomingTmpDir, pattern)
}
