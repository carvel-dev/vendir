// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

// Package v1 contains the public API version 1 used by other tools to interact with imgpkg
package v1

import (
	"fmt"
	"sort"

	"carvel.dev/imgpkg/pkg/imgpkg/bundle"
	"carvel.dev/imgpkg/pkg/imgpkg/lockconfig"
	"carvel.dev/imgpkg/pkg/imgpkg/registry"
	"carvel.dev/imgpkg/pkg/imgpkg/signature"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// Author information from a Bundle
type Author struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// Website URL where more information of the Bundle can be found
type Website struct {
	URL string `json:"url,omitempty"`
}

// Metadata Extra metadata present in a Bundle
type Metadata struct {
	Metadata map[string]string `json:"metadata,omitempty"`
	Authors  []Author          `json:"authors,omitempty"`
	Websites []Website         `json:"websites,omitempty"`
}

// Layers image layers info
type Layers struct {
	Digest string `json:"digest,omitempty"`
}

// ImageInfo URLs where the image can be found as well as annotations provided in the Images Lock
type ImageInfo struct {
	Image       string            `json:"image,omitempty"`
	Origin      string            `json:"origin,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	ImageType   bundle.ImageType  `json:"imageType"`
	Error       string            `json:"error,omitempty"`
	Layers      []Layers          `json:"layers,omitempty"`
}

// Content Contents present in a Bundle
type Content struct {
	Bundles map[string]Description `json:"bundles,omitempty"`
	Images  map[string]ImageInfo   `json:"images,omitempty"`
}

// Description Metadata and Contents of a Bundle
type Description struct {
	Image       string            `json:"image"`
	Origin      string            `json:"origin"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Metadata    Metadata          `json:"metadata,omitempty"`
	Content     Content           `json:"content"`
	Layers      []Layers          `json:"layers,omitempty"`
}

// DescribeOpts Options used when calling the Describe function
type DescribeOpts struct {
	Logger                 bundle.Logger
	Concurrency            int
	IncludeCosignArtifacts bool
	Layers                 bool
}

// SignatureFetcher Interface to retrieve signatures associated with Images
type SignatureFetcher interface {
	FetchForImageRefs(images []lockconfig.ImageRef) ([]lockconfig.ImageRef, error)
}

// Describe Given a Bundle URL fetch the information about the contents of the Bundle and Nested Bundles
func Describe(bundleImage string, opts DescribeOpts, registryOpts registry.Opts) (Description, error) {
	reg, err := registry.NewSimpleRegistry(registryOpts)
	if err != nil {
		return Description{}, err
	}

	var signatureRetriever SignatureFetcher
	if !opts.IncludeCosignArtifacts {
		signatureRetriever = signature.NewNoop()
	} else {
		signatureRetriever = signature.NewSignatures(signature.NewCosign(reg), opts.Concurrency)
	}

	return DescribeWithRegistryAndSignatureFetcher(bundleImage, opts, reg, signatureRetriever)
}

// DescribeWithRegistryAndSignatureFetcher Given a Bundle URL fetch the information about the contents of the Bundle and Nested Bundles
func DescribeWithRegistryAndSignatureFetcher(bundleImage string, opts DescribeOpts, reg bundle.ImagesMetadata, sigFetcher SignatureFetcher) (Description, error) {
	lockReader := bundle.NewImagesLockReader()
	newBundle := bundle.NewBundleFromRef(bundleImage, reg, lockReader, bundle.NewRegistryFetcher(reg, lockReader))
	isBundle, err := newBundle.IsBundle()
	if err != nil {
		return Description{}, fmt.Errorf("Unable to check if %s is a bundle: %s", bundleImage, err)
	}
	if !isBundle {
		return Description{}, fmt.Errorf("Only bundles can be described, and %s is not a bundle", bundleImage)
	}

	allBundles, err := newBundle.FetchAllImagesRefs(opts.Concurrency, opts.Logger, sigFetcher)
	if err != nil {
		return Description{}, fmt.Errorf("Retrieving Images from bundle: %s", err)
	}

	topBundle := refWithDescription{
		imgRef: bundle.NewBundleImageRef(lockconfig.ImageRef{Image: newBundle.DigestRef()}),
	}
	return topBundle.DescribeBundle(allBundles, opts.Layers)
}

type refWithDescription struct {
	imgRef bundle.ImageRef
	bundle Description
}

func (r *refWithDescription) DescribeBundle(bundles []*bundle.Bundle, layers bool) (Description, error) {
	var visitedImgs map[string]refWithDescription
	return r.describeBundleRec(visitedImgs, r.imgRef, bundles, layers)
}

func (r *refWithDescription) describeBundleRec(visitedImgs map[string]refWithDescription, currentBundle bundle.ImageRef, bundles []*bundle.Bundle, showLayers bool) (Description, error) {
	desc, wasVisited := visitedImgs[currentBundle.Image]
	var (
		layers []Layers
		err    error
	)
	if wasVisited {
		return desc.bundle, nil
	}

	if showLayers {
		layers, err = getImageLayersInfo(currentBundle.PrimaryLocation())
		if err != nil {
			return desc.bundle, err
		}
	}

	desc = refWithDescription{
		imgRef: currentBundle,
		bundle: Description{
			Image:       currentBundle.PrimaryLocation(),
			Origin:      currentBundle.Image,
			Annotations: currentBundle.Annotations,
			Metadata:    Metadata{},
			Content: Content{
				Bundles: map[string]Description{},
				Images:  map[string]ImageInfo{},
			},
			Layers: layers,
		},
	}
	var newBundle *bundle.Bundle
	for _, b := range bundles {
		if b.DigestRef() == currentBundle.PrimaryLocation() {
			newBundle = b
			break
		}
	}
	if newBundle == nil {
		return desc.bundle, fmt.Errorf("Internal inconsistency: bundle with ref '%s' could not be found in list of bundles", currentBundle.PrimaryLocation())
	}

	imagesRefs := newBundle.ImagesRefsWithErrors()
	sort.Slice(imagesRefs, func(i, j int) bool {
		return imagesRefs[i].Image < imagesRefs[j].Image
	})

	for _, ref := range imagesRefs {
		if ref.IsBundle == nil {
			return desc.bundle, fmt.Errorf("Internal inconsistency: IsBundle after processing must always have a value")
		}

		if *ref.IsBundle {
			bundleDesc, err := r.describeBundleRec(visitedImgs, ref, bundles, showLayers)
			if err != nil {
				return desc.bundle, err
			}

			digest, err := name.NewDigest(bundleDesc.Image)
			if err != nil {
				return desc.bundle, fmt.Errorf("Internal inconsistency: image %s should be fully resolved", bundleDesc.Image)
			}
			desc.bundle.Content.Bundles[digest.DigestStr()] = bundleDesc
		} else {
			if ref.Error == "" {
				digest, err := name.NewDigest(ref.PrimaryLocation())
				if err != nil {
					return desc.bundle, fmt.Errorf("Internal inconsistency: image %s should be fully resolved", ref.Image)
				}
				if showLayers {
					layers, err = getImageLayersInfo(ref.PrimaryLocation())
					if err != nil {
						return desc.bundle, err
					}
				}
				desc.bundle.Content.Images[digest.DigestStr()] = ImageInfo{
					Image:       ref.PrimaryLocation(),
					Origin:      ref.Image,
					Annotations: ref.Annotations,
					ImageType:   ref.ImageType,
					Layers:      layers,
				}
			} else {
				desc.bundle.Content.Images[ref.Image] = ImageInfo{
					ImageType: ref.ImageType,
					Error:     ref.Error,
				}
			}
		}
	}

	return desc.bundle, nil
}

func getImageLayersInfo(image string) ([]Layers, error) {
	layers := []Layers{}
	parsedImgRef, err := regname.ParseReference(image, regname.WeakValidation)
	if err != nil {
		return nil, fmt.Errorf("Error: %s in parsing image %s", err.Error(), image)
	}

	v1Img, err := remote.Image(parsedImgRef, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return nil, fmt.Errorf("Error: %s in getting remote access of image %s", err.Error(), image)
	}

	imgLayers, err := v1Img.Layers()
	if err != nil {
		return nil, fmt.Errorf("Error: %s in getting layers of image %s", err.Error(), image)
	}

	for _, imgLayer := range imgLayers {
		digHash, err := imgLayer.Digest()
		if err != nil {
			return nil, fmt.Errorf("Error: %s in getting digest of layer's of image %s", err.Error(), image)
		}
		layers = append(layers, Layers{Digest: digHash.String()})
	}
	return layers, nil
}
