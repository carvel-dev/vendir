// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"io/ioutil"
	"testing"
)

func TestUseDirectory(t *testing.T) {
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

	checkFileContent := func(expectedContent string) {
		file, err := ioutil.ReadFile(path + "/vendor/local-dir/file.txt")
		if err != nil {
			t.Fatalf("Expected no err")
		}
		if string(file) != expectedContent {
			t.Fatalf("Expected file contents to be known: %s", string(file))
		}
	}

	checkFileContent("file\n")

	_, err := vendir.RunWithOpts([]string{"sync"}, RunOpts{Dir: path})
	if err != nil {
		t.Fatalf("Expected no err")
	}

	checkFileContent("file\n")

	_, err = vendir.RunWithOpts([]string{"sync", "--directory", "vendor/local-dir=local-dir-dev"}, RunOpts{Dir: path})
	if err != nil {
		t.Fatalf("Expected no err")
	}

	checkFileContent("local-dir2/file\n")
}
