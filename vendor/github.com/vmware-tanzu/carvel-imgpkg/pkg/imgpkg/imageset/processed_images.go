// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imageset

import (
	"fmt"
	"sort"
	"sync"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
)

type ProcessedImage struct {
	UnprocessedImageRef
	DigestRef string

	Image      regv1.Image
	ImageIndex regv1.ImageIndex
}

func (p ProcessedImage) Key() string {
	return p.UnprocessedImageRef.Key()
}

func (p ProcessedImage) Validate() {
	_, err := regname.NewDigest(p.DigestRef)
	if err != nil {
		panic(fmt.Sprintf("Digest need to be provided: %s", err))
	}

	if p.Image == nil && p.ImageIndex == nil {
		panic("Either Image or ImageIndex must be provided")
	}

	p.UnprocessedImageRef.Validate()
}

type ProcessedImages struct {
	imgs     map[string]ProcessedImage
	imgsLock sync.Mutex
}

func NewProcessedImages() *ProcessedImages {
	return &ProcessedImages{imgs: map[string]ProcessedImage{}}
}

func (i *ProcessedImages) Add(img ProcessedImage) {
	i.imgsLock.Lock()
	defer i.imgsLock.Unlock()

	img.Validate()

	i.imgs[img.UnprocessedImageRef.Key()] = img
}

func (i *ProcessedImages) FindByURL(unprocessedImageURL UnprocessedImageRef) (ProcessedImage, bool) {
	i.imgsLock.Lock()
	defer i.imgsLock.Unlock()

	img, found := i.imgs[unprocessedImageURL.Key()]
	return img, found
}

// Len returns the length of Processed Images
func (i *ProcessedImages) Len() int {
	i.imgsLock.Lock()
	defer i.imgsLock.Unlock()

	return len(i.imgs)
}

func (i *ProcessedImages) All() []ProcessedImage {
	i.imgsLock.Lock()
	defer i.imgsLock.Unlock()

	var result []ProcessedImage
	for _, img := range i.imgs {
		result = append(result, img)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UnprocessedImageRef.DigestRef < result[j].UnprocessedImageRef.DigestRef
	})
	return result
}
