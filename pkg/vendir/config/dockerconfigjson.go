// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"fmt"
)

type secretDockerConfigJSON struct {
	Auths map[string]secretDockerConfigJSONAuth
}

type secretDockerConfigJSONAuth struct {
	Username string
	Password string
}

func (s Secret) ToBasicAuthSecret() (Secret, error) {
	const (
		// Constants from Kubernetes core v1
		typeDockerConfigJSON = "kubernetes.io/dockerconfigjson"
		dockerConfigJSONKey  = ".dockerconfigjson"
	)

	if s.Type != typeDockerConfigJSON {
		return s, nil // return itself
	}

	var data secretDockerConfigJSON

	err := json.Unmarshal(s.Data[dockerConfigJSONKey], &data)
	if err != nil {
		return Secret{}, err
	}

	if len(data.Auths) != 1 {
		return Secret{}, fmt.Errorf("Expected exactly one registry configuration within a secret")
	}

	for _, auth := range data.Auths {
		return Secret{
			Data: map[string][]byte{
				SecretK8sCorev1BasicAuthUsernameKey: []byte(auth.Username),
				SecretK8sCorev1BasicAuthPasswordKey: []byte(auth.Password),
			},
		}, nil
	}

	panic("Unreachable")
}
