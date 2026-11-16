// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package diagnostics

import "regexp"

var urlInText = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s"'<>]+`)

// Text sanitizes URLs printed by child processes.
// The actual output returned to their consumer remains intact.
func Text(s string) string { return urlInText.ReplaceAllStringFunc(s, URL) }
