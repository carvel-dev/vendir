package e2e

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHttpTarGz(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	vendir := Vendir{t, env.BinaryPath, logger}
	dstPath, err := os.MkdirTemp("", "vendir-e2e-http-targz-dst")
	require.NoError(t, err)
	// defer os.RemoveAll(dstPath)

	yaml := `
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: vendor
  contents:
  - path: github.com/carvel-dev/vendir
    http:
      url: https://github.com/carvel-dev/vendir/archive/refs/tags/v0.34.4.tar.gz
  `

	logger.Section("sync tar.gz made with git archive", func() {
		_, err := vendir.RunWithOpts([]string{"sync", "-f", "-"}, RunOpts{Dir: dstPath, StdinReader: strings.NewReader(yaml), AllowError: true})
		require.NoError(t, err)
		_, err = os.Stat(fmt.Sprintf("%s/%s", dstPath, "vendor/github.com/carvel-dev/vendir/v0.34.4.tar.gz"))
		require.Error(t, err)
		_, err = os.Stat(fmt.Sprintf("%s/%s", dstPath, "vendor/github.com/carvel-dev/vendir/vendir-0.34.4/cmd/vendir/vendir.go"))
		require.NoError(t, err)
	})
}
