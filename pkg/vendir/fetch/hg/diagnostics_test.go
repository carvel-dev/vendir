// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

// Package hg retrieves content from Mercurial repositories.
package hg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type diagnosticWrites struct{ values []string }

func (w *diagnosticWrites) Write(data []byte) (int, error) {
	w.values = append(w.values, string(data))
	return len(data), nil
}

func TestCommandDiagnosticsSkipEmptyStreams(t *testing.T) {
	const executableMode = 0700
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf '%s' \"$HG_TEST_OUTPUT\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hg"),
		[]byte(script), executableMode))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	const raw = "https://user:password-canary@example.com/?t=query-canary"
	t.Setenv("HG_TEST_OUTPUT", raw)
	var log diagnosticWrites
	command := Hg{infoLog: &log, env: os.Environ()}
	stdout, stderr, err := command.run([]string{"pull"}, dir)
	require.NoError(t, err)
	require.Equal(t, raw, stdout)
	require.Empty(t, stderr)
	const expectedWrites = 2
	require.Len(t, log.values, expectedWrites)
	require.Equal(t, "--> hg pull\n", log.values[0])
	require.NotContains(t, log.values[1], "canary")
}

func TestSilentCommandLogsOnlyItsInvocation(t *testing.T) {
	const executableMode = 0700
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hg"),
		[]byte("#!/bin/sh\nexit 0\n"), executableMode))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	var log diagnosticWrites
	command := Hg{infoLog: &log, env: os.Environ()}
	_, _, err := command.run([]string{"init"}, dir)
	require.NoError(t, err)
	require.Equal(t, []string{"--> hg init\n"}, log.values)
}
