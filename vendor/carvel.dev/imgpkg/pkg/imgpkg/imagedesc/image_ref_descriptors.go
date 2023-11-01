// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagedesc

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	regtran "github.com/google/go-containerregistry/pkg/v1/remote/transport"
	regtypes "github.com/google/go-containerregistry/pkg/v1/types"
	"golang.org/x/sync/errgroup"
)

type Registry interface {
	Get(regname.Reference) (*regremote.Descriptor, error)
	Digest(regname.Reference) (regv1.Hash, error)
	Index(regname.Reference) (regv1.ImageIndex, error)
	Image(regname.Reference) (regv1.Image, error)
}

type Metadata struct {
	Ref     regname.Reference
	Tag     string
	Labels  map[string]string
	OrigRef string
}

type ImageRefDescriptors struct {
	registry Registry

	descs []ImageOrImageIndexDescriptor

	imageLayersLock sync.Mutex
	imageLayers     map[ImageLayerDescriptor]regv1.Layer
}

func NewImageRefDescriptorsFromBytes(data []byte) (*ImageRefDescriptors, error) {
	var descs []ImageOrImageIndexDescriptor

	err := json.Unmarshal(data, &descs)
	if err != nil {
		return nil, err
	}

	return &ImageRefDescriptors{descs: descs}, nil
}

func NewImageRefDescriptors(refs []Metadata, registry Registry) (*ImageRefDescriptors, error) {
	registry = errRegistry{registry}

	imageRefDescs := &ImageRefDescriptors{
		registry:    registry,
		imageLayers: map[ImageLayerDescriptor]regv1.Layer{},
	}

	var imageRefDescsLock sync.Mutex
	var wg errgroup.Group
	buildThrottle := util.NewThrottle(10)

	for _, ref := range refs {
		ref := ref //copy

		wg.Go(func() error {
			buildThrottle.Take()
			defer buildThrottle.Done()

			regDesc, err := registry.Get(ref.Ref)
			if err != nil {
				return err
			}

			var td ImageOrImageIndexDescriptor

			if imageRefDescs.isImageIndex(regDesc.Descriptor) {
				imgIndexTd, err := imageRefDescs.buildImageIndex(ref, regDesc.Descriptor)

				if err != nil {
					return err
				}

				td = ImageOrImageIndexDescriptor{ImageIndex: &imgIndexTd}
			} else {
				img, err := imageRefDescs.buildImage(ref)
				if err != nil {
					return err
				}

				td = ImageOrImageIndexDescriptor{Image: &img}
			}

			imageRefDescsLock.Lock()
			imageRefDescs.descs = append(imageRefDescs.descs, td)
			imageRefDescsLock.Unlock()

			return nil
		})
	}

	err := wg.Wait()

	return imageRefDescs, err
}

func (ids *ImageRefDescriptors) Descriptors() []ImageOrImageIndexDescriptor {
	return ids.descs
}

func (ids *ImageRefDescriptors) buildImageIndex(ref Metadata, regDesc regv1.Descriptor) (ImageIndexDescriptor, error) {
	td := ImageIndexDescriptor{
		Refs:      []string{ref.Ref.Name()},
		MediaType: string(regDesc.MediaType),
		Digest:    regDesc.Digest.String(),
		Tag:       ref.Tag,
		Labels:    ref.Labels,
		OrigRef:   ref.OrigRef,
	}

	imgIndex, err := ids.registry.Index(ref.Ref)
	if err != nil {
		return td, err
	}

	rawManifest, err := imgIndex.RawManifest()
	if err != nil {
		return td, err
	}

	td.Raw = string(rawManifest)

	imgIndexManifest, err := imgIndex.IndexManifest()
	if err != nil {
		return td, err
	}

	for _, manDesc := range imgIndexManifest.Manifests {
		if ids.isImageIndex(manDesc) {
			imgIndexTd, err := ids.buildImageIndex(Metadata{ids.buildRef(ref.Ref, manDesc.Digest.String()), ref.Tag, ref.Labels, ref.OrigRef}, manDesc)
			if err != nil {
				return ImageIndexDescriptor{}, err
			}
			td.Indexes = append(td.Indexes, imgIndexTd)
		} else {
			imgTd, err := ids.buildImage(Metadata{ids.buildRef(ref.Ref, manDesc.Digest.String()), ref.Tag, ref.Labels, ref.OrigRef})
			if err != nil {
				return ImageIndexDescriptor{}, err
			}
			td.Images = append(td.Images, imgTd)
		}
	}

	return td, nil
}

