// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package inline

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
)

type Sync struct {
	opts       ctlconf.DirectoryContentsInline
	refFetcher ctlfetch.RefFetcher
}

func NewSync(opts ctlconf.DirectoryContentsInline, refFetcher ctlfetch.RefFetcher) *Sync {
	return &Sync{opts, refFetcher}
}

func (t *Sync) Sync(dstPath string) (ctlconf.LockDirectoryContentsInline, error) {
	lockConf := ctlconf.LockDirectoryContentsInline{}

	for path, content := range t.opts.Paths {
		err := t.writeFile(dstPath, path, content)
		if err != nil {
			return lockConf, err
		}
	}

	for _, source := range t.opts.PathsFrom {
		switch {
		case source.SecretRef != nil:
			err := t.writeFromSecret(dstPath, *source.SecretRef)
			if err != nil {
				return lockConf, err
			}

		case source.ConfigMapRef != nil:
			err := t.writeFromConfigMap(dstPath, *source.ConfigMapRef)
			if err != nil {
				return lockConf, err
			}

		default:
			return lockConf, fmt.Errorf("Expected either secretRef or configMapRef as a source")
		}
	}

	return lockConf, nil
}

func (t *Sync) writeFromSecret(dstPath string, secretRef ctlconf.DirectoryContentsInlineSourceRef) error {
	secret, err := t.refFetcher.GetSecret(secretRef.Name)
	if err != nil {
		return err
	}

	for name, val := range secret.Data {
		err := t.writeFile(dstPath, filepath.Join(secretRef.DirectoryPath, name), string(val))
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Sync) writeFromConfigMap(dstPath string, configMapRef ctlconf.DirectoryContentsInlineSourceRef) error {
	configMap, err := t.refFetcher.GetConfigMap(configMapRef.Name)
	if err != nil {
		return err
	}

	for name, val := range configMap.Data {
		err := t.writeFile(dstPath, filepath.Join(configMapRef.DirectoryPath, name), val)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Sync) writeFile(dstPath, subPath string, content string) error {
	newPath, err := ctlfetch.ScopedPath(dstPath, subPath)
	if err != nil {
		return err
	}

	parentDir := filepath.Dir(newPath)

	err = os.MkdirAll(parentDir, 0700)
	if err != nil {
		return fmt.Errorf("Making parent directory '%s': %s", parentDir, err)
	}

	err = ioutil.WriteFile(newPath, []byte(content), 0600)
	if err != nil {
		return fmt.Errorf("Writing file '%s': %s", newPath, err)
	}

	return nil
}
