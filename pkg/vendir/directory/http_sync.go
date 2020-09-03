// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package directory

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
)

type HTTPSync struct {
	opts       ConfigContentsHTTP
	refFetcher RefFetcher
}

func NewHTTPSync(opts ConfigContentsHTTP, refFetcher RefFetcher) *HTTPSync {
	return &HTTPSync{opts, refFetcher}
}

func (t *HTTPSync) Sync(dstPath string) (LockConfigContentsHTTP, error) {
	lockConf := LockConfigContentsHTTP{}

	if len(t.opts.URL) == 0 {
		return lockConf, fmt.Errorf("Expected non-empty URL")
	}

	tmpFile, err := TempFile("vendir-http")
	if err != nil {
		return lockConf, err
	}

	defer os.Remove(tmpFile.Name())

	err = t.downloadFileAndChecksum(tmpFile)
	if err != nil {
		return lockConf, fmt.Errorf("Downloading URL: %s", err)
	}

	incomingTmpPath, err := TempDir("http")
	if err != nil {
		return lockConf, err
	}

	defer os.RemoveAll(incomingTmpPath)

	_, err = Archive{tmpFile.Name(), true, t.opts.URL}.Unpack(incomingTmpPath)
	if err != nil {
		return lockConf, fmt.Errorf("Unpacking archive: %s", err)
	}

	err = MoveDir(incomingTmpPath, dstPath)
	if err != nil {
		return lockConf, err
	}

	return lockConf, nil
}

func (t *HTTPSync) downloadFile(dst io.Writer) error {
	req, err := http.NewRequest("GET", t.opts.URL, nil)
	if err != nil {
		return fmt.Errorf("Building request: %s", err)
	}

	err = t.addAuth(req)
	if err != nil {
		return fmt.Errorf("Adding auth to request: %s", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Initiating URL download: %s", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Expected 200 OK, but was '%s'", resp.Status)
	}

	_, err = io.Copy(dst, resp.Body)
	if err != nil {
		return fmt.Errorf("Writing downloaded content: %s", err)
	}

	return nil
}

func (t *HTTPSync) downloadFileAndChecksum(dst io.Writer) error {
	var digestName, expectedDigestVal string
	var digestDst hash.Hash

	switch {
	case len(t.opts.SHA256) > 0:
		digestName = "sha256"
		digestDst = sha256.New()
		expectedDigestVal = t.opts.SHA256
		dst = io.MultiWriter(dst, digestDst)
	}

	err := t.downloadFile(dst)
	if err != nil {
		return err
	}

	if len(expectedDigestVal) > 0 {
		actualDigestVal := fmt.Sprintf("%x", digestDst.Sum(nil))

		if expectedDigestVal != actualDigestVal {
			errMsg := "Expected digest to match '%s:%s', but was '%s:%s'"
			return fmt.Errorf(errMsg, digestName, expectedDigestVal, digestName, actualDigestVal)
		}
	}

	return nil
}

func (t *HTTPSync) addAuth(req *http.Request) error {
	if t.opts.SecretRef == nil {
		return nil
	}

	secret, err := t.refFetcher.GetSecret(t.opts.SecretRef.Name)
	if err != nil {
		return err
	}

	for name, _ := range secret.Data {
		switch name {
		case k8s_corev1_BasicAuthUsernameKey:
		case k8s_corev1_BasicAuthPasswordKey:
		default:
			return fmt.Errorf("Unknown secret field '%s' in secret '%s'", name, secret.Name)
		}
	}

	if _, found := secret.Data[k8s_corev1_BasicAuthUsernameKey]; found {
		req.SetBasicAuth(string(secret.Data[k8s_corev1_BasicAuthUsernameKey]),
			string(secret.Data[k8s_corev1_BasicAuthPasswordKey]))
	}

	return nil
}
