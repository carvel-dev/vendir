// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"fmt"
	"io/ioutil"
	"sort"

	"sigs.k8s.io/yaml"
)

const (
	LocationFilepath   = "image-locations.yml"
	ImageLocationsKind = "ImageLocations"
	LocationAPIVersion = "imgpkg.carvel.dev/v1alpha1"
)

type ImageLocationsConfig struct {
	APIVersion string          `json:"apiVersion"` // This generated yaml, but due to lib we need to use `json`
	Kind       string          `json:"kind"`       // This generated yaml, but due to lib we need to use `json`
	Images     []ImageLocation `json:"images"`     // This generated yaml, but due to lib we need to use `json`
}

type ImageLocation struct {
	Image    string `json:"image"`    // This generated yaml, but due to lib we need to use `json`
	IsBundle bool   `json:"isBundle"` // This generated yaml, but due to lib we need to use `json`
}

func NewLocationConfigFromPath(path string) (ImageLocationsConfig, error) {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return ImageLocationsConfig{}, fmt.Errorf("Reading path %s: %s", path, err)
	}

	return NewLocationConfigFromBytes(bs)
}

func NewLocationConfigFromBytes(data []byte) (ImageLocationsConfig, error) {
	var lock ImageLocationsConfig

	err := yaml.Unmarshal(data, &lock)
	if err != nil {
		return lock, fmt.Errorf("Unmarshaling image locations config: %s", err)
	}

	err = lock.Validate()
	if err != nil {
		return ImageLocationsConfig{}, err
	}

	return lock, nil
}

func (c ImageLocationsConfig) AsBytes() ([]byte, error) {
	err := c.Validate()
	if err != nil {
		return nil, err
	}

	// Ensure images are sorted deterministically based on content
	// within this structure so that it's not influenced by outside
	// factors (such as source image locations)
	sort.Slice(c.Images, func(i, j int) bool {
		return c.Images[i].Image < c.Images[j].Image
	})

	bs, err := yaml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("Marshaling image locations config: %s", err)
	}

	return []byte(fmt.Sprintf("---\n%s", bs)), nil
}

func (c ImageLocationsConfig) WriteToPath(path string) error {
	bs, err := c.AsBytes()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, bs, 0600)
	if err != nil {
		return fmt.Errorf("Writing image locations config: %s", err)
	}

	return nil
}

func (c ImageLocationsConfig) Validate() error {
	if c.APIVersion != LocationAPIVersion {
		return fmt.Errorf("Validating apiVersion: Unknown version (known: %s)", LocationAPIVersion)
	}

	if c.Kind != ImageLocationsKind {
		return fmt.Errorf("Validating kind: Unknown kind (known: %s)", ImageLocationsKind)
	}

	return nil
}
