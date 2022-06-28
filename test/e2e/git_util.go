// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"os/exec"
	"testing"
)

func gitDiffExamplesDir(t *testing.T, path string, cmdDir string) string {
	_, _, err := execGit([]string{"add", "--all", "--", path}, cmdDir)
	if err != nil {
		t.Fatalf("Expected no err")
	}

	diffOut, _, err := execGit([]string{"diff", "--cached", "--", path}, cmdDir)
	if err != nil {
		t.Fatalf("Expected no err")
	}

	return diffOut
}

func execGit(args []string, dir string) (string, string, error) {
	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = &stdoutBs
	cmd.Stderr = &stderrBs

	err := cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("Git %s: %s (stderr: %s)", args, err, stderrBs.String())
	}

	return stdoutBs.String(), stderrBs.String(), nil
}