func (ids *ImageRefDescriptors) buildImage(ref Metadata) (ImageDescriptor, error) {
	td := ImageDescriptor{}

	img, err := ids.registry.Image(ref.Ref)
	if err != nil {
		return td, err
	}

	cfgDigest, err := img.ConfigName()
	if err != nil {
		return td, err
	}
	cfgBlob, err := img.RawConfigFile()
	if err != nil {
		return td, err
	}

	manifestMediaType, err := img.MediaType()
	if err != nil {
		return td, err
	}
	manifestDigest, err := img.Digest()
	if err != nil {
		return td, err
	}
	manifestBlob, err := img.RawManifest()
	if err != nil {
		return td, err
	}

	td = ImageDescriptor{
		Refs: []string{ref.Ref.String()},

		Config: ConfigDescriptor{
			Digest: cfgDigest.String(),
			Raw:    string(cfgBlob),
		},

		Manifest: ManifestDescriptor{
			MediaType: string(manifestMediaType),
			Digest:    manifestDigest.String(),
			Raw:       string(manifestBlob),
		},
		Tag:     ref.Tag,
		Labels:  ref.Labels,
		OrigRef: ref.OrigRef,
	}

	layers, err := img.Layers()
	if err != nil {
		return td, err
	}

	for _, layer := range layers {
		layerMediaType, err := layer.MediaType()
		if err != nil {
			return td, err
		}
		layerDigest, err := layer.Digest()
		if err != nil {
			return td, err
		}
		layerDiffID, err := layer.DiffID()
		if err != nil {
			return td, err
		}
		layerSize, err := layer.Size()
		if err != nil {
			return td, err
		}

		layerTD := ImageLayerDescriptor{
			MediaType: string(layerMediaType),
			Digest:    layerDigest.String(),
			DiffID:    layerDiffID.String(),
			Size:      layerSize,
		}

		td.Layers = append(td.Layers, layerTD)

		ids.imageLayersLock.Lock()
		ids.imageLayers[layerTD] = layer
		ids.imageLayersLock.Unlock()
	}

	return td, nil
}

func (*ImageRefDescriptors) isImageIndex(regDesc regv1.Descriptor) bool {
	switch regDesc.MediaType {
	case regtypes.OCIImageIndex, regtypes.DockerManifestList:
		return true
	}
	return false
}

type wrappedCompressedLayerContents struct {
	layer regv1.Layer
}

var _ LayerContents = wrappedCompressedLayerContents{}

func (lc wrappedCompressedLayerContents) Open() (io.ReadCloser, error) {
	rc, err := lc.layer.Compressed()
	if err != nil {
		return nil, fmt.Errorf("Getting compressed layer: %s", err)
	}
	return rc, nil
}

func (ids *ImageRefDescriptors) FindLayer(layerTD ImageLayerDescriptor) (LayerContents, error) {
	ids.imageLayersLock.Lock()
	defer ids.imageLayersLock.Unlock()

	layer, found := ids.imageLayers[layerTD]
	if !found {
		panic(fmt.Sprintf("Expected to find stream for %#v", layerTD))
	}

	return wrappedCompressedLayerContents{layer}, nil
}

func (ids *ImageRefDescriptors) AsBytes() ([]byte, error) {
	// Ensure result is deterministic
	sort.Slice(ids.descs, func(i, j int) bool {
		return ids.descs[i].SortKey() < ids.descs[j].SortKey()
	})

	return json.Marshal(ids.descs)
}

func (ids *ImageRefDescriptors) buildRef(otherRef regname.Reference, digest string) regname.Reference {
	newRef, err := regname.NewDigest(fmt.Sprintf("%s@%s", otherRef.Context().Name(), digest))
	if err != nil {
		panic(fmt.Sprintf("Building new ref"))
	}
	return newRef
}

type errRegistry struct {
	delegate Registry
}

func (m errRegistry) Get(ref regname.Reference) (*regremote.Descriptor, error) {
	regDesc, err := m.delegate.Get(ref)
	return regDesc, m.betterErr(ref, err)
}

func (m errRegistry) Digest(ref regname.Reference) (regv1.Hash, error) {
	regDesc, err := m.delegate.Digest(ref)
	return regDesc, m.betterErr(ref, err)
}

func (m errRegistry) Index(ref regname.Reference) (regv1.ImageIndex, error) {
	idx, err := m.delegate.Index(ref)
	return idx, m.betterErr(ref, err)
}

func (m errRegistry) Image(ref regname.Reference) (regv1.Image, error) {
	img, err := m.delegate.Image(ref)
	return img, m.betterErr(ref, err)
}

func (m errRegistry) betterErr(ref regname.Reference, err error) error {
	if err != nil {
		if strings.Contains(err.Error(), string(regtran.ManifestUnknownErrorCode)) {
			err = fmt.Errorf("Encountered an error most likely because this image is in Docker Registry v1 format; only v2 or OCI image format is supported (underlying error: %s)", err)
		}
		err = fmt.Errorf("Working with %s: %s", ref.Name(), err)
	}
	return err
}
