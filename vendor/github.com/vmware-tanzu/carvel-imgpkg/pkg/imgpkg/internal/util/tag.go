// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"strings"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imagedigest"
)

// WithDigest are items that Digest() can be called on
type WithDigest interface {
	Digest() (regv1.Hash, error)
}

// TagGenDigest contains Algorithm and Hex values of image digest
type TagGenDigest struct {
	Algorithm string
	Hex       string
}

// Digest returns regv1.Hash instance
func (t TagGenDigest) Digest() (regv1.Hash, error) {
	return regv1.Hash{
		Algorithm: t.Algorithm,
		Hex:       t.Hex,
	}, nil
}

// TagGenerator interface
type TagGenerator interface {
	GenerateTag(item imagedigest.DigestWrap, destinationRepo regname.Repository) (regname.Tag, error)
}

// DefaultTagGenerator implements GenerateTag
// and generates default tag
type DefaultTagGenerator struct{}

// RepoBasedTagGenerator implements GenerateTag
// and generates repo-based tag
type RepoBasedTagGenerator struct{}

// GenerateTag generates default tag
func (tagGen DefaultTagGenerator) GenerateTag(item imagedigest.DigestWrap, importRepo regname.Repository) (regname.Tag, error) {
	digestArr := strings.Split(item.RegnameDigest().DigestStr(), ":")

	withDigest := TagGenDigest{
		Algorithm: digestArr[0],
		Hex:       digestArr[1],
	}
	return BuildDefaultUploadTagRef(withDigest, importRepo)
}

// GenerateTag generates repo-based tags
func (tagGen RepoBasedTagGenerator) GenerateTag(item imagedigest.DigestWrap, importRepo regname.Repository) (regname.Tag, error) {
	origRepoPath := ""
	if item.OrigRef() == "" {
		origRepoPath = strings.Split(item.RegnameDigest().Name(), "@")[0]
	} else {
		origRepoPath = strings.Split(item.OrigRef(), "@")[0]
	}

	origRepoPath = strings.Join(strings.Split(origRepoPath, "/")[1:], "-")
	digestArr := strings.Split(item.RegnameDigest().DigestStr(), ":")
	tagStartIdx := len(origRepoPath) - 49
	if tagStartIdx < 0 {
		tagStartIdx = 0
	}

	dashedRepo := fmt.Sprintf("%s-%s-%s.imgpkg", origRepoPath[tagStartIdx:], digestArr[0], digestArr[1])
	// if tag starts with a "-", PUT to /v2/<repo>/manifests/-<foo>
	// will give an "un-recognized request" error
	if strings.HasPrefix(dashedRepo, "-") {
		dashedRepo = strings.Replace(dashedRepo, "-", "", 1)
	}
	tag := strings.ReplaceAll(dashedRepo, ":", "-")
	uploadTagRef, err := regname.NewTag(fmt.Sprintf("%s:%s", importRepo.Name(), tag))
	if err != nil {
		return regname.Tag{}, fmt.Errorf("building repo-based tag: %s", err)
	}
	return uploadTagRef, nil
}

// BuildDefaultUploadTagRef Builds a tag from the digest Algorithm and Digest
func BuildDefaultUploadTagRef(item WithDigest, importRepo regname.Repository) (regname.Tag, error) {
	digest, err := item.Digest()
	if err != nil {
		return regname.Tag{}, err
	}

	tag := fmt.Sprintf("%s-%s.imgpkg", digest.Algorithm, digest.Hex)
	uploadTagRef, err := regname.NewTag(fmt.Sprintf("%s:%s", importRepo.Name(), tag))
	if err != nil {
		return regname.Tag{}, fmt.Errorf("building default upload tag image ref: %s", err)
	}
	return uploadTagRef, nil
}
