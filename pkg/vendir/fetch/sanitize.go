// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package fetch

import (
	"io"
	"regexp"
)

// sensitivePatterns defines regex patterns for sensitive data redaction
var sensitivePatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{
		name: "password",
		pattern: regexp.MustCompile(
			`(?i)(?:password|passwd)(?:_\d+)?[=:\s]+\S+`),
	},
	{
		name: "username",
		pattern: regexp.MustCompile(
			`(?i)(?:username|user)(?:_\d+)?[=:\s]+\S+`),
	},
	{
		name: "token",
		pattern: regexp.MustCompile(
			`(?i)(?:token|api[_-]?key)(?:_\d+)?[=:\s]+\S+`),
	},
	{
		name: "identity_token",
		pattern: regexp.MustCompile(
			`(?i)(?:identity[_-]?token)(?:_\d+)?[=:\s]+\S+`),
	},
	{
		name: "registry_token",
		pattern: regexp.MustCompile(
			`(?i)(?:registry[_-]?token)(?:_\d+)?[=:\s]+\S+`),
	},
	{
		name:    "bearer",
		pattern: regexp.MustCompile(`(?i)bearer\s+\S+`),
	},
	{
		name:    "auth_header",
		pattern: regexp.MustCompile(`https?://[^:]+:[^@]+@`),
	},
	{
		name:    "ssh_key",
		pattern: regexp.MustCompile(
			`(?i)ssh[_-]?key[=:\s]+\S+`),
	},
}

// RedactSensitiveData replaces sensitive patterns with [REDACTED]
func RedactSensitiveData(input string) string {
	result := input
	for _, p := range sensitivePatterns {
		result = p.pattern.ReplaceAllString(result, "[REDACTED]")
	}
	return result
}

// SanitizingWriter wraps an io.Writer and redacts sensitive data before writing
type SanitizingWriter struct {
	underlying io.Writer
}

// NewSanitizingWriter returns a SanitizingWriter wrapping w
func NewSanitizingWriter(w io.Writer) *SanitizingWriter {
	return &SanitizingWriter{underlying: w}
}

// Write sanitizes the input and writes to the underlying writer
func (w *SanitizingWriter) Write(p []byte) (int, error) {
	sanitized := RedactSensitiveData(string(p))
	_, err := w.underlying.Write([]byte(sanitized))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
