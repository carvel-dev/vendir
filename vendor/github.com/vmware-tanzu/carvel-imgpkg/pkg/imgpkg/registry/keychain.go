// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"

	regauthn "github.com/google/go-containerregistry/pkg/authn"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry/auth"
)

// Keychain implements an authn.Keychain interface by composing multiple keychains.
// It enforces an order, where the keychains that contain credentials for a specific target take precedence over
// keychains that contain credentials for 'any' target. i.e. env keychain takes precedence over the custom keychain.
// Since env keychain contains credentials per HOSTNAME, and custom keychain doesn't.
func Keychain(keychainOpts auth.KeychainOpts, environFunc func() []string) (regauthn.Keychain, error) {
	iaasKeychain, err := auth.NewIaasKeychain(context.Background(), environFunc)
	if err != nil {
		return nil, err
	}

	return regauthn.NewMultiKeychain(auth.NewEnvKeychain(environFunc), iaasKeychain, auth.CustomRegistryKeychain{Opts: keychainOpts}), nil
}
