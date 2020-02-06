package e2e

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestExampleEntireDir(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	dir := "examples/entire-dir"
	path := "../../" + dir

	// remove some directory
	err := os.RemoveAll(path + "/vendor/cfroutesync")
	if err != nil {
		t.Fatalf("Expected no err")
	}

	err = os.MkdirAll(path+"/vendor/extra", 0700)
	if err != nil {
		t.Fatalf("Expected no err")
	}

	// add file that shouldnt exist
	err = ioutil.WriteFile(path+"/vendor/extra/extra", []byte("extra"), 0700)
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

	_, err = vendir.RunWithOpts([]string{"sync"}, RunOpts{Dir: path})
	if err != nil {
		t.Fatalf("Expected no err")
	}

	gitOut = gitDiffExamplesDir(t, dir)
	if gitOut != "" {
		t.Fatalf("Expected no diff, but was: >>>%s<<<", gitOut)
	}
}
