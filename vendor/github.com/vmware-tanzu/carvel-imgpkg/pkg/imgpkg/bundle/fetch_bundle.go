// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"fmt"

	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imageset"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/plainimage"
)

// Fetcher Interface that will fetch the bundle
type Fetcher interface {
	// Bundle search for the imgRef Digest
	// only returns the *Bundle if the current image is a bundle, if not the return value will be nil
	Bundle(throttleReq *util.Throttle, imgRef ImageRef) (lockconfig.ImageRef, *Bundle, error)
}

var _ Fetcher = &RegistryFetcher{}
var _ Fetcher = &FetcherFromProcessedImages{}

// NewRegistryFetcher Creates a bundle Fetcher that talks to the registry
func NewRegistryFetcher(imgRetriever ImagesMetadata, imagesLockReader ImagesLockReader) *RegistryFetcher {
	return &RegistryFetcher{
		imgRetriever:     imgRetriever,
		imagesLockReader: imagesLockReader,
		cacheMutex:       newKeyLock(),
		cache:            map[string]registryImagesRefCacheEntry{},
	}
}

// NewFetcherFromProcessedImages Creates a bundle Fetcher that reads the information from previously processed images
func NewFetcherFromProcessedImages(processedImages []imageset.ProcessedImage, imgRetriever ImagesMetadata, imagesLockReader ImagesLockReader) *FetcherFromProcessedImages {
	return &FetcherFromProcessedImages{
		processedImages:  processedImages,
		imgRetriever:     imgRetriever,
		imagesLockReader: imagesLockReader,
	}
}

// FetcherFromProcessedImages struct that implements Fetcher and searches for the bundle in preprocessed images
type FetcherFromProcessedImages struct {
	processedImages  []imageset.ProcessedImage
	imgRetriever     ImagesMetadata
	imagesLockReader ImagesLockReader
}

// Bundle search for the imgRef Digest on the preprocessed images
// only returns the *Bundle if the current image is a bundle, if not the return value will be nil
func (t *FetcherFromProcessedImages) Bundle(_ *util.Throttle, imgRef ImageRef) (lockconfig.ImageRef, *Bundle, error) {
	img := imageset.ProcessedImage{}
	for _, image := range t.processedImages {
		digest, err := regname.NewDigest(image.DigestRef)
		if err != nil {
			return lockconfig.ImageRef{}, nil, err
		}

		if digest.DigestStr() == imgRef.Digest() {
			img = image
			break
		}
	}
	if img.DigestRef == "" {
		panic(fmt.Sprintf("Internal inconsistency: was not able to find '%s' in the list of procced images", imgRef.Image))
	}
	if img.ImageIndex != nil {
		return imgRef.ImageRef, nil, nil
	}

	b := NewBundle(plainimage.NewFetchedPlainImageWithTag(img.DigestRef, img.Tag, img.Image), t.imgRetriever, t.imagesLockReader, t)
	isBundle, err := b.IsBundle()
	if err != nil {
		return lockconfig.ImageRef{}, nil, fmt.Errorf("Checking if '%s' is a bundle: %s", imgRef.Image, err)
	}

	if isBundle {
		return imgRef.ImageRef, b, nil
	}
	return imgRef.ImageRef, nil, nil
}

type registryImagesRefCacheEntry struct {
	bundle *Bundle
	imgRef lockconfig.ImageRef
}

// RegistryFetcher struct that implements Fetcher and searches for the bundle in the registry
type RegistryFetcher struct {
	imgRetriever     ImagesMetadata
	imagesLockReader ImagesLockReader

	cache      map[string]registryImagesRefCacheEntry
	cacheMutex *keyLock
}

// Bundle search for the imgRef Digest on the registry
// only returns the *Bundle if the current image is a bundle, if not the return value will be nil
func (r *RegistryFetcher) Bundle(throttleReq *util.Throttle, imgRef ImageRef) (lockconfig.ImageRef, *Bundle, error) {
	location := imgRef.PrimaryLocation()
	r.cacheMutex.Lock(location)
	defer r.cacheMutex.Unlock(location)
	cacheEntry, found := r.getLocationEntry(location)
	if found {
		return cacheEntry.imgRef, cacheEntry.bundle, nil
	}

	throttleReq.Take()
	// We need to check where we can find the image we are looking for.
	// First checks the current bundle repository and if it cannot be found there
	// it will check in the original location of the image
	imgURL, err := r.imgRetriever.FirstImageExists(imgRef.Locations())
	throttleReq.Done()
	if err != nil {
		return lockconfig.ImageRef{}, nil, err
	}
	newImgRef := imgRef.DiscardLocationsExcept(imgURL)

	bundle := NewBundleFromRef(newImgRef.PrimaryLocation(), r.imgRetriever, r.imagesLockReader, r)

	throttleReq.Take()
	isBundle, err := bundle.IsBundle()
	throttleReq.Done()
	if err != nil {
		return lockconfig.ImageRef{}, nil, fmt.Errorf("Checking if '%s' is a bundle: %s", imgRef.Image, err)
	}

	if isBundle {
		r.storeLocationEntry(location, bundle, newImgRef)
		return newImgRef, bundle, nil
	}
	r.storeLocationEntry(location, nil, newImgRef)
	return newImgRef, nil, nil
}

func (r *RegistryFetcher) getLocationEntry(location string) (registryImagesRefCacheEntry, bool) {
	r.cacheMutex.Lock("on-cache")
	defer r.cacheMutex.Unlock("on-cache")
	cacheEntry, found := r.cache[location]
	return cacheEntry, found
}

func (r *RegistryFetcher) storeLocationEntry(location string, bundle *Bundle, imgRef lockconfig.ImageRef) {
	r.cacheMutex.Lock("on-cache")
	defer r.cacheMutex.Unlock("on-cache")
	r.cache[location] = registryImagesRefCacheEntry{
		bundle: bundle,
		imgRef: imgRef,
	}
}
