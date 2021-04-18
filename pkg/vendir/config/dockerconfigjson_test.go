// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"reflect"
	"strings"
	"testing"

	. "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
)

func TestSecretToBasicAuthSecretNoop(t *testing.T) {
	s1 := Secret{Data: map[string][]byte{"test": []byte("abc")}}

	result, err := s1.ToBasicAuthSecret()
	if err != nil {
		t.Fatalf("Expected nil err, but was: %s", err)
	}

	if !reflect.DeepEqual(s1, result) {
		t.Fatalf("Expected same secret to be returned")
	}
}

func TestSecretToBasicAuthSecretWithDockerConfigJson(t *testing.T) {
	s1 := Secret{
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string][]byte{
			".dockerconfigjson": []byte(`{"auths":{"registry.io":{"username":"user","password":"pass"}}}`),
		},
	}

	expectedS := Secret{
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}

	result, err := s1.ToBasicAuthSecret()
	if err != nil {
		t.Fatalf("Expected nil err, but was: %s", err)
	}

	if !reflect.DeepEqual(result, expectedS) {
		t.Fatalf("Expected secret to be turned into username/password secret")
	}
}

func TestSecretToBasicAuthSecretWithDockerConfigJsonTooManyRegistries(t *testing.T) {
	s1 := Secret{
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string][]byte{
			".dockerconfigjson": []byte(`{"auths":{"registry.io":{"username":"user","password":"pass"}, "foo":{}}}`),
		},
	}

	_, err := s1.ToBasicAuthSecret()
	if err == nil {
		t.Fatalf("Expected non-nil err")
	}

	if !strings.Contains(err.Error(), "Expected exactly one registry configuration within a secret") {
		t.Fatalf("Expected one registry configuration error, but found err: %s", err)
	}
}
