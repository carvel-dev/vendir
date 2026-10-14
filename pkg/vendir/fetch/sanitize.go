// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package fetch

import (
	"net/url"
	"strings"
)

// SensitiveValues holds credential values that should be redacted from logs
type SensitiveValues struct {
	Passwords []string
	Tokens    []string
	BasicAuth string // base64-encoded user:pass
	URLs      []*url.URL
}

// RedactSensitiveData replaces known secret values with [REDACTED].
// Unlike regex patterns, this approach:
// - Only redacts known values (no false positives)
// - Handles values that appear in URLs via url.Parse
// - Preserves diagnostic messages (doesn't mangle "Username for", etc.)
// - Survives values split across multiple writes (if called on the full string)
func RedactSensitiveData(input string, secrets *SensitiveValues) string {
	if secrets == nil {
		return input
	}

	result := input

	// Redact known password values (skip empty to avoid mangling output)
	for _, val := range secrets.Passwords {
		if val != "" {
			result = strings.ReplaceAll(result, val, "[REDACTED]")
		}
	}

	// Redact known token values
	for _, val := range secrets.Tokens {
		if val != "" {
			result = strings.ReplaceAll(result, val, "[REDACTED]")
		}
	}

	// Redact base64-encoded Basic auth
	if secrets.BasicAuth != "" {
		result = strings.ReplaceAll(result, secrets.BasicAuth, "[REDACTED]")
	}

	// Redact passwords from parsed URLs (e.g., https://user:pass@host/path)
	for _, u := range secrets.URLs {
		if u == nil || u.User == nil {
			continue
		}

		if pass, ok := u.User.Password(); ok && pass != "" {
			result = strings.ReplaceAll(result, pass, "[REDACTED]")
		}

		// Don't redact bare usernames to preserve diagnostic messages
		// (e.g., "could not read Username for 'https://github.com'")
	}

	return result
}
