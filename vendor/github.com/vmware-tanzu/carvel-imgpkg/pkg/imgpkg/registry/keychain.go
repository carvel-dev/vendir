// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"fmt"
	"io"

	"github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
	"github.com/chrismellard/docker-credential-acr-env/pkg/credhelper"
	regauthn "github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/github"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry/auth"
)

// Keychain implements an authn.Keychain interface by composing multiple keychains.
// It enforces an order, where the keychains that contain credentials for a specific target take precedence over
// keychains that contain credentials for 'any' target. i.e. env keychain takes precedence over the custom keychain.
// Since env keychain contains credentials per HOSTNAME, and custom keychain doesn't.
func Keychain(keychainOpts auth.KeychainOpts, environFunc func() []string) (regauthn.Keychain, error) {
	// env keychain comes first
	keychain := []regauthn.Keychain{auth.NewEnvKeychain(environFunc)}

	if keychainOpts.EnableIaasAuthProviders {
		// if enabled, fall back to iaas keychains
		keychain = append(keychain,
			google.Keychain,
			regauthn.NewKeychainFromHelper(ecr.NewECRHelper(ecr.WithLogger(io.Discard))),
			regauthn.NewKeychainFromHelper(credhelper.NewACRCredentialsHelper()),
			github.Keychain,
		)
	} else {
		for _, activeKeychain := range keychainOpts.ActiveKeychains {
			var k regauthn.Keychain
			switch activeKeychain {
			case auth.GKEKeychain:
				k = google.Keychain
			case auth.ECRKeychain:
				k = regauthn.NewKeychainFromHelper(ecr.NewECRHelper(ecr.WithLogger(io.Discard)))
			case auth.AKSKeychain:
				k = regauthn.NewKeychainFromHelper(credhelper.NewACRCredentialsHelper())
			case auth.GithubKeychain:
				k = github.Keychain
			default:
				return nil, fmt.Errorf("Unable to load keychain for %s, available keychains [aks, ecr, gke, github]]", string(activeKeychain))
			}
			keychain = append(keychain, k)
		}
	}

	// command-line flags and docker keychain comes last
	keychain = append(keychain, auth.CustomRegistryKeychain{Opts: keychainOpts})

	return regauthn.NewMultiKeychain(keychain...), nil
}
