package e2e

import (
	"os"
	"strings"
	"testing"
)

type Env struct {
	BinaryPath string
}

func BuildEnv(t *testing.T) Env {
	env := Env{
		BinaryPath: os.Getenv("VENDIR_BINARY_PATH"),
	}
	env.Validate(t)
	return env
}

func (e Env) Validate(t *testing.T) {
	errStrs := []string{}

	if len(e.BinaryPath) == 0 {
		errStrs = append(errStrs, "Expected non-empty binary path")
	}

	if len(errStrs) > 0 {
		t.Fatalf("%s", strings.Join(errStrs, "\n"))
	}
}
