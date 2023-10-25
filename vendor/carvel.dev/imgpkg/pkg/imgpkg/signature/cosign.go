// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package signature

import (
	"fmt"
	"net/http"

	"carvel.dev/imgpkg/pkg/imgpkg/imageset"
	"carvel.dev/imgpkg/pkg/imgpkg/signature/cosign"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

// DigestReader Interface that knows how to read a Digest from a registry
type DigestReader interface {
	Digest(reference regname.Reference) (regv1.Hash, error)
}

// Cosign Signature retriever
type Cosign struct {
	registry DigestReader
}

// NewCosign constructor for Signature retriever
func NewCosign(reg DigestReader) *Cosign {
	return &Cosign{registry: reg}
}

// Signature retrieves the Image information that contains the signature for the provided Image
func (c Cosign) Signature(imageRef regname.Digest) (imageset.UnprocessedImageRef, error) {
	sigTagRef, err := c.signatureTag(imageRef)
	if err != nil {
		return imageset.UnprocessedImageRef{}, err
	}

	sigDigest, err := c.registry.Digest(sigTagRef)
	if err != nil {
		if transportErr, ok := err.(*transport.Error); ok {
			if transportErr.StatusCode == http.StatusNotFound {
				return imageset.UnprocessedImageRef{}, NotFoundErr{}
			}
			if transportErr.StatusCode == http.StatusForbidden {
				return imageset.UnprocessedImageRef{}, AccessDeniedErr{imageRef: sigTagRef.String()}
			}
		}
		return imageset.UnprocessedImageRef{}, err
	}

	return imageset.UnprocessedImageRef{
		DigestRef: imageRef.Digest(sigDigest.String()).Name(),
		Tag:       sigTagRef.TagStr(),
	}, nil
}

func (c Cosign) signatureTag(reference regname.Digest) (regname.Tag, error) {
	digest, err := regv1.NewHash(reference.DigestStr())
	if err != nil {
		return regname.Tag{}, fmt.Errorf("Converting to hash: %s", err)
	}
	return regname.NewTag(reference.Repository.Name() + ":" + cosign.Munge(regv1.Descriptor{Digest: digest}))
}
