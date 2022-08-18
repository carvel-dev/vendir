// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"

	regauthn "github.com/google/go-containerregistry/pkg/authn"
	credentialprovider "github.com/vdemeester/k8s-pkg-credentialprovider"
)

var _ regauthn.Keychain = &EnvKeychain{}

type envKeychainInfo struct {
	URL           string
	Username      string
	Password      string
	IdentityToken string
	RegistryToken string
}

// EnvKeychain implements an authn.Keychain interface by using credentials provided by imgpkg's auth environment vars
type EnvKeychain struct {
	environFunc func() []string

	infos       []envKeychainInfo
	collectErr  error
	collected   bool
	collectLock sync.Mutex
}

// NewEnvKeychain builder for Environment Keychain
func NewEnvKeychain(environFunc func() []string) *EnvKeychain {
	if environFunc == nil {
		environFunc = os.Environ
	}

	return &EnvKeychain{
		environFunc: environFunc,
	}
}

// Resolve looks up the most appropriate credential for the specified target.
func (k *EnvKeychain) Resolve(target regauthn.Resource) (regauthn.Authenticator, error) {
	infos, err := k.collect()
	if err != nil {
		return nil, err
	}

	for _, info := range infos {
		registryURLMatches, err := credentialprovider.URLsMatchStr(info.URL, target.String())
		if err != nil {
			return nil, err
		}

		if registryURLMatches {
			return regauthn.FromConfig(regauthn.AuthConfig{
				Username:      info.Username,
				Password:      info.Password,
				IdentityToken: info.IdentityToken,
				RegistryToken: info.RegistryToken,
			}), nil
		}
	}

	return regauthn.Anonymous, nil
}

type orderedEnvKeychainInfos []envKeychainInfo

func (s orderedEnvKeychainInfos) Len() int {
	return len(s)
}

func (s orderedEnvKeychainInfos) Less(i, j int) bool {
	return s[i].URL < s[j].URL
}

func (s orderedEnvKeychainInfos) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (k *EnvKeychain) collect() ([]envKeychainInfo, error) {
	k.collectLock.Lock()
	defer k.collectLock.Unlock()

	if k.collected {
		return append([]envKeychainInfo{}, k.infos...), nil
	}
	if k.collectErr != nil {
		return nil, k.collectErr
	}

	const (
		globalEnvironPrefix = "IMGPKG_REGISTRY_"
		sep                 = "_"
	)

	funcsMap := map[string]func(*envKeychainInfo, string) error{
		"HOSTNAME": func(info *envKeychainInfo, val string) error {
			if !strings.HasPrefix(val, "https://") && !strings.HasPrefix(val, "http://") {
				val = "https://" + val
			}
			parsedURL, err := url.Parse(val)
			if err != nil {
				return fmt.Errorf("Parsing registry hostname: %s (e.g. gcr.io, index.docker.io)", err)
			}

			// Allows exact matches:
			//    foo.bar.com/namespace
			// Or hostname matches:
			//    foo.bar.com
			// It also considers /v2/  and /v1/ equivalent to the hostname
			effectivePath := parsedURL.Path
			if strings.HasPrefix(effectivePath, "/v2/") || strings.HasPrefix(effectivePath, "/v1/") {
				effectivePath = effectivePath[3:]
			}
			var key string
			if (len(effectivePath) > 0) && (effectivePath != "/") {
				key = parsedURL.Host + effectivePath
			} else {
				key = parsedURL.Host
			}
			info.URL = key
			return nil
		},
		"USERNAME": func(info *envKeychainInfo, val string) error {
			info.Username = val
			return nil
		},
		"PASSWORD": func(info *envKeychainInfo, val string) error {
			info.Password = val
			return nil
		},
		"IDENTITY_TOKEN": func(info *envKeychainInfo, val string) error {
			info.IdentityToken = val
			return nil
		},
		"REGISTRY_TOKEN": func(info *envKeychainInfo, val string) error {
			info.RegistryToken = val
			return nil
		},
	}

	defaultInfo := envKeychainInfo{}
	infos := map[string]envKeychainInfo{}

	for _, env := range k.environFunc() {
		pieces := strings.SplitN(env, "=", 2)
		if len(pieces) != 2 {
			continue
		}

		if !strings.HasPrefix(pieces[0], globalEnvironPrefix) || pieces[0] == "IMGPKG_REGISTRY_AZURE_CR_CONFIG" {
			continue
		}

		var matched bool

		for key, updateFunc := range funcsMap {
			switch {
			case pieces[0] == globalEnvironPrefix+key:
				matched = true
				err := updateFunc(&defaultInfo, pieces[1])
				if err != nil {
					k.collectErr = err
					return nil, k.collectErr
				}
			case strings.HasPrefix(pieces[0], globalEnvironPrefix+key+sep):
				matched = true
				suffix := strings.TrimPrefix(pieces[0], globalEnvironPrefix+key+sep)
				info := infos[suffix]
				err := updateFunc(&info, pieces[1])
				if err != nil {
					k.collectErr = err
					return nil, k.collectErr
				}
				infos[suffix] = info
			}
		}
		if !matched {
			k.collectErr = fmt.Errorf("Unknown env variable '%s'", pieces[0])
			return nil, k.collectErr
		}
	}

	var result []envKeychainInfo

	if defaultInfo != (envKeychainInfo{}) {
		result = append(result, defaultInfo)
	}
	for _, info := range infos {
		result = append(result, info)
	}

	// Update the collected auth infos used to identify which credentials to use for a given
	// image. The info is reverse-sorted by URL so more specific paths are matched
	// first. For example, if for the given image "quay.io/coreos/etcd",
	// credentials for "quay.io/coreos" should match before "quay.io".
	sort.Sort(sort.Reverse(orderedEnvKeychainInfos(result)))

	k.infos = result
	k.collected = true

	return append([]envKeychainInfo{}, k.infos...), nil
}
