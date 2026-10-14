// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package fetch

import (
	"net/url"
	"testing"
)

func TestRedactSensitiveDataWithKnownValues(t *testing.T) {
	password := "my-secret-password"
	token := "my-bearer-token"
	basicAuth := "dXNlcjpwYXNz" // base64 of user:pass

	secrets := &SensitiveValues{
		Passwords: []string{password},
		Tokens:    []string{token},
		BasicAuth: basicAuth,
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "redacts known password",
			input:    "Failed to authenticate with password: " + password,
			expected: "Failed to authenticate with password: [REDACTED]",
		},
		{
			name:     "redacts known token",
			input:    "Authorization: Bearer " + token,
			expected: "Authorization: Bearer [REDACTED]",
		},
		{
			name:     "redacts basic auth",
			input:    "Authorization: Basic " + basicAuth,
			expected: "Authorization: Basic [REDACTED]",
		},
		{
			name:     "redacts multiple values",
			input:    "Password: " + password + " Token: " + token,
			expected: "Password: [REDACTED] Token: [REDACTED]",
		},
		{
			name:     "preserves non-sensitive text",
			input:    "Username for 'https://github.com' is invalid",
			expected: "Username for 'https://github.com' is invalid",
		},
		{
			name:     "skips empty values",
			input:    "Some text",
			expected: "Some text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactSensitiveData(tt.input, secrets)
			if result != tt.expected {
				t.Errorf("RedactSensitiveData() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRedactSensitiveDataWithURLUserinfo(t *testing.T) {
	gitURL, _ := url.Parse("https://user:my-secret@github.com/repo.git")
	secrets := &SensitiveValues{
		URLs: []*url.URL{gitURL},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "redacts password from URL userinfo",
			input:    "Cloning from https://user:my-secret@github.com/repo.git",
			expected: "Cloning from https://user:[REDACTED]@github.com/repo.git",
		},
		{
			name:     "preserves URL structure",
			input:    "git remote add origin https://user:my-secret@github.com/repo.git",
			expected: "git remote add origin https://user:[REDACTED]@github.com/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactSensitiveData(tt.input, secrets)
			if result != tt.expected {
				t.Errorf("RedactSensitiveData() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRedactSensitiveDataWithNilSecrets(t *testing.T) {
	input := "some error message with password=secret"
	result := RedactSensitiveData(input, nil)

	// Should return unchanged when secrets are nil
	if result != input {
		t.Errorf("RedactSensitiveData() with nil secrets = %q, want %q", result, input)
	}
}

func TestRedactSensitiveDataSkipsEmptyValues(t *testing.T) {
	secrets := &SensitiveValues{
		Passwords: []string{"", "secret"},
		Tokens:    []string{},
	}

	input := "Password: secret"
	expected := "Password: [REDACTED]"

	result := RedactSensitiveData(input, secrets)
	if result != expected {
		t.Errorf("RedactSensitiveData() = %q, want %q", result, expected)
	}
}
