// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverlappingPathsErr(t *testing.T) {
	tests := []struct {
		name        string
		description string
		yaml        string
		expectedErr string
	}{
		{
			name:        "contents-paths",
			description: "syncing config with overlapping contents paths",
			yaml: `
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: vendor
  contents:
  - path: foo/bar/baz
    inline:
      paths:
        test.txt: love
  - path: foo
    inline:
      paths:
        test.txt: peace
`,
			expectedErr: `vendir: Error: Parsing resource config '-':
  Unmarshaling config:
    Validating config:
      Expected to not manage overlapping paths: 'vendor/foo/bar/baz' and 'vendor/foo'
`,
		},
		{
			name:        "directories-paths",
			description: "syncing config with overlapping directories paths",
			yaml: `
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: vendor
  contents:
  - path: foo/bar/baz
    inline:
      paths:
        test.txt: love
- path: vendor/foo
  contents:
  - path: bar
    inline:
      paths:
        test.txt: peace
`,
			expectedErr: `vendir: Error: Parsing resource config '-':
  Unmarshaling config:
    Validating config:
      Expected to not manage overlapping paths: 'vendor/foo' and 'vendor'
`,
		},
		{
			name:        "same-directories-paths",
			description: "syncing config with the same directories paths",
			yaml: `
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: vendor
  contents:
  - path: foo
    inline:
      paths:
        test.txt: love
- path: vendor
  contents:
  - path: bar
    inline:
      paths:
        test.txt: peace
`,
			expectedErr: `vendir: Error: Parsing resource config '-':
  Unmarshaling config:
    Validating config:
      Expected to not manage overlapping paths: 'vendor' and 'vendor'
`,
		},
	}

	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := vendir.RunWithOpts(
				[]string{"sync", "-f", "-"},
				RunOpts{
					Dir:         t.TempDir(),
					StdinReader: strings.NewReader(test.yaml),
					AllowError:  true,
				},
			)
			require.Error(t, err, "Expected to err while %s", test.description)
			assert.ErrorContains(t, err, test.expectedErr)
		})
	}
}
