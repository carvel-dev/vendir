// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	"gopkg.in/yaml.v2"
)

func TestInvalidSymlink(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	tmpDir, err := os.MkdirTemp("", "vendir-test")
	if err != nil {
		t.Fatalf("creating tmpdir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	symlinkDir := filepath.Join(tmpDir, "symlink-dir")
	err = os.Mkdir(symlinkDir, os.ModePerm)
	if err != nil {
		t.Fatalf("creating symlink dir: %v", err)

	}

	// valid since it is in the symlink-dir
	validFilePath := filepath.Join(symlinkDir, "a_valid_file.txt")
	validFile, err := os.Create(validFilePath)
	if err != nil {
		t.Fatalf("creating file: %v", err)
	}
	validFile.Close()

	//invalid since it is outside the symlink-dir
	invalidFilePath := filepath.Join(tmpDir, "invalid_file.txt")
	invalidFile, err := os.Create(invalidFilePath)
	if err != nil {
		t.Fatalf("creating file: %v", err)
	}
	invalidFile.Close()

	config := config.Config{
		APIVersion: "vendir.k14s.io/v1alpha1",
		Kind:       "Config",
		Directories: []config.Directory{{
			Path: "result",
			Contents: []config.DirectoryContents{{
				Path: "bad",
				Directory: &config.DirectoryContentsDirectory{
					Path: "symlink-dir",
				},
			}},
		}},
	}
	vendirYML, err := os.Create(filepath.Join(tmpDir, "vendir.yml"))
	if err != nil {
		t.Fatalf("creating vendir.yml: %v", err)
	}
	defer vendirYML.Close()

	err = yaml.NewEncoder(vendirYML).Encode(&config)
	if err != nil {
		t.Fatalf("writing vendir.yml: %v", err)
	}

	tests := []struct {
		symlinkLocation string
		valid           bool
		expectedErr     string
	}{
		{symlinkLocation: "a_valid_file.txt", valid: true},
		{symlinkLocation: invalidFilePath, valid: false, expectedErr: "Invalid symlink found to outside parent directory"},
		{symlinkLocation: "non_existent_file.txt", valid: false, expectedErr: "Unable to resolve symlink"},
	}
	for _, tc := range tests {
		symlinkPath := filepath.Join(symlinkDir, "file")
		err = os.Symlink(tc.symlinkLocation, symlinkPath)
		if err != nil {
			t.Fatalf("creating symlink: %v", err)
		}

		_, err = vendir.RunWithOpts([]string{"sync"}, RunOpts{Dir: tmpDir, AllowError: true})
		if tc.valid && err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !tc.valid {
			if err == nil {
				t.Fatalf("expected an err, got none")
			}
			if !strings.Contains(err.Error(), tc.expectedErr) {
				t.Fatalf("Expected invalid symlink err: %s", err)
			}
		}

		err = os.Remove(symlinkPath)
		if err != nil {
			t.Fatalf("deleting symlink: %v", err)
		}
	}
}
