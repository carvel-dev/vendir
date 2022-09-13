// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagedesc

import (
	regv1 "github.com/google/go-containerregistry/pkg/v1"
)

type DescribedReader struct {
	ids           *ImageRefDescriptors
	layerProvider LayerProvider
}

func NewDescribedReader(ids *ImageRefDescriptors, layerProvider LayerProvider) DescribedReader {
	return DescribedReader{ids, layerProvider}
}

func (r DescribedReader) Read() []ImageOrIndex {
	var result []ImageOrIndex

	for _, td := range r.ids.Descriptors() {
		switch {
		case td.Image != nil:
			var img ImageWithRef = NewDescribedImage(*td.Image, r.layerProvider)
			result = append(result, ImageOrIndex{Image: &img, Labels: td.Image.Labels, OrigRef: td.OrigRef()})

		case td.ImageIndex != nil:
			idx := r.buildIndex(*td.ImageIndex)
			result = append(result, ImageOrIndex{Index: &idx, Labels: td.ImageIndex.Labels, OrigRef: td.OrigRef()})

		default:
			panic("Unknown item")
		}
	}

	return result
}

func (r DescribedReader) buildIndex(iitd ImageIndexDescriptor) ImageIndexWithRef {
	var images []regv1.Image
	var indexes []regv1.ImageIndex

	for _, imgTD := range iitd.Images {
		images = append(images, NewDescribedImage(imgTD, r.layerProvider))
	}
	for _, indexTD := range iitd.Indexes {
		indexes = append(indexes, r.buildIndex(indexTD))
	}

	return NewDescribedImageIndex(iitd, images, indexes)
}
