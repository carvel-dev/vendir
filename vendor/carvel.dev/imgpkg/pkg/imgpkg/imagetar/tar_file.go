// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package imagetar

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"strings"

	"carvel.dev/imgpkg/pkg/imgpkg/imagedesc"
	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
)

type tarFile struct {
	path string
}

var _ imagedesc.LayerProvider = tarFile{}

type tarFileChunk struct {
	file      tarFile
	chunkPath string
}

var _ imagedesc.LayerContents = tarFileChunk{}

type tarFileChunkReadCloser struct {
	DebugID string
	io.Reader
	io.Closer
}

func (f tarFile) Chunk(path string) tarFileChunk {
	return tarFileChunk{f, path}
}

func (f tarFile) FindLayer(layerTD imagedesc.ImageLayerDescriptor) (imagedesc.LayerContents, error) {
	digest, err := regv1.NewHash(layerTD.Digest)
	if err != nil {
		return nil, err
	}
	return tarFileChunk{f, digest.Algorithm + "-" + digest.Hex + ".tar.gz"}, nil
}

func (f tarFileChunk) Open() (io.ReadCloser, error) {
	return f.file.openChunk(f.chunkPath)
}

func (f tarFile) openChunk(path string) (io.ReadCloser, error) {
	file, err := os.Open(f.path)
	if err != nil {
		return nil, err
	}
	tf := tar.NewReader(file)
	for {
		hdr, err := tf.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Name == path {
			return tarFileChunkReadCloser{
				DebugID: fmt.Sprintf("%s/%p", path, tf),
				Reader:  tf, Closer: file}, nil
		}
	}
	return nil, util.NonRetryableError{Message: fmt.Sprintf("file %s not found in tar (hint: This may be because when copying to a tarball, the --include-non-distributable-layers flag should have been provided.)", path)}
}

func (f tarFileChunkReadCloser) Close() error {
	// It seems that there is a race between go-containerregistry library
	// and net/http's transport to close the request body. Specifically
	// between net/http.(*transferWriter).writeBody(...) and
	// gcrr.pkg/v1/remote.(*writer).streamBlob(...).
	// Example trace: https://gist.github.com/cppforlife/aeeb989c83aebd061561d12524c6476b
	// Following lines are used to identify locations where Close()
	// method is being called during runtime.
	//   fmt.Printf("%s: close\n>>>%s<<<\n\n", f.DebugId, debug.Stack())

	err := f.Closer.Close()
	if err != nil {
		// Ignore dup close error since closing file twice does
		// not have any side-effects (except just being conceptually wrong)
		if strings.Contains(err.Error(), "file already closed") {
			// fmt.Printf("%s: dup close\n\n", f.DebugId)
			return nil
		}
	}
	return err
}
