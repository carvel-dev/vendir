// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	env := BuildEnv(t)
	out := Vendir{t, env.BinaryPath, Logger{}}.Run([]string{"version"})

	if !strings.Contains(out, "vendir version") {
		t.Fatalf("Expected to find client version")
	}
}
