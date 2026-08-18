// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"bytes"
	"crypto"
	"strings"
	"testing"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/openpgp/packet" //nolint:staticcheck
)

// Fixed armored detached signatures over the literal text "hello world\n",
// pre-generated with a throwaway GPG key so these tests don't need gpg on
// PATH. approvedSig is RSA+SHA256 (FIPS-approved); legacyHashSig is the
// same RSA key forced to sign with SHA1 (non-approved hash); legacyDSASig
// is a 1024-bit DSA key (non-approved pubkey algorithm).
const (
	approvedSig = `-----BEGIN PGP SIGNATURE-----

iQEzBAABCAAdFiEEQLsnJrIjJBKz+S0Go2HoPiOrblcFAmqH8IQACgkQo2HoPiOr
bleXvQgAgZpJZpuaL3ku/7Oxm/OXF7g0cLwCgbpVzqieSDQ7b/SNWo/NWhPt76V8
YH9fDXgt6yxT4ukqOvNW4ESnzLU9hRNy7Me1h83L/njhLyOqWoUaewBhcuTWTE13
uCF2zSi2O/oCnZ8ZjwHnf6rIR9082mpI4o2zJVTmsw+L0p56fm2vM2QPMjNLNXzC
5CjSgwLC8j+ilklW21wGJVeM+IIC35b8Dl/SqTzpvLKnREwxNdJz8CZnuYzS9KFa
5OSaTmLFq6dy2DVRRBVPe+dfEj8E07MCNvPCAXOVgLnmqpq4gvpZ06UaHt6FfD9u
P/jc8MSTszFZiBk2ipIuuSN4NilYaA==
=sZsk
-----END PGP SIGNATURE-----
`
	legacyHashSig = `-----BEGIN PGP SIGNATURE-----

iQEzBAABAgAdFiEEQLsnJrIjJBKz+S0Go2HoPiOrblcFAmqH8IQACgkQo2HoPiOr
blcLdgf+OSp4P/JC9jG4x8BolhaEJOxGC0pFPbjoRex9zMzyspQzVos+oF2ARq3M
J0SWRSOd1IrDYvgCEstUJvlpWr97P6zJW830NQgr+0Wp7MN3SOkL1sjp1Ax79GmF
DyJ6UFGAslwSBWS9LnpgfOF1lhClExKf2KMMrTwpDpP2qEl/rGV3LP7nRNXBlP0s
6AJdtEoJrUsJqYavqVT8i78juiWSD8JVFtspz15dyZ9dDA9JhTS1QjgBOGJ5sEhj
fTsbePrtmk6TgjEFrs6xt2L4LVZHt8ypyy/H3OpBOY6+/eqr4Yy60DZhV3JRZrHE
qXOThcpTEGCWYakFGjHn0mGpPPXekg==
=S6/W
-----END PGP SIGNATURE-----
`
	legacyDSASig = `-----BEGIN PGP SIGNATURE-----

iF0EABECAB0WIQSNpoTJgEs/HmLRkJ5eLmKg5upf1wUCaofwhAAKCRBeLmKg5upf
10mxAKCoI6PhKxq7bNM9q/6e5kIcN/+RggCaAgMZE9LTYajEEvt4oPjfFxKbVec=
=oCds
-----END PGP SIGNATURE-----
`
)

func TestSignatureAlgorithms(t *testing.T) {
	hash, algo, err := signatureAlgorithms(strings.NewReader(approvedSig))
	require.NoError(t, err)
	require.Equal(t, crypto.SHA256, hash)
	require.Equal(t, packet.PubKeyAlgoRSA, algo)

	hash, algo, err = signatureAlgorithms(strings.NewReader(legacyHashSig))
	require.NoError(t, err)
	require.Equal(t, crypto.SHA1, hash)
	require.Equal(t, packet.PubKeyAlgoRSA, algo)

	hash, algo, err = signatureAlgorithms(strings.NewReader(legacyDSASig))
	require.NoError(t, err)
	require.Equal(t, crypto.SHA1, hash)
	require.Equal(t, packet.PubKeyAlgoDSA, algo)
}

func TestCheckFIPSApproved(t *testing.T) {
	t.Run("approved signature never errors or warns", func(t *testing.T) {
		var buf bytes.Buffer
		v := Verification{infoLog: &buf}
		require.NoError(t, v.checkFIPSApproved("ref", approvedSig))
		require.Empty(t, buf.String())
	})

	t.Run("legacy hash errors by default", func(t *testing.T) {
		v := Verification{infoLog: &bytes.Buffer{}}
		err := v.checkFIPSApproved("myref", legacyHashSig)
		require.ErrorContains(t, err, "myref")
		require.ErrorContains(t, err, "hash algorithm SHA-1")
		require.ErrorContains(t, err, "verification.allowLegacySignatures")
	})

	t.Run("legacy DSA errors by default", func(t *testing.T) {
		v := Verification{infoLog: &bytes.Buffer{}}
		err := v.checkFIPSApproved("myref", legacyDSASig)
		require.ErrorContains(t, err, "pubkey algorithm")
		require.ErrorContains(t, err, "verification.allowLegacySignatures")
	})

	t.Run("legacy signatures only warn when allowed", func(t *testing.T) {
		var buf bytes.Buffer
		opts := ctlconf.DirectoryContentsGitVerification{AllowLegacySignatures: true}
		v := Verification{infoLog: &buf, opts: opts}

		require.NoError(t, v.checkFIPSApproved("myref", legacyHashSig))
		require.Contains(t, buf.String(), "myref")
		require.Contains(t, buf.String(), "hash algorithm SHA-1")

		buf.Reset()
		require.NoError(t, v.checkFIPSApproved("myref", legacyDSASig))
		require.Contains(t, buf.String(), "pubkey algorithm")
	})
}
