// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package directory

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Git struct {
	opts    ConfigContentsGit
	infoLog io.Writer
}

func NewGit(opts ConfigContentsGit, infoLog io.Writer) *Git {
	return &Git{opts, infoLog}
}

type GitInfo struct {
	SHA         string
	CommitTitle string
}

func (t *Git) Retrieve(dstPath string) (GitInfo, error) {
	if len(t.opts.URL) == 0 {
		return GitInfo{}, fmt.Errorf("Expected non-empty URL")
	}
	if len(t.opts.Ref) == 0 {
		return GitInfo{}, fmt.Errorf("Expected non-empty ref (could be branch, tag, commit)")
	}

	argss := [][]string{
		{"init"},
		{"remote", "add", "origin", t.opts.URL},
		{"fetch", "origin"}, // TODO shallow clones?
		{"checkout", t.opts.Ref},
		{"submodule", "update", "--init", "--recursive"},
	}

	for _, args := range argss {
		_, _, err := t.run(args, dstPath)
		if err != nil {
			return GitInfo{}, err
		}
	}

	info := GitInfo{}

	out, _, err := t.run([]string{"rev-parse", "HEAD"}, dstPath)
	if err != nil {
		return GitInfo{}, err
	}

	info.SHA = strings.TrimSpace(out)

	out, _, err = t.run([]string{"log", "-n", "1", "--pretty=%B", info.SHA}, dstPath)
	if err != nil {
		return GitInfo{}, err
	}

	info.CommitTitle = strings.TrimSpace(out)

	return info, nil
}

func (t *Git) run(args []string, dstPath string) (string, string, error) {
	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command("git", args...)
	cmd.Dir = dstPath
	cmd.Stdout = io.MultiWriter(t.infoLog, &stdoutBs)
	cmd.Stderr = io.MultiWriter(t.infoLog, &stderrBs)

	t.infoLog.Write([]byte(fmt.Sprintf("--> git %s\n", strings.Join(args, " "))))

	err := cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("Git %s: %s (stderr: %s)", args, err, stderrBs.String())
	}

	return stdoutBs.String(), stderrBs.String(), nil
}
