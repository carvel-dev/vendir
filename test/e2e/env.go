package e2e

import (
	"os"
	"strings"
	"testing"
)

type Env struct {
	BinaryPath  string
	Helm2Binary string
	Helm3Binary string
}

func BuildEnv(t *testing.T) Env {
	env := Env{
		BinaryPath: os.Getenv("VENDIR_BINARY_PATH"),

		// Allowed to be empty
		Helm2Binary: os.Getenv("VENDIR_E2E_HELM2_BINARY"),
		Helm3Binary: os.Getenv("VENDIR_E2E_HELM3_BINARY"),
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
