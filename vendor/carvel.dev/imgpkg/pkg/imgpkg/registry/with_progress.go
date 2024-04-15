// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"

	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

var _ Registry = &WithProgress{}

func NewRegistryWithProgress(reg Registry, logger util.ProgressLogger) *WithProgress {
	return &WithProgress{delegate: reg, logger: logger}
}

// WithProgress Implements Registry interface and provides progress updates to the logger
type WithProgress struct {
	delegate Registry
	logger   util.ProgressLogger
}

// Get Retrieve Image descriptor for an Image reference
func (w *WithProgress) Get(reference regname.Reference) (*remote.Descriptor, error) {
	return w.delegate.Get(reference)
}

// Digest Retrieve the Digest for an Image reference
func (w *WithProgress) Digest(reference regname.Reference) (regv1.Hash, error) {
	return w.delegate.Digest(reference)
}

// Index Retrieve regv1.ImageIndex struct for an Index reference
func (w *WithProgress) Index(reference regname.Reference) (regv1.ImageIndex, error) {
	return w.delegate.Index(reference)
}

// Image Retrieve the regv1.Image struct for an Image reference
func (w *WithProgress) Image(reference regname.Reference) (regv1.Image, error) {
	return w.delegate.Image(reference)
}

// FirstImageExists Returns the first of the provided Image Digests that exists in the Registry
func (w *WithProgress) FirstImageExists(digests []string) (string, error) {
	return w.delegate.FirstImageExists(digests)
}

// MultiWrite Upload multiple Images in Parallel to the Registry
func (w *WithProgress) MultiWrite(imageOrIndexesToUpload map[regname.Reference]remote.Taggable, concurrency int, _ chan regv1.Update) error {
	uploadProgress := make(chan regv1.Update)
	w.logger.Start(context.Background(), uploadProgress)
	defer w.logger.End()

	return w.delegate.MultiWrite(imageOrIndexesToUpload, concurrency, uploadProgress)
}

// WriteImage Upload Image to registry
func (w *WithProgress) WriteImage(reference regname.Reference, image regv1.Image, _ chan regv1.Update) error {
	uploadProgress := make(chan regv1.Update)
	w.logger.Start(context.Background(), uploadProgress)
	defer w.logger.End()

	return w.delegate.WriteImage(reference, image, uploadProgress)
}

// WriteIndex Uploads the Index manifest to the registry
func (w *WithProgress) WriteIndex(reference regname.Reference, index regv1.ImageIndex) error {
	return w.delegate.WriteIndex(reference, index)
}

// WriteTag Tag the referenced Image
func (w *WithProgress) WriteTag(tag regname.Tag, taggable remote.Taggable) error {
	return w.delegate.WriteTag(tag, taggable)
}

// ListTags Retrieve all tags associated with a Repository
func (w *WithProgress) ListTags(repo regname.Repository) ([]string, error) {
	return w.delegate.ListTags(repo)
}

// CloneWithSingleAuth Clones the provided registry replacing the Keychain with a Keychain that can only authenticate
// the image provided
// A Registry need to be provided as the first parameter or the function will panic
func (w WithProgress) CloneWithSingleAuth(imageRef regname.Tag) (Registry, error) {
	delegate, err := w.delegate.CloneWithSingleAuth(imageRef)
	if err != nil {
		return nil, err
	}

	return &WithProgress{delegate: delegate}, nil
}

// CloneWithLogger Clones the provided registry updating the progress
func (w WithProgress) CloneWithLogger(logger util.ProgressLogger) Registry {
	return &WithProgress{delegate: w.delegate, logger: logger}
}
