// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package directory

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testFilePerms  = os.FileMode(0600)
	testDirPerms   = os.FileMode(0700)
	testGroupPerms = os.FileMode(0640)
	testOtherPerms = os.FileMode(0750)

	unchangedContents = "unchanged contents"
	addedContents     = "added contents"
)

func TestReconcileDirLeavesUnchangedEntriesUntouched(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()

	kept, nested := "kept.yml", filepath.Join("nested", "kept.yml")

	writeFile(t, dst, kept, unchangedContents)
	writeFile(t, dst, nested, unchangedContents)
	writeFile(t, src, kept, unchangedContents)
	writeFile(t, src, nested, unchangedContents)

	before := modTimes(t, dst)

	require.NoError(t, reconcileDir(src, dst, nil))

	require.Equal(t, before, modTimes(t, dst))
}

func TestReconcileDirRewritesChangedFiles(t *testing.T) {
	changed := "changed.yml"

	t.Run("contents of a different length", func(t *testing.T) {
		src, dst := t.TempDir(), t.TempDir()

		writeFile(t, dst, changed, "old")
		writeFile(t, src, changed, "new contents")

		before := modTimes(t, dst)

		require.NoError(t, reconcileDir(src, dst, nil))

		require.Equal(t, "new contents", readFile(t, dst, changed))
		require.NotEqual(t, before, modTimes(t, dst))
	})

	t.Run("contents of the same length", func(t *testing.T) {
		src, dst := t.TempDir(), t.TempDir()

		writeFile(t, dst, changed, "abc")
		writeFile(t, src, changed, "abd")

		require.NoError(t, reconcileDir(src, dst, nil))

		require.Equal(t, "abd", readFile(t, dst, changed))
	})
}

func TestReconcileDirAddsAndDeletesEntries(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()

	staleFile, staleDir := "stale.yml", "stale-dir"
	added := filepath.Join("new-dir", "new.yml")

	writeFile(t, dst, filepath.Join(staleDir, staleFile), "stale")
	writeFile(t, dst, staleFile, "stale")
	writeFile(t, src, added, addedContents)

	require.NoError(t, reconcileDir(src, dst, nil))

	require.NoFileExists(t, filepath.Join(dst, staleFile))
	require.NoDirExists(t, filepath.Join(dst, staleDir))
	require.Equal(t, addedContents, readFile(t, dst, added))
}

func TestReconcileDirKeepsPreservedPaths(t *testing.T) {
	t.Run("at the top level", func(t *testing.T) {
		src, dst := t.TempDir(), t.TempDir()

		unmanaged := "unmanaged"

		writeFile(t, dst, filepath.Join(unmanaged, "mine.yml"), unchangedContents)
		writeFile(t, src, "staged.yml", addedContents)

		before := modTimes(t, filepath.Join(dst, unmanaged))

		preserved := newPreservedPaths([]string{unmanaged})
		require.NoError(t, reconcileDir(src, dst, preserved))

		require.Equal(t, before, modTimes(t, filepath.Join(dst, unmanaged)))
		require.Equal(t, addedContents, readFile(t, dst, "staged.yml"))
	})

	t.Run("nested next to staged contents", func(t *testing.T) {
		src, dst := t.TempDir(), t.TempDir()

		unmanaged := filepath.Join("group", "unmanaged")
		staged := filepath.Join("group", "staged")

		writeFile(t, dst, filepath.Join(unmanaged, "mine.yml"), unchangedContents)
		writeFile(t, dst, filepath.Join(staged, "stale.yml"), "stale")
		writeFile(t, src, filepath.Join(staged, "fresh.yml"), addedContents)

		before := modTimes(t, filepath.Join(dst, unmanaged))

		preserved := newPreservedPaths([]string{unmanaged})
		require.NoError(t, reconcileDir(src, dst, preserved))

		require.Equal(t, before, modTimes(t, filepath.Join(dst, unmanaged)))
		require.NoFileExists(t, filepath.Join(dst, staged, "stale.yml"))
		require.Equal(t, addedContents,
			readFile(t, dst, filepath.Join(staged, "fresh.yml")))
	})

	t.Run("below a dir that holds nothing staged", func(t *testing.T) {
		src, dst := t.TempDir(), t.TempDir()

		unmanaged := filepath.Join("group", "unmanaged")

		writeFile(t, dst, filepath.Join(unmanaged, "mine.yml"), unchangedContents)
		writeFile(t, dst, filepath.Join("group", "stale.yml"), "stale")
		require.NoError(t, os.Mkdir(filepath.Join(src, "group"), testDirPerms))

		before := modTimes(t, filepath.Join(dst, unmanaged))

		preserved := newPreservedPaths([]string{unmanaged})
		require.NoError(t, reconcileDir(src, dst, preserved))

		require.Equal(t, before, modTimes(t, filepath.Join(dst, unmanaged)))
		require.NoFileExists(t, filepath.Join(dst, "group", "stale.yml"))
	})
}

