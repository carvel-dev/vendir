// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package lockconfig

import (
	"fmt"
)

type LockVersion struct {
	APIVersion string `json:"apiVersion"` // This generated yaml, but due to lib we need to use `json`
	Kind       string `json:"kind"`       // This generated yaml, but due to lib we need to use `json`
}

func NewLockFromPath(path string) (*BundleLock, *ImagesLock, error) {
	bundleLock, err := NewBundleLockFromPath(path)
	if err == nil {
		return &bundleLock, nil, nil
	}
	imagesLock, err := NewImagesLockFromPath(path)
	if err == nil {
		return nil, &imagesLock, nil
	}
	return nil, nil, fmt.Errorf("Trying to read bundle or images lock file: %s", err)
}
