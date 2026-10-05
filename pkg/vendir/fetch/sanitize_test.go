// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package fetch

import (
	"bytes"
	"testing"
)

const (
	redacted                  = "[REDACTED]"
	imgpkgRedacted            = "IMGPKG_[REDACTED]"
	imgpkgRegistryRedacted    = "IMGPKG_REGISTRY_[REDACTED]"
	imgpkgIdentityRedacted    = "IMGPKG_REGISTRY_IDENTITY_[REDACTED]"
	imgpkgRegistryTokenRedact = "IMGPKG_REGISTRY_REGISTRY_[REDACTED]"
)

func TestRedactSensitiveDataPasswords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "password pattern",
			input:    "IMGPKG_PASSWORD=mysecret123",
			expected: imgpkgRedacted,
		},
		{
			name:     "password with numbered suffix",
			input:    "IMGPKG_REGISTRY_PASSWORD_0=secret789",
			expected: imgpkgRegistryRedacted,
		},
		{
			name:     "case insensitive password",
			input:    "PASSWORD=mysecret",
			expected: redacted,
		},
		{
			name:     "password with colon separator",
			input:    "password:mysecret",
			expected: redacted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactSensitiveData(tt.input)
			if result != tt.expected {
				t.Errorf("RedactSensitiveData() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRedactSensitiveDataTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "username pattern",
			input:    "IMGPKG_USERNAME=myuser",
			expected: imgpkgRedacted,
		},
		{
			name:     "username with numbered suffix",
			input:    "IMGPKG_REGISTRY_USERNAME_0=user456",
			expected: imgpkgRegistryRedacted,
		},
		{
			name:     "token pattern",
			input:    "IMGPKG_TOKEN=mytoken456",
			expected: imgpkgRedacted,
		},
		{
			name:     "token with numbered suffix",
			input:    "IMGPKG_REGISTRY_TOKEN_0=token789",
			expected: imgpkgRegistryRedacted,
		},
		{
			name:     "identity token pattern",
			input:    "IMGPKG_REGISTRY_IDENTITY_TOKEN_0=idtoken789",
			expected: imgpkgIdentityRedacted,
		},
		{
			name:     "registry token pattern",
			input:    "IMGPKG_REGISTRY_REGISTRY_TOKEN_0=regtoken123",
			expected: imgpkgRegistryTokenRedact,
		},
		{
			name:     "bearer token",
			input:    "Authorization: Bearer token123",
			expected: "Authorization: " + redacted,
		},
		{
			name:     "api-key pattern",
			input:    "api-key=abc123def456",
			expected: redacted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactSensitiveData(tt.input)
			if result != tt.expected {
				t.Errorf("RedactSensitiveData() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRedactSensitiveDataHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic auth header",
			input:    "https://user:password@registry.io/v2/image",
			expected: "https://" + redacted,
		},
		{
			name:     "ssh key",
			input:    "ssh_key=my-private-key-content",
			expected: redacted,
		},
		{
			name:     "multiple credentials in error message",
			input:    "Error: Failed to auth with password=secret123 using username=user456",
			expected: "Error: Failed to auth with " + redacted + " using " + redacted,
		},
		{
			name:     "no sensitive data",
			input:    "This is a normal error message",
			expected: "This is a normal error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactSensitiveData(tt.input)
			if result != tt.expected {
				t.Errorf("RedactSensitiveData() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizingWriter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "redacts password",
			input:    "IMGPKG_PASSWORD=secret123",
			expected: "IMGPKG_[REDACTED]",
		},
		{
			name:     "redacts multiple credentials",
			input:    "password=secret username=user token=tok",
			expected: "[REDACTED] [REDACTED] [REDACTED]",
		},
		{
			name:     "preserves non-sensitive output",
			input:    "Fetching image from registry",
			expected: "Fetching image from registry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			writer := NewSanitizingWriter(buf)

			n, err := writer.Write([]byte(tt.input))
			if err != nil {
				t.Fatalf("Write() error = %v", err)
			}

			// Verify that all input bytes were consumed
			if n != len(tt.input) {
				t.Errorf("Write() returned %d bytes consumed, want %d", n, len(tt.input))
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("SanitizingWriter output = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizingWriterContract(t *testing.T) {
	// Test io.Writer contract: Write must return number of bytes consumed
	input := "IMGPKG_PASSWORD=secretvalue"
	buf := bytes.NewBuffer(nil)
	writer := NewSanitizingWriter(buf)

	n, err := writer.Write([]byte(input))

	// io.Writer contract: n must be <= len(p)
	if n > len(input) {
		t.Errorf("Write() returned %d bytes, but input was only %d bytes (violates io.Writer contract)",
			n, len(input))
	}

	// We consume all input
	if n != len(input) {
		t.Errorf("Write() returned %d bytes consumed, want %d (all input)",
			n, len(input))
	}

	if err != nil {
		t.Errorf("Write() unexpected error = %v", err)
	}

	// Verify output is redacted (may be shorter due to redaction)
	output := buf.String()
	if output != "IMGPKG_[REDACTED]" {
		t.Errorf("SanitizingWriter output = %q, want redacted", output)
	}
}

func TestSanitizingWriterError(t *testing.T) {
	// Test that Write returns 0 bytes on underlying write error
	failingWriter := &failingWriter{}
	writer := NewSanitizingWriter(failingWriter)

	n, err := writer.Write([]byte("test input"))

	if err == nil {
		t.Error("Write() expected error, got nil")
	}

	if n != 0 {
		t.Errorf("Write() returned %d bytes on error, want 0", n)
	}
}

// failingWriter is a test helper that always fails on Write
type failingWriter struct{}

func (*failingWriter) Write(_ []byte) (int, error) {
	return 0, bytes.ErrTooLarge
}
