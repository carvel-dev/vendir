// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package directory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveDanglingSymlinks(t *testing.T) {
	root, err := os.MkdirTemp("", "vendir-test")
	if err != nil {
		t.Fatalf("failed to create tmpdir: %v", err)
	}
	defer os.RemoveAll(root)

	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("failed to read link tmpdir: %v", err)
	}

	// existing file that a valid symlink points to
	existingFile := filepath.Join(root, "existing.txt")
	f, err := os.Create(existingFile)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	f.Close()

	// valid symlink: points to existing file
	validLink := filepath.Join(root, "valid_link")
	if err = os.Symlink(existingFile, validLink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// dangling symlink: points to non-existent file
	danglingLink := filepath.Join(root, "dangling_link")
	if err = os.Symlink(filepath.Join(root, "nonexistent.txt"), danglingLink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	if err = RemoveDanglingSymlinks(root); err != nil {
		t.Fatalf("RemoveDanglingSymlinks() error = %v", err)
	}

	// valid symlink must still exist
	if _, err = os.Lstat(validLink); err != nil {
		t.Errorf("valid symlink was unexpectedly removed: %v", err)
	}
	// dangling symlink must be gone
	if _, err = os.Lstat(danglingLink); err == nil {
		t.Errorf("dangling symlink was not removed")
	}
}

func TestValidateSymlinks(t *testing.T) {
	root, err := os.MkdirTemp("", "vendir-test")
	if err != nil {
		t.Fatalf("failed to create tmpdir: %v", err)
	}
	defer os.RemoveAll(root)

	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("failed to read link tmpdir: %v", err)
	}

	wd := filepath.Join(root, "wd")
	validFilePath := filepath.Join(wd, "file")

	sibling := filepath.Join(root, "wd2")
	siblingFilePath := filepath.Join(sibling, "file")

	for _, path := range []string{wd, sibling} {
		if err = os.Mkdir(path, os.ModePerm); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
	}
	for _, path := range []string{validFilePath, siblingFilePath} {
		file, err := os.Create(path)
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		file.Close()
	}

	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{
			name:    "valid symlink",
			target:  validFilePath,
			wantErr: false,
		},
		{
			name:    "valid symlink to containing directory",
			target:  wd,
			wantErr: false,
		},
		{
			name:    "invalid symlink",
			target:  siblingFilePath,
			wantErr: true,
		},
		{
			name:    "invalid symlink to sibling directory",
			target:  sibling,
			wantErr: true,
		},
		{
			name:    "invalid symlink to parent directory",
			target:  root,
			wantErr: true,
		},
		{
			name:    "invalid symlink to non-existent path",
			target:  filepath.Join(wd, "foo"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		test := func(t *testing.T) {
			newName := filepath.Join(wd, "symlink")
			err := os.Symlink(tt.target, newName)
			if err != nil {
				t.Fatalf("creating symlink: %v", err)
			}
			defer os.Remove(newName)
			if err := ValidateSymlinks(wd); (err != nil) != tt.wantErr {
				t.Errorf("ValidateSymlinks() error = %v, wantErr %v", err, tt.wantErr)
			}
		}

		t.Run(tt.name+" absolute", test)
		t.Run(tt.name+" relative", func(t *testing.T) {
			oldName, err := filepath.Rel(wd, tt.target)
			if err != nil {
				t.Fatalf("relativizing path: %v", err)
			}
			tt.target = oldName
			test(t)
		})
	}
}
