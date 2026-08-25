// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package openpgparmor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// testPublicKey is a throwaway test-only key, not used anywhere else.
const testPublicKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----

mQENBGp63p8BCADcYVU8PAXvQlprliTU7+bBSuqc+xBwdkGXokWkFUxD8bmBktID
+GF7b3PKO2rXL/Swo+8VGXn32GDQTP4+mk5W9FZZjbZalPGCebUbUdWPqnqM1U2X
FsrM3nG2utTWEWdNnIZJdhgcX1OP2u4nBhjk4evOtGVaXl4Q7LuIuNzGXiUJVi2e
BITXDwk2BM31ZinOw2p7Qi/GfLER1SxXF7TOb779YmlrP6OjXKPx8+qlvJXQLpT3
3iaanbXtLWzwsOpjp88J2wBtKNaR48c/cUsFKRtnqmvXBAj6aJjoofZfNeVl1hBe
HSs0fyOSJHNp5OxMoVtLcdG4wDmjEGWWKsFJABEBAAG0M3ZlbmRpciBGSVBTIHRl
c3Qga2V5IDx2ZW5kaXItZmlwcy10ZXN0QGV4YW1wbGUuY29tPokBUgQTAQgAPBYh
BANAH2vdaiRUD6bLJylneNRUfu0pBQJqet6fAxsvBAULCQgHAgIiAgYVCgkICwIE
FgIDAQIeBwIXgAAKCRApZ3jUVH7tKU0vB/9kCTI+WBmsnWyUFcxA4ep8cuK7jwNj
fJS21ahvSSCzixjznYke7hY2oFcF7Da/bz6SKGzpKY0JaIZLZKwnXD5+BF3zQnzk
UZehE4Q4QFLlURPP/P6iad2BmldoGUQpOUMh+MiTBehfFn25J0te93WSt0tH0KLc
XX+hicHBSK+f6QkWO/dLXQsjY/3COSoFmwjP5GwZQzidjtT7MZOXrip+fR3obPeC
xGgyp1Kb3Bndgg7XlUPMJGf+k4lG6SrvxZ1WtMfe1fJ6dETUtADFaVfjmq844avD
Ac7hpC5RL9wPo77I9B3iRcBXX4+WK0b3+XsEiTHtUr4ZrnaUpWaF4d3k
=76gJ
-----END PGP PUBLIC KEY BLOCK-----
`

// TestReadArmoredKeys_UnderFIPS140Only guards against regressing on
// crypto/fips140.WithoutEnforcement in ReadArmoredKeys: parsing an OpenPGP
// key computes its RFC 4880 SHA-1 fingerprint internally, which panics under
// GODEBUG=fips140=only if not wrapped. Run this test's binary with
// GODEBUG=fips140=only (as a native-FIPS vendir build does by default) to
// exercise that path.
func TestReadArmoredKeys_UnderFIPS140Only(t *testing.T) {
	keys, err := ReadArmoredKeys(testPublicKey)
	require.NoError(t, err)
	require.Len(t, keys, 1)
}
