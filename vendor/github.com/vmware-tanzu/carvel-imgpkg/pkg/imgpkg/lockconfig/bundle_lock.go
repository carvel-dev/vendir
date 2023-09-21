// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package lockconfig

import (
	"fmt"
	"os"

	regname "github.com/google/go-containerregistry/pkg/name"
	"sigs.k8s.io/yaml"
)

const (
	BundleLockKind       = "BundleLock"
	BundleLockAPIVersion = "imgpkg.carvel.dev/v1alpha1"
)

type BundleLock struct {
	LockVersion
	Bundle BundleRef `json:"bundle"` // This generated yaml, but due to lib we need to use `json`
}

type BundleRef struct {
	Image string `json:"image,omitempty"` // This generated yaml, but due to lib we need to use `json`
	Tag   string `json:"tag,omitempty"`   // This generated yaml, but due to lib we need to use `json`
}

func NewBundleLockFromPath(path string) (BundleLock, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return BundleLock{}, fmt.Errorf("Reading path %s: %s", path, err)
	}

	return NewBundleLockFromBytes(bs)
}

func NewBundleLockFromBytes(data []byte) (BundleLock, error) {
	var lock BundleLock

	err := yaml.UnmarshalStrict(data, &lock)
	if err != nil {
		return lock, fmt.Errorf("Unmarshaling bundle lock: %s", err)
	}

	err = lock.Validate()
	if err != nil {
		return lock, fmt.Errorf("Validating bundle lock: %s", err)
	}

	return lock, nil
}

func (b BundleLock) Validate() error {
	if b.APIVersion != BundleLockAPIVersion {
		return fmt.Errorf("Validating apiVersion: Unknown version (known: %s)", BundleLockAPIVersion)
	}
	if b.Kind != BundleLockKind {
		return fmt.Errorf("Validating kind: Unknown kind (known: %s)", BundleLockKind)
	}
	if _, err := regname.NewDigest(b.Bundle.Image); err != nil {
		return fmt.Errorf("Expected ref to be in digest form, got '%s'", b.Bundle.Image)
	}
	return nil
}

func (b BundleLock) AsBytes() ([]byte, error) {
	err := b.Validate()
	if err != nil {
		return nil, fmt.Errorf("Validating bundle lock: %s", err)
	}

	bs, err := yaml.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("Marshaling config: %s", err)
	}

	return []byte(fmt.Sprintf("---\n%s", bs)), nil
}

func (b BundleLock) WriteToPath(path string) error {
	bs, err := b.AsBytes()
	if err != nil {
		return err
	}

	err = os.WriteFile(path, bs, 0600)
	if err != nil {
		return fmt.Errorf("Writing bundle config: %s", err)
	}

	return nil
}
