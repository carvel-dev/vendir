// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package signature

import (
	"errors"
	"fmt"
	"sync"

	"carvel.dev/imgpkg/pkg/imgpkg/imageset"
	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	"carvel.dev/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/google/go-containerregistry/pkg/name"
	"golang.org/x/sync/errgroup"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . Finder
type Finder interface {
	Signature(reference name.Digest) (imageset.UnprocessedImageRef, error)
}

// FetchingError Error type that happen when fetching signatures
type FetchingError interface {
	error
	ImageRef() string
}

// NotFoundErr specific not found error
type NotFoundErr struct {
	imageRef string
}

// ImageRef Image Reference and associated to the error
func (n NotFoundErr) ImageRef() string {
	return n.imageRef
}

// Error Not Found Error message
func (n NotFoundErr) Error() string {
	return "signature not found"
}

// AccessDeniedErr specific access denied error
type AccessDeniedErr struct {
	imageRef string
}

// ImageRef Image Reference and associated to the error
func (a AccessDeniedErr) ImageRef() string {
	return a.imageRef
}

// Error Access Denied message
func (a AccessDeniedErr) Error() string {
	return "access denied"
}

// FetchError Struct that will contain all the errors found while fetching signatures
type FetchError struct {
	AllErrors []FetchingError
}

// Error message that contains all errors
func (f *FetchError) Error() string {
	msg := "Unable to retrieve the following images:"
	for _, err := range f.AllErrors {
		msg = fmt.Sprintf("%s\nImage: '%s'\nError: %s", msg, err.ImageRef(), err.Error())
	}
	return msg
}

// HasErrors check if any error happened
func (f *FetchError) HasErrors() bool {
	return len(f.AllErrors) > 0
}

// Add a new error to the list of errors
func (f *FetchError) Add(err FetchingError) {
	f.AllErrors = append(f.AllErrors, err)
}

// Signatures Signature fetcher
type Signatures struct {
	signatureFinder Finder
	concurrency     int
}

// NewSignatures constructs the Signature Fetcher
func NewSignatures(finder Finder, concurrency int) *Signatures {
	return &Signatures{
		signatureFinder: finder,
		concurrency:     concurrency,
	}
}

// Fetch Retrieve the available signatures associated with the images provided
func (s *Signatures) Fetch(images *imageset.UnprocessedImageRefs) (*imageset.UnprocessedImageRefs, error) {
	signatures := imageset.NewUnprocessedImageRefs()
	var imgs []lockconfig.ImageRef
	for _, ref := range images.All() {
		imgs = append(imgs, lockconfig.ImageRef{
			Image: ref.DigestRef,
		})
	}
	imagesRefs, err := s.FetchForImageRefs(imgs)
	if err != nil {
		var fetchError *FetchError
		if !errors.As(err, &fetchError) {
			return nil, err
		}

		for _, fError := range fetchError.AllErrors {
			var accessDeniedErr AccessDeniedErr
			if !errors.As(fError, &accessDeniedErr) {
				return nil, fetchError
			}
		}
	}
	for _, ref := range imagesRefs {
		signatures.Add(imageset.UnprocessedImageRef{
			DigestRef: ref.Image,
			Tag:       ref.Annotations["tag"],
		})
	}

	return signatures, nil
}

// FetchForImageRefs Retrieve the available signatures associated with the images provided
func (s *Signatures) FetchForImageRefs(images []lockconfig.ImageRef) ([]lockconfig.ImageRef, error) {
	lock := &sync.Mutex{}
	var signatures []lockconfig.ImageRef

	throttle := util.NewThrottle(s.concurrency)
	var wg errgroup.Group
	allErrs := &FetchError{}

	for _, ref := range images {
		ref := ref //copy
		wg.Go(func() error {
			imgDigest, err := name.NewDigest(ref.PrimaryLocation())
			if err != nil {
				return fmt.Errorf("Parsing '%s': %s", ref.Image, err)
			}

			throttle.Take()
			defer throttle.Done()

			signature, err := s.signatureFinder.Signature(imgDigest)
			if err != nil {
				if _, ok := err.(NotFoundErr); ok {
					return nil
				}
				if deniedErr, ok := err.(AccessDeniedErr); ok {
					lock.Lock()
					defer lock.Unlock()
					allErrs.Add(deniedErr)
					return nil
				}
				return fmt.Errorf("Fetching signature for image '%s': %s", imgDigest.Name(), err)
			}

			lock.Lock()
			signatures = append(signatures, lockconfig.ImageRef{
				Image:       signature.DigestRef,
				Annotations: map[string]string{"tag": signature.Tag},
			})
			lock.Unlock()
			return nil
		})
	}

	err := wg.Wait()

	if err != nil {
		return signatures, err
	}

	if allErrs.HasErrors() {
		return signatures, allErrs
	}

	return signatures, nil
}

// Noop No Operation signature fetcher
type Noop struct{}

// NewNoop Constructs a no operation signature fetcher
func NewNoop() *Noop { return &Noop{} }

// Fetch Do nothing
func (n Noop) Fetch(*imageset.UnprocessedImageRefs) (*imageset.UnprocessedImageRefs, error) {
	return imageset.NewUnprocessedImageRefs(), nil
}

// FetchForImageRefs Retrieve the available signatures associated with the images provided
func (n Noop) FetchForImageRefs(_ []lockconfig.ImageRef) ([]lockconfig.ImageRef, error) {
	return nil, nil
}
