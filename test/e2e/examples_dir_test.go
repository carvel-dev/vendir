// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"
)

func TestExamplesDir(t *testing.T) {
	env := BuildEnv(t)

	// Useful when commenting out examples
	_ = env.Helm3Binary

	tests := []example{
		{Name: "git"},
		{Name: "hg"},
		{Name: "http"},
		{Name: "image"},
		{Name: "imgpkgBundle"},
		{Name: "helm-chart", Env: []string{"VENDIR_HELM_BINARY=" + env.Helm2Binary}},
		{Name: "helm-chart", Env: []string{"VENDIR_HELM_BINARY=" + env.Helm3Binary}},
		{Name: "github-release"},
		{Name: "entire-dir"},
		{Name: "inline"},
		{Name: "locked", OnlyLocked: true, Env: []string{"VENDIR_HELM_BINARY=" + env.Helm3Binary}},
		{Name: "new-root-path"},
		{Name: "versionselection"},
		{Name: "ignore", SkipRemove: true},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test.Check(t)
		})
	}
}
