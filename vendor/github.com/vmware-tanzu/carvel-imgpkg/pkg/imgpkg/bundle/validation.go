// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	plainimg "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/plainimage"
)

type notABundleError struct {
}

func (n notABundleError) Error() string {
	return "Not a Bundle"
}

func IsNotBundleError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(notABundleError)
	return ok
}

func (o *Bundle) IsBundle() (bool, error) {
	img, err := o.plainImg.Fetch()
	if err != nil {
		if plainimg.IsNotAnImageError(err) {
			return false, nil
		}
		return false, err
	}

	if img == nil {
		return false, nil
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return false, err
	}
	_, present := cfg.Config.Labels[BundleConfigLabel]
	return present, nil
}
