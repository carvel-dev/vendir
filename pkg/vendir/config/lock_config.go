// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
	ctldir "github.com/k14s/vendir/pkg/vendir/directory"
)

type LockConfig struct {
	APIVersion  string              `json:"apiVersion"`
	Kind        string              `json:"kind"`
	Directories []ctldir.LockConfig `json:"directories"`
}

func NewLockConfig() LockConfig {
	return LockConfig{
		APIVersion: "vendir.k14s.io/v1alpha1",
		Kind:       "LockConfig",
	}
}

func NewLockConfigFromFile(path string) (LockConfig, error) {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return LockConfig{}, fmt.Errorf("Reading lock config '%s': %s", path, err)
	}

	return NewLockConfigFromBytes(bs)
}

func NewLockConfigFromBytes(bs []byte) (LockConfig, error) {
	var config LockConfig

	err := yaml.Unmarshal(bs, &config)
	if err != nil {
		return LockConfig{}, fmt.Errorf("Unmarshaling lock config: %s", err)
	}

	err = config.Validate()
	if err != nil {
		return LockConfig{}, fmt.Errorf("Validating lock config: %s", err)
	}

	return config, nil
}

func (c LockConfig) WriteToFile(path string) error {
	bs, err := c.AsBytes()
	if err != nil {
		return fmt.Errorf("Marshaling lock config: %s", err)
	}

	err = ioutil.WriteFile(path, bs, 0700)
	if err != nil {
		return fmt.Errorf("Writing lock config: %s", err)
	}

	return nil
}

func (c LockConfig) AsBytes() ([]byte, error) {
	bs, err := yaml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("Marshaling lock config: %s", err)
	}

	return bs, nil
}

func (c LockConfig) Validate() error {
	const (
		knownAPIVersion = "vendir.k14s.io/v1alpha1"
		knownKind       = "LockConfig"
	)

	if c.APIVersion != knownAPIVersion {
		return fmt.Errorf("Validating apiVersion: Unknown version (known: %s)", knownAPIVersion)
	}
	if c.Kind != knownKind {
		return fmt.Errorf("Validating kind: Unknown kind (known: %s)", knownKind)
	}
	return nil
}

func (c LockConfig) FindContents(dirPath, conPath string) (ctldir.LockConfigContents, error) {
	for _, dir := range c.Directories {
		if dir.Path == dirPath {
			for _, con := range dir.Contents {
				if con.Path == conPath {
					return con, nil
				}
			}
			return ctldir.LockConfigContents{}, fmt.Errorf("Expected to find contents '%s' "+
				"within directory '%s' in lock config, but did not", conPath, dirPath)
		}
	}
	return ctldir.LockConfigContents{}, fmt.Errorf(
		"Expected to find directory '%s' within lock config, but did not", dirPath)
}
