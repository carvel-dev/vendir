package e2e

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestExample(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	// remove some directory
	err := os.RemoveAll("../../examples/git-and-manual/vendor/github.com/cloudfoundry/cf-k8s-networking")
	if err != nil {
		t.Fatalf("Expected no err")
	}

	err = os.MkdirAll("../../examples/git-and-manual/vendor/github.com/cloudfoundry/extra", 0700)
	if err != nil {
		t.Fatalf("Expected no err")
	}

	// add file that shouldnt exist
	err = ioutil.WriteFile("../../examples/git-and-manual/vendor/github.com/cloudfoundry/extra/extra", []byte("extra"), 0700)
	if err != nil {
		t.Fatalf("Expected no err")
	}

	gitOut := gitDiffExamplesDir(t)
	if gitOut == "" {
		t.Fatalf("Expected diff, but was: >>>%s<<<", gitOut)
	}
	if !strings.Contains(gitOut, "LICENSE") {
		t.Fatalf("Expected license file to be deleted, but was: >>>%s<<<", gitOut)
	}
	if !strings.Contains(gitOut, "extra") {
		t.Fatalf("Expected extra file to be added, but was: >>>%s<<<", gitOut)
	}

	_, err = vendir.RunWithOpts([]string{"sync"}, RunOpts{Dir: "../../examples/git-and-manual"})
	if err != nil {
		t.Fatalf("Expected no err")
	}

	gitOut = gitDiffExamplesDir(t)
	if gitOut != "" {
		t.Fatalf("Expected no diff, but was: >>>%s<<<", gitOut)
	}
}

func gitDiffExamplesDir(t *testing.T) string {
	_, _, err := execGit([]string{"add", "--all", "--", "examples/git-and-manual/"}, "../../")
	if err != nil {
		t.Fatalf("Expected no err")
	}

	diffOut, _, err := execGit([]string{"diff", "--cached", "--", "examples/git-and-manual/"}, "../../")
	if err != nil {
		t.Fatalf("Expected no err")
	}

	return diffOut
}

func execGit(args []string, dir string) (string, string, error) {
	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = &stdoutBs
	cmd.Stderr = &stderrBs

	err := cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("Git %s: %s (stderr: %s)", args, err, stderrBs.String())
	}

	return stdoutBs.String(), stderrBs.String(), nil
}
