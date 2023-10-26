// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	semver "github.com/hashicorp/go-version"
	"github.com/vmware-tanzu/carvel-vendir/pkg/vendir/version"
	"sigs.k8s.io/yaml"
)

const (
	knownAPIVersion = "vendir.k14s.io/v1alpha1"
	knownKind       = "Config"
)

type Config struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	MinimumRequiredVersion string `json:"minimumRequiredVersion"`

	Directories []Directory `json:"directories,omitempty"`
}

func NewConfigFromFiles(paths []string) (Config, []Secret, []ConfigMap, error) {
	var configs []Config
	var secrets []Secret
	var configMaps []ConfigMap
	secretsNames := map[string]Secret{}
	err := parseResources(paths, func(docBytes []byte) error {
		var res resource

		err := yaml.Unmarshal(docBytes, &res)
		if err != nil {
			return fmt.Errorf("Unmarshaling doc: %s", err)
		}

		switch {
		case res.APIVersion == "v1" && res.Kind == "Secret":
			var secret Secret

			err := yaml.Unmarshal(docBytes, &secret)
			if err != nil {
				return fmt.Errorf("Unmarshaling secret: %s", err)
			}

			if s, ok := secretsNames[secret.Metadata.Name]; ok {
				if !reflect.DeepEqual(s.Data, secret.Data) {
					return fmt.Errorf(
						"Expected to find one secret '%s', but found multiple", s.Metadata.Name)
				}
			}
			secretsNames[secret.Metadata.Name] = secret

		case res.APIVersion == "v1" && res.Kind == "ConfigMap":
			var cm ConfigMap

			err := yaml.Unmarshal(docBytes, &cm)
			if err != nil {
				return fmt.Errorf("Unmarshaling config map: %s", err)
			}
			configMaps = append(configMaps, cm)

		case res.APIVersion == knownAPIVersion && res.Kind == knownKind:
			config, err := NewConfigFromBytes(docBytes)
			if err != nil {
				return fmt.Errorf("Unmarshaling config: %s", err)
			}
			configs = append(configs, config)

		default:
			return fmt.Errorf("Unknown apiVersion '%s' or kind '%s' for resource",
				res.APIVersion, res.Kind)
		}
		return nil
	})

	for _, v := range secretsNames {
		secrets = append(secrets, v)
	}

	if err != nil {
		return Config{}, nil, nil, err
	}

	if len(configs) == 0 {
		return Config{}, nil, nil, fmt.Errorf("Expected to find at least one config, but found none")
	}
	if len(configs) > 1 {
		return Config{}, nil, nil, fmt.Errorf("Expected to find exactly one config, but found multiple")
	}

	return configs[0], secrets, configMaps, nil
}

func NewConfigFromBytes(bs []byte) (Config, error) {
	var config Config

	err := yaml.Unmarshal(bs, &config)
	if err != nil {
		return Config{}, fmt.Errorf("Unmarshaling config: %s", err)
	}

	err = config.Validate()
	if err != nil {
		return Config{}, fmt.Errorf("Validating config: %s", err)
	}

	return config, nil
}

func (c Config) Validate() error {
	if c.APIVersion != knownAPIVersion {
		return fmt.Errorf("Validating apiVersion: Unknown version (known: %s)", knownAPIVersion)
	}
	if c.Kind != knownKind {
		return fmt.Errorf("Validating kind: Unknown kind (known: %s)", knownKind)
	}

	if len(c.MinimumRequiredVersion) > 0 {
		if c.MinimumRequiredVersion[0] == 'v' {
			return fmt.Errorf("Validating minimum version: Must not have prefix 'v' (e.g. '0.8.0')")
		}

		userConstraint, err := semver.NewConstraint(">=" + c.MinimumRequiredVersion)
		if err != nil {
			return fmt.Errorf("Parsing minimum version constraint: %s", err)
		}

		vendirVersion, err := semver.NewVersion(version.Version)
		if err != nil {
			return fmt.Errorf("Parsing version constraint: %s", err)
		}

		if !userConstraint.Check(vendirVersion) {
			return fmt.Errorf("vendir version '%s' does "+
				"not meet the minimum required version '%s'", version.Version, c.MinimumRequiredVersion)
		}
	}

	for i, dir := range c.Directories {
		err := dir.Validate()
		if err != nil {
			return fmt.Errorf("Validating directory '%s' (%d): %s", dir.Path, i, err)
		}
	}

	return c.checkOverlappingPaths()
}

func (c Config) AsBytes() ([]byte, error) {
	bs, err := yaml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("Marshaling config: %s", err)
	}

	return bs, nil
}

func (c Config) UseDirectory(path, dirPath string) error {
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

			newCon := DirectoryContents{
				Path:         con.Path,
				Directory:    &DirectoryContentsDirectory{Path: dirPath},
				IncludePaths: con.IncludePaths,
				ExcludePaths: con.ExcludePaths,
				IgnorePaths:  con.IgnorePaths,
				LegalPaths:   con.LegalPaths,
				Lazy:         con.Lazy,
			}
			dir.Contents[j] = newCon
			c.Directories[i] = dir
		}
	}

	if !matched {
		return fmt.Errorf("Expected to match exactly one directory, but did not match any")
	}
	return nil
}

func (c Config) Subset(paths []string) (Config, error) {
	result := Config{
		APIVersion: c.APIVersion,
		Kind:       c.Kind,
	}
	pathsToSeen := map[string]bool{}

	for _, path := range paths {
		pathsToSeen[path] = false
	}

	for _, dir := range c.Directories {
		for _, con := range dir.Contents {
			path := filepath.Join(dir.Path, con.Path)

			seen, found := pathsToSeen[path]
			if !found {
				continue
			}
			if seen {
				return Config{}, fmt.Errorf("Expected to match path '%s' once, but matched multiple", path)
			}
			pathsToSeen[path] = true

			newCon := con // copy (but not deep unfortunately)
			newCon.Path = EntireDirPath

			result.Directories = append(result.Directories, Directory{
				Path:     path,
				Contents: []DirectoryContents{newCon},
			})
		}
	}

	for path, seen := range pathsToSeen {
		if !seen {
			return Config{}, fmt.Errorf("Expected to match path '%s' once, but did not match any", path)
		}
	}

	// return validated config
	return result, result.Validate()
}

func (c Config) Lock(lockConfig LockConfig) error {
	for _, dir := range c.Directories {
		for _, con := range dir.Contents {
			lockContents, err := lockConfig.FindContents(dir.Path, con.Path)
			if err != nil {
				return err
			}

			err = con.Lock(lockContents)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c Config) checkOverlappingPaths() error {
	paths := []string{}

	for _, dir := range c.Directories {
		for _, con := range dir.Contents {
			paths = append(paths, filepath.Join(dir.Path, con.Path))
		}
	}

	for i, path := range paths {
		for i2, path2 := range paths {
			if i != i2 && strings.HasPrefix(path2+string(filepath.Separator), path+string(filepath.Separator)) {
				return fmt.Errorf("Expected to not "+
					"manage overlapping paths: '%s' and '%s'", path2, path)
			}
		}
	}

	return nil
}
