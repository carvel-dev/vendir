// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"sort"
)

type secretDockerConfigJSON struct {
	Auths map[string]secretDockerConfigJSONAuth
}

type secretDockerConfigJSONAuth struct {
	Username string
	Password string
}

// ToRegistryAuthSecrets splits secret into multiple secrets
// if secret of type dockerconfigjson; otherwise returns same secret.
func (s Secret) ToRegistryAuthSecrets() ([]Secret, error) {
	const (
		// Constants from Kubernetes core v1
		typeDockerConfigJSON = "kubernetes.io/dockerconfigjson"
		dockerConfigJSONKey  = ".dockerconfigjson"
	)

	if s.Type != typeDockerConfigJSON {
		return []Secret{s}, nil // return itself
	}

	var data secretDockerConfigJSON

	err := json.Unmarshal(s.Data[dockerConfigJSONKey], &data)
	if err != nil {
		return nil, err
	}

	var secrets []Secret

	// Sort hostnames so that secrets always come out in deterministic order
	var hostnames []string
	for hostname := range data.Auths {
		hostnames = append(hostnames, hostname)
	}
	sort.Strings(hostnames)

	for _, hostname := range hostnames {
		auth, found := data.Auths[hostname]
		if !found {
			panic("Internal inconsistency: hostname missing")
		}

		secrets = append(secrets, Secret{
			Metadata: s.Metadata,
			// Careful adding new keys here, since consumers of these secrets
			// might be returning errors for any unexpected keys found
			Data: map[string][]byte{
				SecretRegistryHostnameKey:           []byte(hostname),
				SecretK8sCorev1BasicAuthUsernameKey: []byte(auth.Username),
				SecretK8sCorev1BasicAuthPasswordKey: []byte(auth.Password),
			},
		})
	}

	return secrets, nil
}
