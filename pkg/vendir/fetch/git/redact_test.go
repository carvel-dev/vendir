// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactGitArgs(t *testing.T) {
	args := []string{"config", "--add", "http.extraHeader", "Authorization: Basic dXNlcjpwYXNz"}
	got := redactGitArgs(args)

	require.Equal(t, []string{"config", "--add", "http.extraHeader", "Authorization: Basic [REDACTED]"}, got)
	require.Equal(t, "Authorization: Basic dXNlcjpwYXNz", args[3], "original args must be untouched")
}
