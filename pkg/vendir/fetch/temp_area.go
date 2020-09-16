package fetch

import (
	"os"
)

type TempArea interface {
	NewTempDir(string) (string, error)
	NewTempFile(string) (*os.File, error)
}
