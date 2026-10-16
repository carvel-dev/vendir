// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

// Package githubrelease tests unexported helpers of the GitHub release fetcher
package githubrelease

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIErrorMessage(t *testing.T) {
	t.Run("returns only the message field", func(t *testing.T) {
		body := `{"message":"API rate limit exceeded","token":"ghp_secret","documentation_url":"https://docs"}`
		require.Equal(t, "API rate limit exceeded", apiErrorMessage(strings.NewReader(body)))
	})

	t.Run("does not echo non-JSON bodies", func(t *testing.T) {
		got := apiErrorMessage(strings.NewReader("Authorization: token ghp_secret"))
		require.NotContains(t, got, "ghp_secret")
	})
}
