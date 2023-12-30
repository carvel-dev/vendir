// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConcurrentProcesses(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	vendir := Vendir{t, env.BinaryPath, logger}
	tmpRoot := t.TempDir()
	processes := 2

	yaml := `
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: vendor/dest-%d
  contents:
  - path: .
    git:
      url: https://github.com/carvel-dev/ytt
      ref: v0.27.x
      depth: 1
    includePaths:
      - README.md
      - pkg/version/version.go`

	logger.Section("execute several vendir processes concurrently", func() {
		wg := sync.WaitGroup{}
		for i := 0; i < processes; i++ {
			wg.Add(1)
			go func(n int, t *testing.T, wg *sync.WaitGroup) {
				defer wg.Done()
				_, err := vendir.RunWithOpts(
					[]string{"sync", "-f", "-"},
					RunOpts{
						Dir:         tmpRoot,
						StdinReader: strings.NewReader(fmt.Sprintf(yaml, n)),
						AllowError:  true,
					})
				require.NoError(t, err)
			}(i, t, &wg)
		}
		wg.Wait()
		for i := 0; i < processes; i++ {
			_, err := os.Stat(fmt.Sprintf("%s/vendor/dest-%d/README.md", tmpRoot, i))
			require.NoError(t, err)
		}
	})
}
