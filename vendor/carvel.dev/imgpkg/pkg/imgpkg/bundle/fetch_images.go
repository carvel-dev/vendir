// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"fmt"

	"carvel.dev/imgpkg/pkg/imgpkg/lockconfig"
	"carvel.dev/imgpkg/pkg/imgpkg/signature"
	regname "github.com/google/go-containerregistry/pkg/name"
)

// SignatureFetcher Interface to retrieve signatures associated with Images
type SignatureFetcher interface {
	FetchForImageRefs(images []lockconfig.ImageRef) ([]lockconfig.ImageRef, error)
}

// FetchAllImagesRefs returns a flat list of nested bundles and every image reference for a specific bundle
func (o *Bundle) FetchAllImagesRefs(concurrency int, ui Logger, sigFetcher SignatureFetcher) ([]*Bundle, error) {
	bundles, _, err := o.AllImagesLockRefs(concurrency, ui)
	if err != nil {
		return nil, err
	}

	for _, bundle := range bundles {
		imgs := []lockconfig.ImageRef{{
			Image: bundle.DigestRef(),
		}}
		for _, ref := range bundle.cachedImageRefs.All() {
			imgs = append(imgs, ref.ImageRef)
		}
		refs, err := sigFetcher.FetchForImageRefs(imgs)
		if err != nil {
			fetchErr, ok := err.(*signature.FetchError)
			if !ok {
				return nil, err
			}
			for _, err := range fetchErr.AllErrors {
				bundle.cachedImageRefs.StoreImageRef(
					NewImageRefWithTypeAndError(
						lockconfig.ImageRef{Image: err.ImageRef()}, SignatureImage, err.Error()))
			}
		}

		for _, ref := range refs {
			bundle.cachedImageRefs.StoreImageRef(NewImageRefWithType(ref, SignatureImage))
		}

		// Get the Locations image for this particular bundle
		// If there is no locations image present it just skips to the next bundle
		bundleRef, err := regname.NewDigest(bundle.plainImg.DigestRef())
		if err != nil {
			panic(fmt.Sprintf("Internal inconsistency: '%s' have to be a digest", bundle.plainImg.DigestRef()))
		}

		locationsImageRef, err := NewLocations(ui).LocationsImageDigest(o.imgRetriever, bundleRef)
		if err != nil {
			if _, ok := err.(*LocationsNotFound); ok {
				continue
			}
			return nil, err
		}

		bundle.cachedImageRefs.StoreImageRef(NewImageRefWithType(
			lockconfig.ImageRef{
				Image: locationsImageRef.String(),
			},
			InternalImage,
		))
	}

	return bundles, nil
}
