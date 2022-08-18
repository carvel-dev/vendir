// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagedesc

import (
	"encoding/json"
	"fmt"

	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type DescribedImageIndex struct {
	desc    ImageIndexDescriptor
	images  []regv1.Image
	indexes []regv1.ImageIndex
}

var _ regv1.ImageIndex = DescribedImageIndex{}

func NewDescribedImageIndex(desc ImageIndexDescriptor,
	images []regv1.Image, indexes []regv1.ImageIndex) DescribedImageIndex {
	return DescribedImageIndex{desc, images, indexes}
}

func (i DescribedImageIndex) Ref() string { return i.desc.Refs[0] }

func (i DescribedImageIndex) MediaType() (types.MediaType, error) {
	return types.MediaType(i.desc.MediaType), nil
}

func (i DescribedImageIndex) Tag() string {
	return i.desc.Tag
}

func (i DescribedImageIndex) Digest() (regv1.Hash, error)  { return regv1.NewHash(i.desc.Digest) }
func (i DescribedImageIndex) RawManifest() ([]byte, error) { return []byte(i.desc.Raw), nil }

func (i DescribedImageIndex) Size() (int64, error) { return int64(len(i.desc.Raw)), nil }

func (i DescribedImageIndex) IndexManifest() (*regv1.IndexManifest, error) {
	var manifest *regv1.IndexManifest
	err := json.Unmarshal([]byte(i.desc.Raw), &manifest)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

// Image retrieve the image that matches the provided digest
func (i DescribedImageIndex) Image(digest regv1.Hash) (regv1.Image, error) {
	for _, img := range i.images {
		imgDigest, err := img.Digest()
		if err != nil {
			return nil, err
		}
		if imgDigest.String() == digest.String() {
			return img, nil
		}
	}
	return nil, fmt.Errorf("Expected to find image '%s' by digest", digest)
}

// Images retrieve all images associated with the described image
func (i DescribedImageIndex) Images() []regv1.Image {
	return i.images
}

// Indexes retrieve all indexes associated with the described image
func (i DescribedImageIndex) Indexes() []regv1.ImageIndex {
	return i.indexes
}

// ImageIndex retrieve the index that matches the provided digest
func (i DescribedImageIndex) ImageIndex(digest regv1.Hash) (regv1.ImageIndex, error) {
	for _, idx := range i.indexes {
		idxDigest, err := idx.Digest()
		if err != nil {
			return nil, err
		}
		if idxDigest.String() == digest.String() {
			return idx, nil
		}
	}
	return nil, fmt.Errorf("Expected to find index '%s' by digest", digest)
}
