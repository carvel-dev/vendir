// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/cppforlife/go-cli-ui/ui"
)

type syncCaptureUI struct {
	ui.UI
	output bytes.Buffer
}

func (u *syncCaptureUI) PrintBlock(b []byte) { u.output.Write(b) }
func TestPartialAndLockedSyncDoNotPrintInlineContent(t *testing.T) {
	t.Chdir(t.TempDir())
	const privateFileMode = 0600
	const config = `apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: output
  contents:
  - path: .
    inline:
      paths:
        credentials.txt: password-canary
`
	if err := os.WriteFile("vendir.yml", []byte(config), privateFileMode); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"full", "partial", "locked"} {
		capture := &syncCaptureUI{UI: ui.NewNoopUI()}
		opts := NewSyncOptions(capture)
		NewSyncCmd(opts)
		if mode == "partial" {
			opts.Directories = []string{"output"}
		}
		if mode == "locked" {
			opts.Locked = true
		}
		if err := opts.Run(); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(capture.output.String(), "password-canary") {
			t.Fatal("inline contents exposed")
		}
		actual, err := os.ReadFile("output/credentials.txt")
		if err != nil || string(actual) != "password-canary" {
			t.Fatalf("synced content changed: %v", err)
		}
	}
}
