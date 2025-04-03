// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package imagetar

import (
	"fmt"
	"io"
	"sync"

	"carvel.dev/imgpkg/pkg/imgpkg/imagedesc"
	"carvel.dev/imgpkg/pkg/imgpkg/imageutils/verify"
	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type TarReader struct {
	path string

	throttle util.Throttle
	lock     sync.Mutex
}

// NewTarReader creates a new TarReader struct
func NewTarReader(path string, concurrency int) TarReader {
	return TarReader{path: path, throttle: util.NewThrottle(concurrency), lock: sync.Mutex{}}
}

func (r TarReader) Read() ([]imagedesc.ImageOrIndex, error) {
	file := tarFile{r.path}

	ids, err := r.getIdsFromManifest(file)
	if err != nil {
		return nil, err
	}

	return imagedesc.NewDescribedReader(ids, file).Read(), nil
}

// PresentLayers retrieves all the layers that are present in a tar file
func (r *TarReader) PresentLayers() ([]v1.Layer, error) {
	allImages, err := r.Read()
	if err != nil {
		return nil, err
	}
	layerCache := map[string]v1.Layer{}
	for _, image := range allImages {
		if image.Image != nil {
			img := *image.Image
			err = r.presentLayersForImage(img, layerCache)
			if err != nil {
				return nil, fmt.Errorf("Processing Image %s: %s", image.OrigRef, err)
			}
		} else if image.Index != nil {
			idx := *image.Index
			err = r.presentLayersForIndex(image.Ref(), idx, layerCache)
			if err != nil {
				return nil, fmt.Errorf("Processing Index %s: %s", image.OrigRef, err)
			}
		}
	}

	var result []v1.Layer
	for _, layer := range layerCache {
		if layer != nil {
			result = append(result, layer)
		}
	}

	return result, nil
}

func (r *TarReader) presentLayersForImage(img v1.Image, layerCache map[string]v1.Layer) error {
	layers, err := img.Layers()
	if err != nil {
		return fmt.Errorf("Unable to retrieve layers: %s", err)
	}

	errChan := make(chan error, len(layers))
	for _, layer := range layers {
		h, err := layer.Digest()
		if err != nil {
			return fmt.Errorf("Unable to get digest from layer: %s", err)
		}
		r.lock.Lock()
		if _, ok := layerCache[h.String()]; ok {
			errChan <- nil
			r.lock.Unlock()
			continue
		}
		// Initializing the cache to ensure that if this layer appears again in the loop
		// we do not try to process it again
		layerCache[h.String()] = nil
		r.lock.Unlock()

		go func() {
			layer := layer
			h := h

			reader, err := layer.Compressed()
			if err != nil {
				errChan <- nil
				return
			}

			size, err := layer.Size()
			if err != nil {
				errChan <- err
				return
			}
			closer, err := verify.ReadCloser(reader, size, h)
			if err != nil {
				errChan <- err
				return
			}

			r.throttle.Take()
			_, err = io.Copy(io.Discard, closer)
			r.throttle.Done()
			if err != nil {
				errChan <- nil
				return
			}

			r.lock.Lock()
			layerCache[h.String()] = layer
			r.lock.Unlock()
			errChan <- nil
		}()
	}

	for range layers {
		if err := <-errChan; err != nil {
			return err
		}
	}
	return nil
}

func (r *TarReader) presentLayersForIndex(indexRef string, idx v1.ImageIndex, layerCache map[string]v1.Layer) error {
	dIdx, correct := idx.(imagedesc.DescribedImageIndex)
	if !correct {
		panic(fmt.Sprintf("Internal inconsistency: unexpected index type with ref: %s", indexRef))
	}
	for _, image := range dIdx.Images() {
		err := r.presentLayersForImage(image, layerCache)
		if err != nil {
			return err
		}
	}

	idxRef, err := name.ParseReference(indexRef)
	if err != nil {
		return err
	}

	for _, idx := range dIdx.Indexes() {
		digest, err := idx.Digest()
		if err != nil {
			return err
		}
		idxDigest := idxRef.Context().Digest(digest.String())
		err = r.presentLayersForIndex(idxDigest.String(), idx, layerCache)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r TarReader) getIdsFromManifest(file tarFile) (*imagedesc.ImageRefDescriptors, error) {
	manifestFile, err := file.Chunk("manifest.json").Open()
	if err != nil {
		return nil, err
	}
	defer manifestFile.Close()

	manifestBytes, err := io.ReadAll(manifestFile)
	if err != nil {
		return nil, err
	}

	ids, err := imagedesc.NewImageRefDescriptorsFromBytes(manifestBytes)
	if err != nil {
		return nil, err
	}
	return ids, nil
}
