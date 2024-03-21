// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

type Vendir struct {
	t          *testing.T
	binaryPath string
	l          Logger
}

type RunOpts struct {
	AllowError   bool
	StderrWriter io.Writer
	StdoutWriter io.Writer
	StdinReader  io.Reader
	CancelCh     chan struct{}
	Redact       bool
	Dir          string
	Env          []string
}

func (k Vendir) Run(args []string) string {
	out, _ := k.RunWithOpts(args, RunOpts{})
	return out
}

func (k Vendir) RunWithOpts(args []string, opts RunOpts) (string, error) {
	k.l.Debugf("Running '%s'...\n", k.cmdDesc(args, opts))

	cmd := exec.Command(k.binaryPath, args...)
	cmd.Env = append(os.Environ(), opts.Env...)
	cmd.Stdin = opts.StdinReader

	if len(opts.Dir) > 0 {
		cmd.Dir = opts.Dir
	}

	var stderr, stdout bytes.Buffer

	if opts.StderrWriter != nil {
		cmd.Stderr = opts.StderrWriter
	} else {
		cmd.Stderr = &stderr
	}

	if opts.StdoutWriter != nil {
		cmd.Stdout = opts.StdoutWriter
	} else {
		cmd.Stdout = &stdout
	}

	if opts.CancelCh != nil {
		go func() {
			select {
			case <-opts.CancelCh:
				cmd.Process.Signal(os.Interrupt)
			}
		}()
	}

	err := cmd.Run()
	stdoutStr := stdout.String()

	if err != nil {
		err = fmt.Errorf("Execution error: stdout: '%s' stderr: '%s' error: '%s'", stdoutStr, stderr.String(), err)

		if !opts.AllowError {
			k.t.Fatalf("Failed to successfully execute '%s': %v", k.cmdDesc(args, opts), err)
		}
	}

	return stdoutStr, err
}

func (k Vendir) cmdDesc(args []string, opts RunOpts) string {
	prefix := "vendir"
	if opts.Redact {
		return prefix + " -redacted-"
	}
	return fmt.Sprintf("%s %s", prefix, strings.Join(args, " "))
}
