// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package imagedesc

import (
	"encoding/json"
	"fmt"

	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type DescribedImage struct {
	desc          ImageDescriptor
	layerProvider LayerProvider
}

var _ regv1.Image = DescribedImage{}

func NewDescribedImage(desc ImageDescriptor, layerProvider LayerProvider) DescribedImage {
	return DescribedImage{desc, layerProvider}
}

func (i DescribedImage) Ref() string { return i.desc.Refs[0] }

func (i DescribedImage) Tag() string { return i.desc.Tag }

// Layers returns the ordered collection of filesystem layers that comprise this image.
// The order of the list is oldest/base layer first, and most-recent/top layer last.
func (i DescribedImage) Layers() ([]regv1.Layer, error) {
	var layers []regv1.Layer
	for _, layerTD := range i.desc.Layers {
		layerFile, err := i.layerProvider.FindLayer(layerTD)
		if err != nil {
			return nil, err
		}
		layers = append(layers, NewDescribedCompressedLayer(layerTD, layerFile))
	}
	return layers, nil
}

// MediaType of this image's manifest.
func (i DescribedImage) MediaType() (types.MediaType, error) {
	return types.MediaType(i.desc.Manifest.MediaType), nil
}

// ConfigName returns the hash of the image's config file.
func (i DescribedImage) ConfigName() (regv1.Hash, error) {
	return regv1.NewHash(i.desc.Config.Digest)
}

// ConfigFile returns this image's config file.
func (i DescribedImage) ConfigFile() (*regv1.ConfigFile, error) {
	var config *regv1.ConfigFile
	err := json.Unmarshal([]byte(i.desc.Config.Raw), &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// RawConfigFile returns the serialized bytes of ConfigFile()
func (i DescribedImage) RawConfigFile() ([]byte, error) {
	return []byte(i.desc.Config.Raw), nil
}

// Digest returns the sha256 of this image's manifest.
func (i DescribedImage) Digest() (regv1.Hash, error) {
	return regv1.NewHash(i.desc.Manifest.Digest)
}

// Manifest returns this image's Manifest object.
func (i DescribedImage) Manifest() (*regv1.Manifest, error) {
	var manifest *regv1.Manifest
	err := json.Unmarshal([]byte(i.desc.Manifest.Raw), &manifest)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

// RawManifest returns the serialized bytes of Manifest()
func (i DescribedImage) RawManifest() ([]byte, error) {
	return []byte(i.desc.Manifest.Raw), nil
}

func (i DescribedImage) Size() (int64, error) {
	return int64(len(i.desc.Manifest.Raw)), nil
}

// LayerByDigest returns a Layer for interacting with a particular layer of
// the image, looking it up by "digest" (the compressed hash).
func (i DescribedImage) LayerByDigest(digest regv1.Hash) (regv1.Layer, error) {
	for _, layerTD := range i.desc.Layers {
		if layerTD.Digest == digest.String() {
			layerFile, err := i.layerProvider.FindLayer(layerTD)
			if err != nil {
				return nil, err
			}
			return NewDescribedCompressedLayer(layerTD, layerFile), nil
		}
	}
	return nil, fmt.Errorf("Expected to find layer '%s' by digest", digest)
}

// LayerByDiffID is an analog to LayerByDigest, looking up by "diff id"
// (the uncompressed hash).
func (i DescribedImage) LayerByDiffID(diffID regv1.Hash) (regv1.Layer, error) {
	for _, layerTD := range i.desc.Layers {
		if layerTD.DiffID == diffID.String() {
			layerFile, err := i.layerProvider.FindLayer(layerTD)
			if err != nil {
				return nil, err
			}
			return NewDescribedCompressedLayer(layerTD, layerFile), nil
		}
	}
	return nil, fmt.Errorf("Expected to find layer '%s' by diff id", diffID)
}
