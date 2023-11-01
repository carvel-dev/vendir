// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package plainimage

import (
	"fmt"

	ctlimg "carvel.dev/imgpkg/pkg/imgpkg/image"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type ImagesDescriptor interface {
	Get(regname.Reference) (*regremote.Descriptor, error)
}

// Logger logs information
type Logger interface {
	Logf(string, ...interface{})
}

// PlainImage struct that represents an OCI Image
type PlainImage struct {
	imagesDescriptor ImagesDescriptor

	unparsedRef  string
	parsedRef    regname.Reference
	parsedDigest string

	fetchedImage regv1.Image
}

// NewPlainImage creates the struct that represents the OCI Image referenced by ref
func NewPlainImage(ref string, imgDescriptor ImagesDescriptor) *PlainImage {
	return &PlainImage{unparsedRef: ref, imagesDescriptor: imgDescriptor}
}

// NewFetchedPlainImageWithTag creates the struct that represents the OCI Image reference by the fetchedImage
// This function should only be used after an initial retrieval of information from registry
func NewFetchedPlainImageWithTag(digestRef string, tag string, fetchedImage regv1.Image) *PlainImage {
	if fetchedImage == nil {
		panic("Expected a pre-fetched image")
	}

	parsedDigestRef, err := regname.NewDigest(digestRef)
	if err != nil {
		panic(fmt.Sprintf("Expected valid Digest Ref: %s", err))
	}

	var parsedRef regname.Reference
	if tag == "" {
		parsedRef = parsedDigestRef
	} else {
		parsedRef, err = regname.NewTag(parsedDigestRef.Context().Name() + ":" + tag)
		if err != nil {
			panic(fmt.Sprintf("Expected valid Tag Ref: %s", err))
		}
	}

	return &PlainImage{
		parsedRef:    parsedRef,
		parsedDigest: parsedDigestRef.DigestStr(),
		fetchedImage: fetchedImage,
	}
}

// Repo Repository where the image is stored
func (i *PlainImage) Repo() string {
	if i.parsedRef == nil {
		panic("Unexpected usage of Repo(); call Fetch before")
	}
	return i.parsedRef.Context().Name()
}

// DigestRef Image full location including registry, repository and digest
func (i *PlainImage) DigestRef() string {
	if i.parsedRef == nil {
		panic("Unexpected usage of DigestRef(); call Fetch before")
	}
	if len(i.parsedDigest) == 0 {
		panic("Unexpected usage of DigestRef(); call Fetch before")
	}
	return i.parsedRef.Context().Name() + "@" + i.parsedDigest
}

// Digest Image Digest
func (i *PlainImage) Digest() string {
	if i.parsedRef == nil {
		panic("Unexpected usage of Digest(); call Fetch before")
	}
	if len(i.parsedDigest) == 0 {
		panic("Unexpected usage of Digest(); call Fetch before")
	}
	return i.parsedDigest
}

// Tag of the image or "" if the image is referenced by digest
func (i *PlainImage) Tag() string {
	if i.parsedRef == nil {
		panic("Unexpected usage of Tag(); call Fetch before")
	}
	if tagRef, ok := i.parsedRef.(regname.Tag); ok {
		return tagRef.TagStr()
	}
	return "" // was a digest ref, so no tag
}

// Fetch the information about the referenced image
func (i *PlainImage) Fetch() (regv1.Image, error) {
	var err error
	if i.fetchedImage != nil {
		return i.fetchedImage, nil
	}

	i.parsedRef, err = regname.ParseReference(i.unparsedRef, regname.WeakValidation)
	if err != nil {
		return nil, err
	}

	imgDescriptor, err := i.imagesDescriptor.Get(i.parsedRef)
	if err != nil {
		return nil, fmt.Errorf("Fetching image: %s", err)
	}

	if !imgDescriptor.MediaType.IsImage() {
		i.parsedDigest = imgDescriptor.Digest.String()
		return nil, notAnImageError{imgDescriptor.MediaType}
	}

	i.fetchedImage, err = imgDescriptor.Image()
	if err != nil {
		return nil, fmt.Errorf("Fetching image: %s", err)
	}

	digest, err := i.fetchedImage.Digest()
	if err != nil {
		return nil, fmt.Errorf("Getting image digest: %s", err)
	}

	i.parsedDigest = digest.String()

	return i.fetchedImage, nil
}

// IsImage checks if the provided reference is an OCI Image
func (i *PlainImage) IsImage() (bool, error) {
	img, err := i.Fetch()
	if img == nil && err == nil {
		panic("Unreachable code")
	}

	if err != nil {
		if IsNotAnImageError(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// Pull the OCI Image to disk
func (i *PlainImage) Pull(outputPath string, logger Logger) error {
	img, err := i.Fetch()
	if err != nil {
		return err
	}

	if img == nil {
		panic("Not supported Pull on pre fetched PlainImage")
	}

	logger.Logf("Pulling image '%s'\n", i.DigestRef())

	err = ctlimg.NewDirImage(outputPath, img, logger).AsDirectory()
	if err != nil {
		return fmt.Errorf("Extracting image into directory: %s", err)
	}

	return nil
}

func IsNotAnImageError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(notAnImageError)
	return ok
}

type notAnImageError struct {
	mediaType types.MediaType
}

func (n notAnImageError) Error() string {
	return fmt.Sprintf("Expected an Image but got: %s", n.mediaType)
}
