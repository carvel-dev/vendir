package fetch

import (
	"fmt"
	"os"
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
