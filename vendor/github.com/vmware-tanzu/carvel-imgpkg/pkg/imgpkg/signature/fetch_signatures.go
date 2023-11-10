// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package signature

import (
	"fmt"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imageset"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
	"golang.org/x/sync/errgroup"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . Finder
type Finder interface {
	Signature(reference name.Digest) (imageset.UnprocessedImageRef, error)
}

// NotFoundErr specific not found error
type NotFoundErr struct{}

// Error Not Found Error message
func (s NotFoundErr) Error() string {
	return "signature not found"
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
		return nil, err
	}
	for _, ref := range imagesRefs {
		signatures.Add(imageset.UnprocessedImageRef{
			DigestRef: ref.Image,
			Tag:       ref.Annotations["tag"],
		})
	}

	return signatures, err
}

// FetchForImageRefs Retrieve the available signatures associated with the images provided
func (s *Signatures) FetchForImageRefs(images []lockconfig.ImageRef) ([]lockconfig.ImageRef, error) {
	lock := &sync.Mutex{}
	var signatures []lockconfig.ImageRef

	throttle := util.NewThrottle(s.concurrency)
	var wg errgroup.Group

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
				if _, ok := err.(NotFoundErr); !ok {
					return fmt.Errorf("Fetching signature for image '%s': %s", imgDigest.Name(), err)
				}
				return nil
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

	return signatures, err
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
