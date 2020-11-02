package e2e

import (
	"testing"
)

func TestExamplesDir(t *testing.T) {
	env := BuildEnv(t)

	// Useful when commenting out examples
	_ = env.Helm3Binary

	tests := []example{
		{Name: "http"},
		{Name: "image"},
		{Name: "helm-chart", Env: []string{"VENDIR_HELM_BINARY=" + env.Helm2Binary}},
		{Name: "helm-chart", Env: []string{"VENDIR_HELM_BINARY=" + env.Helm3Binary}},
		{Name: "github-release"},
		{Name: "entire-dir"},
		{Name: "inline"},
		{Name: "locked", OnlyLocked: true, Env: []string{"VENDIR_HELM_BINARY=" + env.Helm3Binary}},
		{Name: "new-root-path"},
		{Name: "versionselection"},
	}

	for _, test := range tests {
		test.Check(t)
	}
}
