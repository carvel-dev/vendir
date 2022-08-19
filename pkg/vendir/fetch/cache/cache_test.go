// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/cache"
)

type hitTest struct {
	expectedPath string
	name         string
	cacheID      string
	isPresent    bool
}

func TestHit(t *testing.T) {
	allTests := []hitTest{
		{
			isPresent:    false,
			expectedPath: "",
			name:         "when cache is not populated, it returns false",
			cacheID:      "not-present",
		},
		{
			isPresent:    true,
			expectedPath: "present",
			name:         "when cache exists, it returns true and the path to the folder",
			cacheID:      "present",
		},
		{
			isPresent:    true,
			expectedPath: "some-present",
			name:         "when id contains : converts it to '-' on the folder",
			cacheID:      "some:present",
		},
	}

	for _, test := range allTests {
		t.Run(test.name, func(t *testing.T) {
			cacheFolder, err := os.MkdirTemp("", "vendir-cache-hit-test")
			require.NoError(t, err)
			defer os.RemoveAll(cacheFolder)
			subject := cache.NewCache(cacheFolder)

			if test.isPresent {
				err = os.Mkdir(filepath.Join(cacheFolder, test.expectedPath), 0700)
				require.NoError(t, err)
			}

			folder, found := subject.Hit(test.cacheID)
			if test.isPresent {
				require.True(t, found)
				require.Equal(t, filepath.Join(cacheFolder, test.expectedPath), folder)
			} else {
				require.False(t, found)
				require.Equal(t, "", folder)
			}
		})
	}
}

func TestSave(t *testing.T) {
	t.Run("copies the files from folder to cache", func(t *testing.T) {
		cacheFolder, err := os.MkdirTemp("", "vendir-cache-save-test")
		require.NoError(t, err)
		defer os.RemoveAll(cacheFolder)
		subject := cache.NewCache(cacheFolder)

		src, err := os.MkdirTemp("", "source")
		require.NoError(t, err)
		defer os.RemoveAll(src)
		createRandomFile(t, filepath.Join(src, "file1.txt"), 500, 0555)
		createRandomFile(t, filepath.Join(src, "file2.txt"), 500, 0400)
		err = os.Mkdir(filepath.Join(src, "folder1"), 0700)
		require.NoError(t, err)
		createRandomFile(t, filepath.Join(src, "folder1", "file3.txt"), 500, 0555)

		err = subject.Save("to-save", src)
		require.NoError(t, err)

		outputFolder, err := os.MkdirTemp("", "vendir-cache-save-output-test")
		require.NoError(t, err)
		defer os.RemoveAll(outputFolder)
		err = subject.CopyFrom("to-save", outputFolder)
		require.NoError(t, err)
		require.FileExists(t, filepath.Join(outputFolder, "file1.txt"))
		require.FileExists(t, filepath.Join(outputFolder, "file2.txt"))
		require.FileExists(t, filepath.Join(outputFolder, "folder1", "file3.txt"))
		compareFiles(t, filepath.Join(src, "file1.txt"), filepath.Join(outputFolder, "file1.txt"))
		compareFiles(t, filepath.Join(src, "file2.txt"), filepath.Join(outputFolder, "file2.txt"))
		compareFiles(t, filepath.Join(src, "folder1", "file3.txt"), filepath.Join(outputFolder, "folder1", "file3.txt"))
	})

	t.Run("when save called twice with same id it deletes previous entry", func(t *testing.T) {
		cacheFolder, err := os.MkdirTemp("", "vendir-cache-save-test")
		require.NoError(t, err)
		defer os.RemoveAll(cacheFolder)
		subject := cache.NewCache(cacheFolder)

		src, err := os.MkdirTemp("", "source")
		require.NoError(t, err)
		defer os.RemoveAll(src)
		createRandomFile(t, filepath.Join(src, "file1.txt"), 500, 0555)

		err = subject.Save("to-save", src)
		require.NoError(t, err)

		src2, err := os.MkdirTemp("", "source-2")
		require.NoError(t, err)
		defer os.RemoveAll(src)
		createRandomFile(t, filepath.Join(src2, "file2.txt"), 500, 0400)

		err = subject.Save("to-save", src2)
		require.NoError(t, err)

		folder, hit := subject.Hit("to-save")
		require.True(t, hit)
		require.NoFileExists(t, filepath.Join(folder, "file1.txt"))
		require.FileExists(t, filepath.Join(folder, "file2.txt"))
	})
}

func createRandomFile(t *testing.T, path string, size int, perm fs.FileMode) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	require.NoError(t, err, "Creating random file")
	defer f.Close()

	_, err = f.Write([]byte(randString(size)))
	require.NoError(t, err, "Writing to file")
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func compareFiles(t *testing.T, expected, actual string) {
	t.Helper()
	actualBs, err := os.ReadFile(actual)
	require.NoError(t, err, "reading actual")

	expectedBs, err := os.ReadFile(expected)
	require.NoError(t, err, "reading expected")

	require.Equal(t, string(expectedBs), string(actualBs))
	expectedStat, err := os.Lstat(expected)
	require.NoError(t, err, "lstat of expected file")
	actualStat, err := os.Lstat(actual)
	require.NoError(t, err, "lstat of actual file")
	require.Equal(t, expectedStat.Mode(), actualStat.Mode())
}
