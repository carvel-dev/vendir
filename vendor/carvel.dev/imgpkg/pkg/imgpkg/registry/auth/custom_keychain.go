// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

// Package auth provides different keychains used in imgpkg
package auth

import (
	"fmt"
	"strings"
	"time"

	regauthn "github.com/google/go-containerregistry/pkg/authn"
)

var _ regauthn.Keychain = CustomRegistryKeychain{}

// IAASKeychain defines the type IAAS Keychain names
type IAASKeychain string

var (
	// GKEKeychain GKE keychain name
	GKEKeychain IAASKeychain = "gke"
	// AKSKeychain AKS keychain name
	AKSKeychain IAASKeychain = "aks"
	// ECRKeychain ECR keychain name
	ECRKeychain IAASKeychain = "ecr"
	// GithubKeychain Github keychain name
	GithubKeychain IAASKeychain = "github"
)

// KeychainOpts Contains credentials (passed down via flags) used by custom keychain to auth with a registry
type KeychainOpts struct {
	Username                string
	Password                string
	Token                   string
	Anon                    bool
	EnableIaasAuthProviders bool
	ActiveKeychains         []IAASKeychain
}

// NewSingleAuthKeychain Builds a SingleAuthKeychain struct
func NewSingleAuthKeychain(auth regauthn.Authenticator) SingleAuthKeychain {
	return SingleAuthKeychain{auth: auth}
}

// SingleAuthKeychain This Keychain will always provide the same authentication for all images
type SingleAuthKeychain struct {
	auth regauthn.Authenticator
}

// Resolve returns the same authentication for all images
func (s SingleAuthKeychain) Resolve(_ regauthn.Resource) (regauthn.Authenticator, error) {
	return s.auth, nil
}

// CustomRegistryKeychain implements an authn.Keychain interface by using credentials provided by imgpkg's auth options
type CustomRegistryKeychain struct {
	Opts KeychainOpts
}

// Resolve looks up the most appropriate credential for the specified target.
func (k CustomRegistryKeychain) Resolve(res regauthn.Resource) (regauthn.Authenticator, error) {
	switch {
	case len(k.Opts.Username) > 0:
		return &regauthn.Basic{Username: k.Opts.Username, Password: k.Opts.Password}, nil
	case len(k.Opts.Token) > 0:
		return &regauthn.Bearer{Token: k.Opts.Token}, nil
	case k.Opts.Anon:
		return regauthn.Anonymous, nil
	default:
		return k.retryDefaultKeychain(func() (regauthn.Authenticator, error) {
			return regauthn.DefaultKeychain.Resolve(res)
		})
	}
}

func (k CustomRegistryKeychain) retryDefaultKeychain(doFunc func() (regauthn.Authenticator, error)) (regauthn.Authenticator, error) {
	// constants copied from https://github.com/carvel-dev/imgpkg/blob/c8b1bc196e5f1af82e6df8c36c290940169aa896/vendor/github.com/docker/docker-credential-helpers/credentials/error.go#L4-L11

	// ErrCredentialsNotFound standardizes the not found error, so every helper returns
	// the same message and docker can handle it properly.
	const errCredentialsNotFoundMessage = "credentials not found in native keychain"
	// ErrCredentialsMissingServerURL and ErrCredentialsMissingUsername standardize
	// invalid credentials or credentials management operations
	const errCredentialsMissingServerURLMessage = "no credentials server URL"
	const errCredentialsMissingUsernameMessage = "no credentials username"

	var auth regauthn.Authenticator
	var lastErr error

	for i := 0; i < 5; i++ {
		auth, lastErr = doFunc()
		if lastErr == nil {
			return auth, nil
		}

		if strings.Contains(lastErr.Error(), errCredentialsNotFoundMessage) || strings.Contains(lastErr.Error(), errCredentialsMissingUsernameMessage) || strings.Contains(lastErr.Error(), errCredentialsMissingServerURLMessage) {
			return auth, lastErr
		}

		time.Sleep(2 * time.Second)
	}
	return auth, fmt.Errorf("Retried 5 times: %s", lastErr)
}
