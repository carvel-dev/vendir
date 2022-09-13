// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagedesc

import (
	"io"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	regv1types "github.com/google/go-containerregistry/pkg/v1/types"
)

type RegistryRemoteImage interface {
	Get(reference regname.Reference) (*remote.Descriptor, error)
}

type ImageOrIndex struct {
	Image *ImageWithRef
	Index *ImageIndexWithRef

	Labels map[string]string

	OrigRef string
}
type ImageWithRef interface {
	regv1.Image
	Ref() string
	Tag() string
}

type ImageIndexWithRef interface {
	regv1.ImageIndex
	Ref() string
	Tag() string
}

type LayerProvider interface {
	FindLayer(ImageLayerDescriptor) (LayerContents, error)
}

type LayerContents interface {
	Open() (io.ReadCloser, error)
}

type ImageOrImageIndexDescriptor struct {
	ImageIndex *ImageIndexDescriptor
	Image      *ImageDescriptor
}

type ImageIndexDescriptor struct {
	Refs    []string
	Images  []ImageDescriptor
	Indexes []ImageIndexDescriptor

	MediaType string
	Digest    string
	Raw       string
	Tag       string
	OrigRef   string

	Labels map[string]string
}

type ImageDescriptor struct {
	Refs   []string
	Layers []ImageLayerDescriptor

	Config   ConfigDescriptor
	Manifest ManifestDescriptor
	Tag      string
	OrigRef  string
	Labels   map[string]string
}

type ImageLayerDescriptor struct {
	MediaType string
	Digest    string
	DiffID    string
	Size      int64
}

type ConfigDescriptor struct {
	Digest string
	Raw    string
}

type ManifestDescriptor struct {
	MediaType string
	Digest    string
	Raw       string
}

func (td ImageLayerDescriptor) IsDistributable() bool {
	// Example layer representation for windows rootfs:
	//   "mediaType": "application/vnd.docker.image.rootfs.foreign.diff.tar.gzip",
	//   "size": 1654613376,
	//   "digest": "sha256:31f9df80631e7b5d379647ee7701ff50e009bd2c03b30a67a0a8e7bba4a26f94",
	//   "urls": ["https://mcr.microsoft.com/v2/windows/servercore/blobs/sha256:31f9df80631e7b5d379647ee7701ff50e009bd2c03b30a67a0a8e7bba4a26f94"]
	return regv1types.MediaType(td.MediaType).IsDistributable()
}

func (td ImageOrImageIndexDescriptor) SortKey() string {
	switch {
	case td.ImageIndex != nil:
		return td.ImageIndex.SortKey()
	case td.Image != nil:
		return td.Image.SortKey()
	default:
		panic("ImageOrImageIndexDescriptor: expected imageIndex or image to be non-nil")
	}
}

// OrigRef returns ImagesLock's image value
// of an Image or an ImageIndex
func (td ImageOrImageIndexDescriptor) OrigRef() string {
	switch {
	case td.Image != nil:
		return (*td.Image).OrigRef
	case td.ImageIndex != nil:
		return (*td.ImageIndex).OrigRef
	default:
		panic("Unknown item")
	}
}

func (td ImageIndexDescriptor) SortKey() string { return td.Digest }
func (td ImageDescriptor) SortKey() string      { return td.Manifest.Digest + "/" + td.Config.Digest }

func (t ImageOrIndex) Digest() (regv1.Hash, error) {
	switch {
	case t.Image != nil:
		return (*t.Image).Digest()
	case t.Index != nil:
		return (*t.Index).Digest()
	default:
		panic("Unknown item")
	}
}

func (t ImageOrIndex) Ref() string {
	switch {
	case t.Image != nil:
		return (*t.Image).Ref()
	case t.Index != nil:
		return (*t.Index).Ref()
	default:
		panic("Unknown item")
	}
}

func (t ImageOrIndex) Tag() string {
	switch {
	case t.Image != nil:
		return (*t.Image).Tag()
	case t.Index != nil:
		return (*t.Index).Tag()
	default:
		panic("Unknown item")
	}
}
