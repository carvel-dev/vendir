// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package openpgparmor

import (
	"crypto/fips140"
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

		// Parsing an OpenPGP key packet unconditionally computes its
		// RFC 4880 V4 fingerprint using SHA-1, which panics under
		// GODEBUG=fips140=only. That fingerprint is used only as a
		// key identifier for later signature lookups (see
		// fetch/git/verification.go).
		var el openpgp.EntityList
		var err error
		fips140.WithoutEnforcement(func() {
			r := strings.NewReader(startMarker + part)
			el, err = openpgp.ReadArmoredKeyRing(r)
		})
		if err != nil {
			return nil, fmt.Errorf("Reading armored key [idx=%d]: %s", i, err)
		}

		result = append(result, el...)
	}

	return result, nil
}
