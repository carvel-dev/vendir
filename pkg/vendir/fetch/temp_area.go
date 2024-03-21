// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package fetch

import (
	"os"
)

type TempArea interface {
	NewTempDir(string) (string, error)
	NewTempFile(string) (*os.File, error)
}
