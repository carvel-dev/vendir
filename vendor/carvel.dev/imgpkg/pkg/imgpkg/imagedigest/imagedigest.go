// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

// Package imagedigest utility packages to abstract the retrieval of Digests
package imagedigest

import (
	regname "github.com/google/go-containerregistry/pkg/name"
)

// DigestWrap holds regname.Digest and orig reference
// retrieved from ImagesLock's image field
type DigestWrap struct {
	regnameDigest regname.Digest
	origRef       string
}

// DigestWrap sets regnameDigest and origRef fields' values
func (dw *DigestWrap) DigestWrap(imgIdxRef string, origRef string) error {
	regnameDigest, err := regname.NewDigest(imgIdxRef)
	if err != nil {
		return err
	}
	dw.regnameDigest = regnameDigest
	dw.origRef = origRef

	return nil
}

// RegnameDigest returns regnameDigest value of
// DigestWrap instance
func (dw *DigestWrap) RegnameDigest() regname.Digest {
	return dw.regnameDigest
}

// OrigRef returns origRef value of
// DigestWrap instance
func (dw *DigestWrap) OrigRef() string {
	return dw.origRef
}
