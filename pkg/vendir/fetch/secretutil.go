// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package fetch

import (
	"strings"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
)

// GetSecretField retrieves a secret field with case-insensitive key matching
func GetSecretField(secret *ctlconf.Secret, key string) ([]byte, bool) {
	// First try exact match
	if value, ok := secret.Data[key]; ok {
		return value, true
	}

	// Then try case-insensitive match
	keyLower := strings.ToLower(key)
	for k, v := range secret.Data {
		if strings.ToLower(k) == keyLower {
			return v, true
		}
	}

	return nil, false
}

// NormalizeSecretData returns a new map with all keys normalized to lowercase
func NormalizeSecretData(secret *ctlconf.Secret) map[string][]byte {
	normalized := make(map[string][]byte)
	for k, v := range secret.Data {
		normalized[strings.ToLower(k)] = v
	}
	return normalized
}
