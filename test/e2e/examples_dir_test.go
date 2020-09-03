package e2e

import (
	"fmt"
	"os"
	"testing"
)

type exampleTest struct {
	Name       string
	OnlyLocked bool
}

func TestExamplesDir(t *testing.T) {
	tests := []exampleTest{
		{Name: "http"},
		{Name: "image"},
		{Name: "helm-chart"},
		{Name: "github-release"},
		{Name: "entire-dir"},
		{Name: "locked", OnlyLocked: true},
	}
	for _, test := range tests {
		test.Check(t)
	}
}

func (et exampleTest) Check(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	vendir := Vendir{t, env.BinaryPath, logger}

	logger.Section(et.Name, func() {
		err := et.check(t, vendir)
		if err != nil {
			t.Fatalf("[example: %s] %s", et.Name, err)
		}
	})
}

func (et exampleTest) check(t *testing.T, vendir Vendir) error {
	dir := "examples/" + et.Name
	path := "../../" + dir

	vendorPath := path + "/vendor"

	vendorDir, err := os.Stat(vendorPath)
	if err != nil {
		return fmt.Errorf("Expected no err")
	}
	if !vendorDir.IsDir() {
		return fmt.Errorf("Expected to be dir")
	}

	// remove all vendored bits
	err = os.RemoveAll(vendorPath)
	if err != nil {
		return fmt.Errorf("Expected no err")
	}

	if !et.OnlyLocked {
		_, err = vendir.RunWithOpts([]string{"sync"}, RunOpts{Dir: path})
		if err != nil {
			return fmt.Errorf("Expected no err")
		}

		// This assumes that example's vendor directory is committed to git
		gitOut := gitDiffExamplesDir(t, dir)
		if gitOut != "" {
			return fmt.Errorf("Expected no diff, but was: >>>%s<<<", gitOut)
		}
	}

	_, err = vendir.RunWithOpts([]string{"sync", "--locked"}, RunOpts{Dir: path})
	if err != nil {
		return fmt.Errorf("Expected no err")
	}

	gitOut := gitDiffExamplesDir(t, dir)
	if gitOut != "" {
		return fmt.Errorf("Expected no diff, but was: >>>%s<<<", gitOut)
	}

	return nil
}
