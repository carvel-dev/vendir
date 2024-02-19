// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	ctlfetch "carvel.dev/vendir/pkg/vendir/fetch"
)

type Sync struct {
	opts       ctlconf.DirectoryContentsHTTP
	refFetcher ctlfetch.RefFetcher
}

func NewSync(opts ctlconf.DirectoryContentsHTTP, refFetcher ctlfetch.RefFetcher) *Sync {
	return &Sync{opts, refFetcher}
}

func (t *Sync) Sync(dstPath string, tempArea ctlfetch.TempArea) (ctlconf.LockDirectoryContentsHTTP, error) {
	lockConf := ctlconf.LockDirectoryContentsHTTP{}

	if len(t.opts.URL) == 0 {
		return lockConf, fmt.Errorf("Expected non-empty URL")
	}

	tmpFile, err := tempArea.NewTempFile("vendir-http")
	if err != nil {
		return lockConf, err
	}

	defer os.Remove(tmpFile.Name())

	err = t.downloadFileAndChecksum(tmpFile)
	if err != nil {
		tmpFile.Close()
		return lockConf, fmt.Errorf("Downloading URL: %s", err)
	}

	incomingTmpPath := filepath.Dir(tmpFile.Name())
	archivePath := filepath.Join(incomingTmpPath, path.Base(t.opts.URL))
	tmpFile.Close()
	err = os.Rename(tmpFile.Name(), archivePath)
	if err != nil {
		return lockConf, err
	}

	if !t.opts.DisableUnpack {
		incomingTmpPath, err = tempArea.NewTempDir("http")
		if err != nil {
			return lockConf, err
		}

		defer os.RemoveAll(incomingTmpPath)

		_, err = ctlfetch.NewArchive(archivePath, true, t.opts.URL).Unpack(incomingTmpPath)
		if err != nil {
			return lockConf, fmt.Errorf("Unpacking archive: %s", err)
		}

		err = ctlfetch.MoveDir(incomingTmpPath, dstPath)
	} else {
		err = ctlfetch.MoveFile(archivePath, dstPath)
	}

	return lockConf, err
}

func (t *Sync) downloadFile(dst io.Writer) error {
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

func (t *Sync) downloadFileAndChecksum(dst io.Writer) error {
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

func (t *Sync) addAuth(req *http.Request) error {
	if t.opts.SecretRef == nil {
		return nil
	}

	secret, err := t.refFetcher.GetSecret(t.opts.SecretRef.Name)
	if err != nil {
		return err
	}

	for name := range secret.Data {
		switch name {
		case ctlconf.SecretK8sCorev1BasicAuthUsernameKey:
		case ctlconf.SecretK8sCorev1BasicAuthPasswordKey:
		default:
			return fmt.Errorf("Unknown secret field '%s' in secret '%s'", name, secret.Metadata.Name)
		}
	}

	if _, found := secret.Data[ctlconf.SecretK8sCorev1BasicAuthUsernameKey]; found {
		req.SetBasicAuth(string(secret.Data[ctlconf.SecretK8sCorev1BasicAuthUsernameKey]),
			string(secret.Data[ctlconf.SecretK8sCorev1BasicAuthPasswordKey]))
	}

	return nil
}
