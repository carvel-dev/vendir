// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	. "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
)

func TestSecretToRegistryAuthSecretsNoop(t *testing.T) {
	s1 := Secret{Data: map[string][]byte{"test": []byte("abc")}}

	result, err := s1.ToRegistryAuthSecrets()
	require.NoError(t, err)

	require.Equal(t, []Secret{s1}, result, "Expected same secret to be returned")
}

func TestSecretToRegistryAuthSecretsWithDockerConfigJsonZeroRegistries(t *testing.T) {
	s1 := Secret{
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string][]byte{
			".dockerconfigjson": []byte(`{"auths":{}}`),
		},
	}

	result, err := s1.ToRegistryAuthSecrets()
	require.NoError(t, err)

	require.Len(t, result, 0)
}

func TestSecretToRegistryAuthSecretsWithDockerConfigJson(t *testing.T) {
	s1 := Secret{
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string][]byte{
			".dockerconfigjson": []byte(`{"auths":{"registry.io":{"username":"user","password":"pass"}}}`),
		},
	}

	result, err := s1.ToRegistryAuthSecrets()
	require.NoError(t, err)

	require.Equal(t, []Secret{{
		Data: map[string][]byte{
			"hostname": []byte("registry.io"),
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}}, result)
}

func TestSecretToRegistryAuthSecretsWithDockerConfigJsonMultipleRegistries(t *testing.T) {
	s1 := Secret{
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string][]byte{
			".dockerconfigjson": []byte(`{"auths":{"registry.io":{"username":"user","password":"pass"}, "foo":{}}}`),
		},
	}

	result, err := s1.ToRegistryAuthSecrets()
	require.NoError(t, err)

	require.Equal(t, []Secret{{
		Data: map[string][]byte{
			"hostname": []byte("foo"),
			"username": []byte(""),
			"password": []byte(""),
		},
	}, {
		Data: map[string][]byte{
			"hostname": []byte("registry.io"),
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}}, result)
}
