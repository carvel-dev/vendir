// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitURLArgumentsAndChildOutputAreSanitized(t *testing.T) {
	const executableMode = 0700
	const urlArgument = 3
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf '%s\n' \"$@\" >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), executableMode); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	raw := "https://user:password-canary@example.com/repo?token=query-canary"
	args := []string{"remote", "add", "origin", raw}
	safe := redactGitArgs(args)
	if strings.Contains(strings.Join(safe, " "), "canary") || args[urlArgument] != raw {
		t.Fatal("unsafe argument presentation or modified input")
	}
	var log bytes.Buffer
	runner := runner{infoLog: &log}
	_, _, err := runner.Run(args, nil, dir)
	if err == nil {
		t.Fatal("expected child failure")
	}
	if strings.Contains(log.String()+err.Error(), "canary") {
		t.Fatal("child diagnostic exposed credentials")
	}
}
