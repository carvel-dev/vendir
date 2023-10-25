// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"io"
	"os"

	regv1 "github.com/google/go-containerregistry/pkg/v1"
	regpartial "github.com/google/go-containerregistry/pkg/v1/partial"
	regtypes "github.com/google/go-containerregistry/pkg/v1/types"
)

type UncompressedFileLayer struct {
	diffID    regv1.Hash
	mediaType regtypes.MediaType
	path      string
}

var _ regpartial.UncompressedLayer = (*UncompressedFileLayer)(nil)

func (ul *UncompressedFileLayer) DiffID() (regv1.Hash, error) {
	return ul.diffID, nil
}

func (ul *UncompressedFileLayer) Uncompressed() (io.ReadCloser, error) {
	file, err := os.Open(ul.path)
	return file, err
}

func (ul *UncompressedFileLayer) MediaType() (regtypes.MediaType, error) {
	return ul.mediaType, nil
}
