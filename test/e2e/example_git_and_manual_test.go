// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"strings"
	"testing"
)

func TestExampleGitAndManual(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	dir := "examples/git-and-manual"
	path := "../../" + dir

	reset := func() {
		// Make sure state is reset
		_, err := vendir.RunWithOpts([]string{"sync"}, RunOpts{Dir: path})
		if err != nil {
			t.Fatalf("Expected no err")
		}
	}

	reset()
	defer reset()

	// remove some directory
	err := os.RemoveAll(path + "/vendor/github.com/cloudfoundry/cf-k8s-networking")
	if err != nil {
		t.Fatalf("Expected no err")
	}

	err = os.MkdirAll(path+"/vendor/github.com/cloudfoundry/extra", 0700)
	if err != nil {
		t.Fatalf("Expected no err")
	}

	// add file that shouldnt exist
	err = os.WriteFile(path+"/vendor/github.com/cloudfoundry/extra/extra", []byte("extra"), 0600)
	if err != nil {
		t.Fatalf("Expected no err")
	}

	gitOut := gitDiffExamplesDir(t, dir, "../../")
	if gitOut == "" {
		t.Fatalf("Expected diff, but was: >>>%s<<<", gitOut)
	}
	if !strings.Contains(gitOut, "extra") {
		t.Fatalf("Expected extra file to be added, but was: >>>%s<<<", gitOut)
	}

	_, err = vendir.RunWithOpts([]string{"sync"}, RunOpts{Dir: path})
	if err != nil {
		t.Fatalf("Expected no err")
	}

	gitOut = gitDiffExamplesDir(t, dir, "../../")
	if gitOut != "" {
		t.Fatalf("Expected no diff, but was: >>>%s<<<", gitOut)
	}
}
