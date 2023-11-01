// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"testing"

	. "carvel.dev/vendir/pkg/vendir/config"
	"github.com/stretchr/testify/require"
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

func TestSecretToRegistryAuthSecretsWithAuthFieldFallback(t *testing.T) {
	t.Run("password is non-empty and auth is invalid (ignored)", func(t *testing.T) {
		s1 := Secret{
			Type: "kubernetes.io/dockerconfigjson",
			Data: map[string][]byte{
				".dockerconfigjson": []byte(`{"auths":{"registry.io":{"username":"user","password":"pass", "auth":"invalid"}, "foo":{}}}`),
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
	})

	t.Run("password is empty and auth is invalid base64 (errors)", func(t *testing.T) {
		s1 := Secret{
			Type: "kubernetes.io/dockerconfigjson",
			Data: map[string][]byte{
				".dockerconfigjson": []byte(`{"auths":{"registry.io":{"username":"user","password":"", "auth":"invalid"}, "foo":{}}}`),
			},
		}

		_, err := s1.ToRegistryAuthSecrets()
		require.EqualError(t, err, "Decoding auth field: illegal base64 data at input byte 4")
	})

	t.Run("password is empty and auth is invalid due to missing password (errors)", func(t *testing.T) {
		s1 := Secret{
			Type: "kubernetes.io/dockerconfigjson",
			Data: map[string][]byte{
				".dockerconfigjson": []byte(`{"auths":{"registry.io":{"username":"user","password":"", "auth":"Zm9v"}, "foo":{}}}`),
			},
		}

		_, err := s1.ToRegistryAuthSecrets()
		require.EqualError(t, err, "Expected auth field to have 'username:password' format, but did not")
	})

	t.Run("password is empty, falls back on auth field username+password", func(t *testing.T) {
		s1 := Secret{
			Type: "kubernetes.io/dockerconfigjson",
			Data: map[string][]byte{
				".dockerconfigjson": []byte(`{"auths":{
					"registry.io":{"username":"user-not","password":"", "auth":"dXNlcjpwYXNzd29yZA=="},
					"foo":{},
					"foo2":{"auth":"dXNlcjE6cGFzczpwYXNz"}
				}}`),
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
				"hostname": []byte("foo2"),
				"username": []byte("user1"),
				"password": []byte("pass:pass"),
			},
		}, {
			Data: map[string][]byte{
				"hostname": []byte("registry.io"),
				"username": []byte("user"),
				"password": []byte("password"),
			},
		}}, result)
	})
}
