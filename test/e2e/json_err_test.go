// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONErr(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	var stderr bytes.Buffer
	out, err := vendir.RunWithOpts([]string{"sync", "--json", "-f", "missing.yml"}, RunOpts{
		Dir:          t.TempDir(),
		StderrWriter: &stderr,
		AllowError:   true,
	})

	require.EqualError(t, err, "Execution error: stdout: '' stderr: '' error: 'exit status 1'")
	require.Empty(t, out)
	require.Equal(t, "vendir: Error: Reading config 'missing.yml': open missing.yml: no such file or directory\n", stderr.String())
}
