package e2e

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestExampleLocked(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	osEnv := []string{"VENDIR_HELM_BINARY="+env.Helm2Binary}

	dir := "examples/locked"
	path := "../../" + dir

	reset := func() {
		// Make sure state is reset
		_, err := vendir.RunWithOpts([]string{"sync", "--locked"}, RunOpts{Dir: path, Env: osEnv})
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
	err = ioutil.WriteFile(path+"/vendor/github.com/cloudfoundry/extra/extra", []byte("extra"), 0700)
	if err != nil {
		t.Fatalf("Expected no err")
	}

	gitOut := gitDiffExamplesDir(t, dir)
	if gitOut == "" {
		t.Fatalf("Expected diff, but was: >>>%s<<<", gitOut)
	}
	if !strings.Contains(gitOut, "extra") {
		t.Fatalf("Expected extra file to be added, but was: >>>%s<<<", gitOut)
	}

	_, err = vendir.RunWithOpts([]string{"sync", "--locked"}, RunOpts{Dir: path, Env: osEnv})
	if err != nil {
		t.Fatalf("Expected no err")
	}

	gitOut = gitDiffExamplesDir(t, dir)
	if gitOut != "" {
		t.Fatalf("Expected no diff, but was: >>>%s<<<", gitOut)
	}
}
