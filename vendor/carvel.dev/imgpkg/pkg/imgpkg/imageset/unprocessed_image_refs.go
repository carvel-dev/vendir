// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imageset

import (
	"fmt"
	"sort"
	"sync"

	regname "github.com/google/go-containerregistry/pkg/name"
)

type UnprocessedImageRef struct {
	DigestRef string
	Tag       string
	Labels    map[string]string
	OrigRef   string
}

func (u UnprocessedImageRef) Key() string {
	return u.DigestRef + ":" + u.Tag
}

type UnprocessedImageRefs struct {
	imgRefs map[string]UnprocessedImageRef

	lock sync.Mutex
}

func NewUnprocessedImageRefs() *UnprocessedImageRefs {
	return &UnprocessedImageRefs{imgRefs: map[string]UnprocessedImageRef{}}
}

func (i *UnprocessedImageRefs) Add(imgRef UnprocessedImageRef) {
	imgRef.Validate()

	i.lock.Lock()
	defer i.lock.Unlock()
	i.imgRefs[imgRef.Key()] = imgRef
}

func (i *UnprocessedImageRefs) Length() int {
	return len(i.imgRefs)
}

func (i *UnprocessedImageRefs) All() []UnprocessedImageRef {
	i.lock.Lock()
	defer i.lock.Unlock()

	var result []UnprocessedImageRef
	for _, imgRef := range i.imgRefs {
		result = append(result, imgRef)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].DigestRef < result[j].DigestRef
	})
	return result
}

func (u UnprocessedImageRef) Validate() {
	_, err := regname.NewDigest(u.DigestRef)
	if err != nil {
		panic(fmt.Sprintf("Digest need to be provided: %s", err))
	}
}
