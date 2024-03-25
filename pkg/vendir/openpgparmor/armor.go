// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package openpgparmor

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/openpgp" //nolint:staticcheck
)

func ReadArmoredKeys(keys string) (openpgp.EntityList, error) {
	const startMarker = "-----BEGIN "

	parts := strings.Split(keys, startMarker)
	if len(parts) == 1 {
		return nil, fmt.Errorf("Expected to find armored block, but did not")
	}

	var result openpgp.EntityList

	for i, part := range parts {
		if len(part) == 0 {
			continue
		}

		el, err := openpgp.ReadArmoredKeyRing(strings.NewReader(startMarker + part))
		if err != nil {
			return nil, fmt.Errorf("Reading armored key [idx=%d]: %s", i, err)
		}

		result = append(result, el...)
	}

	return result, nil
}
