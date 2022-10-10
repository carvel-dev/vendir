// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package fetch_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
)

func TestMoveFile(t *testing.T) {
	t.Run("Move file to folder that does not exist, creates folder with 0700 permission", func(t *testing.T) {
		testFolder, err := os.MkdirTemp("", "vendir-folder-does-not-exist")
		require.NoError(t, err)
		defer os.RemoveAll(testFolder)

		f, err := os.CreateTemp(testFolder, "some-file")
		require.NoError(t, err)
		_, err = f.Write([]byte("something"))
		require.NoError(t, err)
		require.NoError(t, f.Close())

		dstFolder := filepath.Join(testFolder, "destination-folder")
		require.NoError(t, ctlfetch.MoveFile(f.Name(), dstFolder))

		require.FileExists(t, filepath.Join(dstFolder, filepath.Base(f.Name())))
		info, err := os.Stat(dstFolder)
		require.NoError(t, err)

		require.Equal(t, os.FileMode(0700), info.Mode().Perm())
	})

	t.Run("Move file to folder that does exist, removes folder before moving file", func(t *testing.T) {
		testFolder, err := os.MkdirTemp("", "vendir-folder-does-not-exist")
		require.NoError(t, err)
		defer os.RemoveAll(testFolder)

		f, err := os.CreateTemp(testFolder, "some-file")
		require.NoError(t, err)
		_, err = f.Write([]byte("something"))
		require.NoError(t, err)
		require.NoError(t, f.Close())

		dstFolder := filepath.Join(testFolder, "destination-folder")
		require.NoError(t, os.Mkdir(dstFolder, 0777))
		dstFile, err := os.CreateTemp(testFolder, "some-other-file")
		require.NoError(t, err)
		_, err = dstFile.Write([]byte("something else in a file on the destination folder"))
		require.NoError(t, err)
		require.NoError(t, dstFile.Close())

		require.NoError(t, ctlfetch.MoveFile(f.Name(), dstFolder))

		require.FileExists(t, filepath.Join(dstFolder, filepath.Base(f.Name())))
		require.NoFileExists(t, filepath.Join(dstFolder, "some-other-file"))
		info, err := os.Stat(dstFolder)
		require.NoError(t, err)

		require.Equal(t, os.FileMode(0700), info.Mode().Perm())
	})
}