func TestReconcileDirReplacesEntriesWhoseTypeChanged(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()

	becameDir, becameFile := "became-dir", "became-file"
	nested := filepath.Join(becameDir, "nested.yml")

	writeFile(t, dst, becameDir, "leftover file")
	writeFile(t, src, nested, "nested contents")
	writeFile(t, dst, filepath.Join(becameFile, "nested.yml"), "leftover dir")
	writeFile(t, src, becameFile, "file contents")

	require.NoError(t, reconcileDir(src, dst, nil))

	require.Equal(t, "nested contents", readFile(t, dst, nested))
	require.Equal(t, "file contents", readFile(t, dst, becameFile))
}

func TestReconcileDirDoesNotWriteThroughLeftOverSymlinks(t *testing.T) {
	src, dst, outside := t.TempDir(), t.TempDir(), t.TempDir()

	planted := filepath.Join("escape", "planted.yml")

	require.NoError(t, os.Symlink(outside, filepath.Join(dst, "escape")))
	writeFile(t, src, planted, "planted")

	require.NoError(t, reconcileDir(src, dst, nil))

	require.Equal(t, "planted", readFile(t, dst, planted))
	require.NoFileExists(t, filepath.Join(outside, "planted.yml"))
}

func TestReconcileDirWithSymlinks(t *testing.T) {
	link := "link"

	t.Run("same target is left untouched", func(t *testing.T) {
		src, dst := t.TempDir(), t.TempDir()

		target := "target.yml"

		writeFile(t, src, target, "target")
		writeFile(t, dst, target, "target")
		require.NoError(t, os.Symlink(target, filepath.Join(src, link)))
		require.NoError(t, os.Symlink(target, filepath.Join(dst, link)))

		before := modTimes(t, dst)

		require.NoError(t, reconcileDir(src, dst, nil))

		require.Equal(t, before, modTimes(t, dst))
	})

	t.Run("different target is updated", func(t *testing.T) {
		src, dst := t.TempDir(), t.TempDir()

		staged := "staged-target.yml"

		require.NoError(t, os.Symlink("old-target.yml", filepath.Join(dst, link)))
		require.NoError(t, os.Symlink(staged, filepath.Join(src, link)))

		require.NoError(t, reconcileDir(src, dst, nil))

		target, err := os.Readlink(filepath.Join(dst, link))
		require.NoError(t, err)
		require.Equal(t, staged, target)
	})
}

func TestReconcileDirAlignsPermissionsWithoutRewriting(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()

	unchanged := "unchanged.yml"

	writeFile(t, src, unchanged, unchangedContents)
	writeFile(t, dst, unchanged, unchangedContents)
	require.NoError(t, os.Chmod(filepath.Join(src, unchanged), testGroupPerms))

	before := modTimes(t, dst)

	require.NoError(t, reconcileDir(src, dst, nil))

	require.Equal(t, before, modTimes(t, dst))

	info, err := os.Stat(filepath.Join(dst, unchanged))
	require.NoError(t, err)
	require.Equal(t, testGroupPerms, info.Mode().Perm())
}

func TestReconcileDirCreatesMissingFinalLocation(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "missing")

	added := "added.yml"

	writeFile(t, src, added, addedContents)
	require.NoError(t, os.Chmod(src, testOtherPerms))

	require.NoError(t, reconcileDir(src, dst, nil))

	require.Equal(t, addedContents, readFile(t, dst, added))

	info, err := os.Stat(dst)
	require.NoError(t, err)
	require.Equal(t, testOtherPerms, info.Mode().Perm())
}

func TestReconcileDirKeepsStagedModTimeOfMovedEntries(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()

	moved := "moved.yml"

	writeFile(t, src, moved, "moved")
	staged := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
	require.NoError(t, os.Chtimes(filepath.Join(src, moved), staged, staged))

	require.NoError(t, reconcileDir(src, dst, nil))

	info, err := os.Stat(filepath.Join(dst, moved))
	require.NoError(t, err)
	require.True(t, info.ModTime().Equal(staged),
		"expected mod time %s, got %s", staged, info.ModTime())
}

func writeFile(t *testing.T, dir, path, contents string) {
	t.Helper()

	fullPath := filepath.Join(dir, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), testDirPerms))
	require.NoError(t, os.WriteFile(fullPath, []byte(contents), testFilePerms))
}

func readFile(t *testing.T, dir, path string) string {
	t.Helper()

	bs, err := os.ReadFile(filepath.Join(dir, path))
	require.NoError(t, err)

	return string(bs)
}

// modTimes collects the modification time of every entry below dir, keyed by
// its path relative to dir
func modTimes(t *testing.T, dir string) map[string]time.Time {
	t.Helper()

	times := map[string]time.Time{}

	require.NoError(t, filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err)

		relPath, err := filepath.Rel(dir, path)
		require.NoError(t, err)
		times[relPath] = info.ModTime()

		return nil
	}))

	return times
}
