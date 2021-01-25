// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strings"
	"testing"
)

func TestMinVersionErr(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	path := "../../examples/min-version"

	_, err := vendir.RunWithOpts([]string{"sync"}, RunOpts{Dir: path, AllowError: true})
	if err == nil {
		t.Fatalf("Expected err")
	}
	if !strings.Contains(err.Error(), "does not meet the minimum required version '100.0.0'") {
		t.Fatalf("Expected overlapping err: %s", err)
	}
}
