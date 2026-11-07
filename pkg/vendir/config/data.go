// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package config

const (
	SecretK8sCorev1BasicAuthUsernameKey = "username"
	SecretK8sCorev1BasicAuthPasswordKey = "password"
	SecretK8sCorev1HTTPBearerTokenKey   = "token"

	SecretK8sCoreV1SSHAuthPrivateKey = "ssh-privatekey"
	SecretSSHAuthKnownHosts          = "ssh-knownhosts" // not part of k8s

	SecretRegistryHostnameKey   = "hostname"
	SecretRegistryBearerToken   = "token"
	SecretRegistryIdentityToken = "identity-token"

	SecretGithubAPIToken = "token"
)

// There structs have minimal used set of fields from their K8s representations.

type GenericMetadata struct {
	Name string
}

type Secret struct {
	APIVersion string
	Kind       string

	Metadata GenericMetadata
	Type     string
	Data     map[string][]byte
	// StringData holds the same values as Data, unencoded. Kubernetes accepts
	// either, so a secret written by hand usually uses this one.
	StringData map[string]string
}

// FoldStringData moves any StringData entries into Data, so that everything
// reading a secret only has to look in one place. Kubernetes gives StringData
// precedence when a key appears in both, so do the same.
func (s *Secret) FoldStringData() {
	if len(s.StringData) == 0 {
		return
	}

	if s.Data == nil {
		s.Data = map[string][]byte{}
	}
	for k, v := range s.StringData {
		s.Data[k] = []byte(v)
	}
	s.StringData = nil
}

//nolint:revive
type ConfigMap struct {
	APIVersion string
	Kind       string

	Metadata GenericMetadata
	Data     map[string]string
}
