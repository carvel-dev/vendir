// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

// LockConfig outputs a vendir.lock with shas for reproducible vendir-ing.
type LockConfig struct {
	APIVersion  string          `json:"apiVersion"`
	Kind        string          `json:"kind"`
	Directories []LockDirectory `json:"directories"`
}

func NewLockConfig() LockConfig {
	return LockConfig{
		APIVersion: "vendir.k14s.io/v1alpha1",
		Kind:       "LockConfig",
	}
}

func LockFileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

func NewLockConfigFromFile(path string) (LockConfig, error) {
	bs, err := os.ReadFile(path)
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
	existingBytes, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("Failed to check existing lock file: %w", err)
	}

	bs, err := c.AsBytes()
	if err != nil {
		return fmt.Errorf("Marshaling lock config: %s", err)
	}

	if bytes.Compare(existingBytes, bs) != 0 {
		err = os.WriteFile(path, bs, 0600)
		if err != nil {
			return fmt.Errorf("Writing lock config: %s", err)
		}
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

func (c LockConfig) FindContents(dirPath, conPath string) (LockDirectoryContents, error) {
	for _, dir := range c.Directories {
		if dir.Path == dirPath {
			for _, con := range dir.Contents {
				if con.Path == conPath {
					return con, nil
				}
			}
			return LockDirectoryContents{}, fmt.Errorf("Expected to find contents '%s' "+
				"within directory '%s' in lock config, but did not", conPath, dirPath)
		}
	}
	return LockDirectoryContents{}, fmt.Errorf(
		"Expected to find directory '%s' within lock config, but did not", dirPath)
}

func (c LockConfig) FindDirectory(dirPath string) (LockDirectory, error) {
	for _, dir := range c.Directories {
		if dir.Path == dirPath {
			return dir, nil
		}
	}
	return LockDirectory{}, fmt.Errorf(
		"Expected to find directory '%s' within lock config, but did not", dirPath)
}

func (c LockConfig) Merge(other LockConfig) error {
	for _, dir := range other.Directories {
		for _, con := range dir.Contents {
			err := c.MergeContents(filepath.Join(dir.Path, con.Path), con)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c LockConfig) MergeContents(path string, replaceCon LockDirectoryContents) error {
	var matched bool

	for i, dir := range c.Directories {
		for j, con := range dir.Contents {
			if filepath.Join(dir.Path, con.Path) != path {
				continue
			}

			if matched {
				return fmt.Errorf("Expected to match exactly one directory, but matched multiple")
			}
			matched = true

			newCon := replaceCon
			newCon.Path = con.Path

			dir.Contents[j] = newCon
			c.Directories[i] = dir
		}
	}

	if !matched {
		return fmt.Errorf("Expected to match exactly one directory, but did not match any")
	}
	return nil
}
