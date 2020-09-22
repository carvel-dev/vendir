package e2e

import (
	"testing"
)

func TestExamplesDir(t *testing.T) {
	env := BuildEnv(t)

	tests := []example{
		{Name: "http"},
		{Name: "image"},
		{Name: "helm-chart", Env: []string{"VENDIR_HELM_BINARY=" + env.Helm2Binary}},
		{Name: "helm-chart", Env: []string{"VENDIR_HELM_BINARY=" + env.Helm3Binary}},
		{Name: "github-release"},
		{Name: "entire-dir"},
		{Name: "inline"},
		{Name: "locked", OnlyLocked: true, Env: []string{"VENDIR_HELM_BINARY=" + env.Helm3Binary}},
	}

	for _, test := range tests {
		test.Check(t)
	}
}
