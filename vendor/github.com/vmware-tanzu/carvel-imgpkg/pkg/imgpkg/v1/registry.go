// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"strings"

	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry/auth"
)

// OptsFromEnv Using the base Opts fills up the missing information using the environment variables
func OptsFromEnv(base registry.Opts, readEnv func(string) (string, bool)) registry.Opts {
	opts := base.DeepCopy()

	if len(opts.Username) == 0 {
		opts.Username, _ = readEnv("IMGPKG_USERNAME")
	}
	if len(opts.Password) == 0 {
		opts.Password, _ = readEnv("IMGPKG_PASSWORD")
	}
	if len(opts.Token) == 0 {
		opts.Token, _ = readEnv("IMGPKG_TOKEN")
	}

	if anon, _ := readEnv("IMGPKG_ANON"); anon == "true" {
		opts.Anon = true
	}
	iaasAuth, found := readEnv("IMGPKG_ENABLE_IAAS_AUTH")
	if found && strings.ToLower(iaasAuth) == "true" {
		opts.EnableIaasAuthProviders = true
	}

	keychains, found := readEnv("IMGPKG_ACTIVE_KEYCHAINS")
	if found {
		if len(keychains) > 0 {
			if strings.Contains(keychains, ",") {
				for _, keychainName := range strings.Split(keychains, ",") {
					opts.ActiveKeychains = append(opts.ActiveKeychains, auth.IAASKeychain(strings.TrimSpace(keychainName)))
				}
			} else {
				opts.ActiveKeychains = append(opts.ActiveKeychains, auth.IAASKeychain(strings.TrimSpace(keychains)))
			}
		}
	}

	return opts
}
