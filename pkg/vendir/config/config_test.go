// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
)

func TestEmptyLineStartsConfig(t *testing.T) {
	t.Run("Empty document is ignored", func(t *testing.T) {
		tempConfigPath := filepath.Join(t.TempDir(), "config.yml")
		configWithWhitespace := []byte(`

---
apiVersion: vendir.k14s.io/v1alpha1
kind: Config`)

		require.NoError(t, os.WriteFile(tempConfigPath, configWithWhitespace, 0666))

		_, _, _, err := config.NewConfigFromFiles([]string{tempConfigPath})
		require.NoError(t, err)
	})
}
