// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package imagetar

import (
	regv1 "github.com/google/go-containerregistry/pkg/v1"
)

type ImageLayerWriterFilter struct {
	includeNonDistributable bool
}

func NewImageLayerWriterCheck(includeNonDistributable bool) ImageLayerWriterFilter {
	return ImageLayerWriterFilter{includeNonDistributable}
}

func (f ImageLayerWriterFilter) ShouldLayerBeIncluded(layer regv1.Layer) (bool, error) {
	mediaType, err := layer.MediaType()
	if err != nil {
		return false, err
	}
	return mediaType.IsDistributable() || f.includeNonDistributable, nil
}
