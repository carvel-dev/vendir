// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package lockconfig

import (
	"fmt"
	"io/ioutil"

	regname "github.com/google/go-containerregistry/pkg/name"
	"sigs.k8s.io/yaml"
)

const (
	ImagesLockKind       = "ImagesLock"
	ImagesLockAPIVersion = "imgpkg.carvel.dev/v1alpha1"
)

type ImagesLock struct {
	LockVersion
	Images []ImageRef `json:"images,omitempty"` // This generated yaml, but due to lib we need to use `json`
}

type ImageRef struct {
	Image       string            `json:"image,omitempty"`       // This generated yaml, but due to lib we need to use `json`
	Annotations map[string]string `json:"annotations,omitempty"` // This generated yaml, but due to lib we need to use `json`
	locations   []string
}

func NewEmptyImagesLock() ImagesLock {
	return ImagesLock{
		LockVersion: LockVersion{
			APIVersion: ImagesLockAPIVersion,
			Kind:       ImagesLockKind,
		},
	}
}

func NewImagesLockFromPath(path string) (ImagesLock, error) {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return ImagesLock{}, fmt.Errorf("Reading path %s: %s", path, err)
	}

	return NewImagesLockFromBytes(bs)
}

func NewImagesLockFromBytes(data []byte) (ImagesLock, error) {
	var lock ImagesLock

	err := yaml.UnmarshalStrict(data, &lock)
	if err != nil {
		return lock, fmt.Errorf("Unmarshaling images lock: %s", err)
	}

	err = lock.Validate()
	if err != nil {
		return lock, fmt.Errorf("Validating images lock: %s", err)
	}

	// Update the image lock file to use a fully qualified name
	// i.e. if a user provides ubuntu (short hand for library/ubuntu) in the ImageLock file,
	// downstream processing will fail when comparing if images match.
	for i, img := range lock.Images {
		parsedImageRefName, err := regname.NewDigest(img.Image)
		if err != nil {
			panic(fmt.Sprintf("Image reference (%s) is in an invalid format: %s", img.Image, err.Error()))
		}
		lock.Images[i].Image = parsedImageRefName.Name()
	}

	return lock, nil
}

func (i *ImagesLock) AddImageRef(ref ImageRef) {
	for _, image := range i.Images {
		if image.Image == ref.Image {
			return
		}
	}
	i.Images = append(i.Images, ref)
}

func (i ImagesLock) Validate() error {
	if i.APIVersion != ImagesLockAPIVersion {
		return fmt.Errorf("Validating apiVersion: Unknown version (known: %s)", ImagesLockAPIVersion)
	}
	if i.Kind != ImagesLockKind {
		return fmt.Errorf("Validating kind: Unknown kind (known: %s)", ImagesLockKind)
	}
	for _, imageRef := range i.Images {
		if _, err := regname.NewDigest(imageRef.Image); err != nil {
			return fmt.Errorf("Expected ref to be in digest form, got '%s'", imageRef.Image)
		}
	}
	return nil
}

func (i ImagesLock) AsBytes() ([]byte, error) {
	err := i.Validate()
	if err != nil {
		return nil, fmt.Errorf("Validating images lock: %s", err)
	}

	// Use the first location instead of the value present in Image
	var imgRefs []ImageRef
	for _, image := range i.Images {
		image.Image = image.PrimaryLocation()
		imgRefs = append(imgRefs, image)
	}
	updatedImagesLock := i
	updatedImagesLock.Images = imgRefs

	bs, err := yaml.Marshal(updatedImagesLock)
	if err != nil {
		return nil, fmt.Errorf("Marshaling config: %s", err)
	}

	return []byte(fmt.Sprintf("---\n%s", bs)), nil
}

func (i ImagesLock) WriteToPath(path string) error {
	bs, err := i.AsBytes()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, bs, 0600)
	if err != nil {
		return fmt.Errorf("Writing images config: %s", err)
	}

	return nil
}

func (i ImageRef) DeepCopy() ImageRef {
	annotations := map[string]string{}
	for key, val := range i.Annotations {
		annotations[key] = val
	}

	return ImageRef{
		Image:       i.Image,
		locations:   append([]string{}, i.locations...),
		Annotations: annotations,
	}
}

func (i ImageRef) Locations() []string {
	if i.locations == nil {
		return []string{i.Image}
	}

	locations := append([]string{}, i.locations...)
	locations = append(locations, i.Image)
	return locations
}

func (i *ImageRef) AddLocation(location string) {
	if location == i.Image {
		return
	}

	for _, m := range i.locations {
		if m == location {
			return
		}
	}
	i.locations = append([]string{location}, i.locations...)
}

func (i *ImageRef) PrimaryLocation() string {
	return i.Locations()[0]
}

func (i *ImageRef) DiscardLocationsExcept(viableLocation string) ImageRef {
	imgRef := i.DeepCopy()
	if viableLocation == imgRef.Image {
		imgRef.locations = []string{}
		return imgRef
	}

	imgRef.locations = []string{viableLocation}
	return imgRef
}
