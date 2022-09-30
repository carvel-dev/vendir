// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
)

// TagInfo Contains the tag name and the digest associated with the tag
// TagInfo.Digest might be empty if caller ask for it not to be retrieved
type TagInfo struct {
	Tag    string
	Digest string
}

// TagsInfo Contains all the tags associated with the repository on Image
type TagsInfo struct {
	Repository string
	Tags       []TagInfo
}

// TagList Retrieve all the tags associated with a repository
// imageRef contains the address for the repository
// getDigests when set to true, provides
func TagList(imageRef string, getDigests bool, registryOpts registry.Opts) (TagsInfo, error) {
	reg, err := registry.NewSimpleRegistry(registryOpts)
	if err != nil {
		return TagsInfo{}, err
	}

	ref, err := regname.ParseReference(imageRef, regname.WeakValidation)
	if err != nil {
		return TagsInfo{}, err
	}

	tags, err := reg.ListTags(ref.Context())
	if err != nil {
		return TagsInfo{}, err
	}

	tagList := TagsInfo{
		Repository: ref.Context().String(),
	}

	for _, tag := range tags {
		tagInfo := TagInfo{
			Tag: tag,
		}

		if getDigests {
			tagRef, err := regname.NewTag(ref.Context().String()+":"+tag, regname.WeakValidation)
			if err != nil {
				return TagsInfo{}, err
			}

			hash, err := reg.Digest(tagRef)
			if err != nil {
				return TagsInfo{}, err
			}

			tagInfo.Digest = hash.String()
		}
		tagList.Tags = append(tagList.Tags, tagInfo)
	}

	return tagList, nil
}
