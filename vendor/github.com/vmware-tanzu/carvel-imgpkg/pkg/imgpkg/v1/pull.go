// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"fmt"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/plainimage"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
)

// Logger Interface used for logging
type Logger interface {
	Errorf(msg string, args ...interface{})
	Warnf(msg string, args ...interface{})
	Debugf(msg string, args ...interface{})
	Tracef(msg string, args ...interface{})
	Logf(msg string, args ...interface{})
}

// PullOpts Option that can be provided to the pull request
type PullOpts struct {
	Logger Logger
	// AsImage Pull the contents of the OCI Image
	AsImage bool
	// IsBundle the image being pulled is a Bundle
	IsBundle bool
}

// ImagesLockInfo Information about the ImagesLock file
type ImagesLockInfo struct {
	Path    string `json:"path"`
	Updated bool   `json:"updated"`
}

// BundleInfo Information related to the specific bundle
type BundleInfo struct {
	ImageRef      string          `json:"image"`
	ImagesLock    *ImagesLockInfo `json:"imagesLock,omitempty"`
	NestedBundles []BundleInfo    `json:"nestedBundles,omitempty"`
}

// buildBundleInfoFromBundle Given a Bundle struct it can generate the BundleInfo associated to the nested bundles
// the updated key is used to indicate if the ImagesLock for the bundle was updated or not
func buildBundleInfoFromBundle(b *bundle.Bundle, updated bool) []BundleInfo {
	var nestedBundles []BundleInfo

	for _, nBundle := range b.NestedBundles() {
		nestedBundles = append(nestedBundles, newBundleInfo(nBundle, updated))
	}
	return nestedBundles
}

// newBundleInfo create a BundleInfo struct from bundle information
// the updated key is used to indicate if the ImagesLock for the bundle was updated or not
func newBundleInfo(node bundle.GraphNode, updated bool) BundleInfo {
	bInfo := BundleInfo{
		ImageRef: node.ImageRef,
		ImagesLock: &ImagesLockInfo{
			Path:    filepath.Join(node.Path, bundle.ImgpkgDir, bundle.ImagesLockFile),
			Updated: updated,
		},
		NestedBundles: nil,
	}
	for _, nestedBundle := range node.NestedBundles {
		bInfo.NestedBundles = append(bInfo.NestedBundles, newBundleInfo(nestedBundle, updated))
	}
	return bInfo
}

// Status Report from the Pull command
type Status struct {
	BundleInfo
	IsBundle  bool `json:"-"`
	Cacheable bool `json:"cacheable"`
}

// Pull Download the contents of the image referenced by imageRef to the folder outputPath
func Pull(imageRef string, outputPath string, pullOptions PullOpts, registryOpts registry.Opts) (Status, error) {
	reg, err := registry.NewSimpleRegistry(registryOpts)
	if err != nil {
		return Status{}, err
	}

	bundleToPull := bundle.NewBundle(imageRef, reg)
	isBundle, err := bundleToPull.IsBundle()
	if err != nil {
		return Status{}, err
	}

	switch {
	case isBundle && pullOptions.AsImage: // Trying to pull the OCI Image of a Bundle
		st, err := pullImage(imageRef, outputPath, pullOptions, reg)
		if err != nil {
			return Status{}, err
		}
		st.IsBundle = true
		return st, nil

	case isBundle && pullOptions.IsBundle: // Trying to pull a Bundle
		return pullBundle(imageRef, bundleToPull, outputPath, pullOptions, false)

	case !isBundle && pullOptions.IsBundle: // Trying to pull an Image as a Bundle
		return Status{}, &ErrIsNotBundle{}

	case !isBundle && !pullOptions.IsBundle: // Trying to pull an OCI Image
		return pullImage(imageRef, outputPath, pullOptions, reg)

	case isBundle && !pullOptions.IsBundle: // Trying to pull a Bundle as if it where an OCI Image
		return Status{}, &ErrIsBundle{}
	}

	return Status{}, fmt.Errorf("Unknown option")
}

// PullRecursive Downloads the contents of the Bundle and Nested Bundles referenced by imageRef to the folder outputPath.
// This functions should error out when imageRef does not point to a Bundle
func PullRecursive(imageRef string, outputPath string, pullOptions PullOpts, registryOpts registry.Opts) (Status, error) {
	reg, err := registry.NewSimpleRegistry(registryOpts)
	if err != nil {
		return Status{}, err
	}

	bundleToPull := bundle.NewBundle(imageRef, reg)
	isBundle, err := bundleToPull.IsBundle()
	if err != nil {
		return Status{}, err
	}
	if !isBundle {
		return Status{}, &ErrIsNotBundle{}
	}

	return pullBundle(imageRef, bundleToPull, outputPath, pullOptions, true)
}

// pullBundle Downloads the contents of the Bundle Image referenced by imageRef to the folder outputPath.
// This functions should error out when imageRef does not point to a Bundle
func pullBundle(imgRef string, bundleToPull *bundle.Bundle, outputPath string, pullOptions PullOpts, pullNestedBundles bool) (Status, error) {
	isRootBundleRelocated, err := bundleToPull.Pull(outputPath, pullOptions.Logger, pullNestedBundles)
	if err != nil {
		return Status{}, err
	}

	isCacheable, err := isCacheable(imgRef, isRootBundleRelocated)
	if err != nil {
		return Status{}, err
	}

	bInfo := buildBundleInfoFromBundle(bundleToPull, isRootBundleRelocated)
	return Status{
		BundleInfo: BundleInfo{
			ImageRef: bundleToPull.DigestRef(),
			ImagesLock: &ImagesLockInfo{
				Path:    filepath.Join(outputPath, bundle.ImgpkgDir, bundle.ImagesLockFile),
				Updated: isRootBundleRelocated,
			},
			NestedBundles: bInfo,
		},
		Cacheable: isCacheable,
		IsBundle:  true,
	}, nil
}

func pullImage(imageRef string, outputPath string, pullOptions PullOpts, reg registry.Registry) (Status, error) {
	plainImg := plainimage.NewPlainImage(imageRef, reg)
	isImage, err := plainImg.IsImage()
	if err != nil {
		return Status{}, err
	}
	if !isImage {
		return Status{}, fmt.Errorf("Unable to pull non-images, such as image indexes. (hint: provide a specific digest to the image instead)")
	}

	err = plainImg.Pull(outputPath, pullOptions.Logger)
	if err != nil {
		return Status{}, err
	}
	isCacheable, err := isCacheable(imageRef, true)
	if err != nil {
		return Status{}, err
	}
	return Status{
		BundleInfo: BundleInfo{
			ImageRef: plainImg.DigestRef(),
		},
		Cacheable: isCacheable,
		IsBundle:  false,
	}, nil
}

func isCacheable(imageRef string, isRootBundleRelocated bool) (bool, error) {
	_, err := name.NewDigest(imageRef)
	if err != nil {
		if _, ok := err.(*name.ErrBadName); !ok {
			return false, fmt.Errorf("Unexpected error checking for digest: %+v", err)
		}

		return false, nil
	}
	if isRootBundleRelocated {
		return true, nil
	}

	return false, nil
}
