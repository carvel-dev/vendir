// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package fetch

import (
	"os"
)

type TempArea interface {
	NewTempDir(string) (string, error)
	NewTempFile(string) (*os.File, error)
}
