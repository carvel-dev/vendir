// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
	oarmor "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/openpgparmor"
	"golang.org/x/crypto/openpgp" //nolint:staticcheck
)

// Verification verifies Git commit/tag against a set of public keys
// (Git details: https://github.com/git/git/blob/e2850a27a95c6f5b141dd88398b1702d2e524a81/Documentation/technical/hash-function-transition.txt#L386-L414)
type Verification struct {
	repoPath   string
	opts       ctlconf.DirectoryContentsGitVerification
	refFetcher ctlfetch.RefFetcher
}

func (v Verification) Verify(ref string) error {
	secret, err := v.refFetcher.GetSecret(v.opts.PublicKeysSecretRef.Name)
	if err != nil {
		return err
	}

	publicKeysStr := ""

	for _, val := range secret.Data {
		publicKeysStr += string(val) + "\n"
	}

	publicKeys, err := oarmor.ReadArmoredKeys(publicKeysStr)
	if err != nil {
		return fmt.Errorf("Reading armored key ring: %s", err)
	}

	if len(publicKeys) == 0 {
		return fmt.Errorf("Expected at least one public key, but found 0")
	}

	signedObj, err := v.readObject(ref)
	if err != nil {
		return err
	}

	target := strings.NewReader(signedObj.Contents)
	sig := strings.NewReader(signedObj.Signature)

	_, err = openpgp.CheckArmoredDetachedSignature(publicKeys, target, sig)
	if err != nil {
		hintMsg := ""
		if strings.Contains(err.Error(), "signature made by unknown entity") {
			hintMsg = " (hint: provided public key does not match signature)"
		}
		return fmt.Errorf("Checking signature: %s%s", err, hintMsg)
	}

	return nil
}

type signedObj struct {
	Contents  string
	Signature string
}

func (v Verification) readObject(ref string) (signedObj, error) {
	// Check tag first since "cat-file commit <tag>" will resolve tag first,
	// then return commit which may not be signed itself
	out, _, err := v.run([]string{"cat-file", "tag", ref})
	if err == nil {
		return v.extractTagSignature(out)
	}

	out, _, err = v.run([]string{"cat-file", "commit", ref})
	if err == nil {
		return v.extractCommitSignature(out)
	}

	return signedObj{}, fmt.Errorf("Reading git object for '%s': %s", ref, err)
}

func (v Verification) extractCommitSignature(obj string) (signedObj, error) {
	// TODO deal with gpgsig-sha256
	sectionReader := lineSectionReader{
		// sig is in the gpgsig key
		StartLine:   "gpgsig -----BEGIN PGP SIGNATURE-----",
		EndLine:     " -----END PGP SIGNATURE-----",
		Description: "PGP SIGNATURE",
	}

	nonSig, sig, err := sectionReader.Read(obj, true)
	if err != nil {
		return signedObj{}, fmt.Errorf("Expected to find commit signature: %s", err)
	}

	sig = strings.TrimPrefix(sig, "gpgsig ")    // header
	sig = strings.Replace(sig, "\n ", "\n", -1) // indents

	return signedObj{Contents: nonSig, Signature: sig}, nil
}

func (v Verification) extractTagSignature(obj string) (signedObj, error) {
	// TODO deal with gpgsig-sha256
	sectionReader := lineSectionReader{
		// sig is in the body
		StartLine:   "-----BEGIN PGP SIGNATURE-----",
		EndLine:     "-----END PGP SIGNATURE-----",
		Description: "PGP SIGNATURE",
	}

	nonSig, sig, err := sectionReader.Read(obj, true)
	if err != nil {
		return signedObj{}, fmt.Errorf("Expected to find tag signature: %s", err)
	}

	return signedObj{Contents: nonSig, Signature: sig}, nil
}

func (v Verification) run(args []string) (string, string, error) {
	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command("git", args...)
	cmd.Dir = v.repoPath
	cmd.Stdout = &stdoutBs
	cmd.Stderr = &stderrBs

	err := cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("Git %s: %s (stderr: %s)", args, err, stderrBs.String())
	}

	return stdoutBs.String(), stderrBs.String(), nil
}

/*

Example signed commit ($ is end of line):

tree f903fdfd4a595927b4b5eb392813b6f43ef55400$
parent 733d814eb2de09f0af7246a58351dd4b7106c48d$
author Git Git <git@k14s.io> 1604343706 +0000$
committer Git Git <git@k14s.io> 1604343706 +0000$
gpgsig -----BEGIN PGP SIGNATURE-----$
 Version: GnuPG v1$
 $
 iQEcBAABAgAGBQJfoFeaAAoJEIEYa81u6pVfLQQIAJJPuS5y47MB50HhmHrdF28T$
 msd/D6WMDa+YoKkOfHuUWKpx+GWm1jKN+apinBcAiArhSwxRh9bgqEmKgwdzclTV$
 FQJYyXdZqgpSiX6y4FTPzkKaINrw2ZuqkbPttLEW9TYIOnKFF0aDc9Lme18R0sv8$
 RQoyR4wi6eYNQ68gXuTXgwc7k3kCQl2PJhxurMxNrZDvJmq46dn+K0I2MeN2MNZ2$
 8z1as3B40KUCQsAZUfN6PJgxTlbz7f0lGi83Lq0rNvhM5AIUkMm5hUUMtX9bq75e$
 U8gY4YqK4DuOr9ArGzF7qJxaga2ZXf4CnW2zsmsTldBdRsnnm4M6EU3XK0dPVD0=$
 =hg/4$
 -----END PGP SIGNATURE-----$
$
signed-stranger-commit-msg$

*/

/*

Example signed tag ($ is end of line):

object 733d814eb2de09f0af7246a58351dd4b7106c48d$
type commit$
tag signed-trusted-tag$
tagger Git Git <git@k14s.io> 1604343706 +0000$
$
signed-trusted-tag-msg$
-----BEGIN PGP SIGNATURE-----$
Version: GnuPG v1$
$
iQEcBAABAgAGBQJfoFeaAAoJEHRsovO6vWHiLSkH/1/4dS1cm1Mq/cLaoeAJJUP7$
G4kYACY1iGUZgOKLquMkJ0Ng25CSVoRG/o2HEZeEw+QcSoElQJ48S/rdw6jHAQ7+$
yXBA6Q7wKw7opDJFGU1R6wv5ut8rzvqzLuCvSMwzYC3fJqyQAhbqftJWU348m9K5$
EWx1fEYorNtB3EX73gctyQgDMAxgCxCfRrrTdOBBQM3dnuUSOn+ShpogyQKS47sv$
IrtQ++wh3b5EXM6qw90tImzD7IXdNHtUP2yqwhKtb61o2g8f8GaPOMpXHEMe3yu3$
/M3q0pOTwOJasQlnN13uQyKiSeFY6lXhktuHJIdY8hQ3b2v3NrKIq9Ns4VOOvLY=$
=fAAV$
-----END PGP SIGNATURE-----$

*/
