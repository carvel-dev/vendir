package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	ctldir "github.com/k14s/vendir/pkg/vendir/directory"
)

type Config struct {
	APIVersion  string          `json:"apiVersion"`
	Kind        string          `json:"kind"`
	Directories []ctldir.Config `json:"directories,omitempty"`
}

func NewConfigFromFile(path string) (Config, error) {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("Reading config '%s': %s", path, err)
	}

	return NewConfigFromBytes(bs)
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
	const (
		knownAPIVersion = "vendir.k14s.io/v1alpha1"
		knownKind       = "Config"
	)

	if c.APIVersion != knownAPIVersion {
		return fmt.Errorf("Validating apiVersion: Unknown version (known: %s)", knownAPIVersion)
	}
	if c.Kind != knownKind {
		return fmt.Errorf("Validating kind: Unknown kind (known: %s)", knownKind)
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

			newCon := ctldir.ConfigContents{
				Path:         con.Path,
				Directory:    &ctldir.ConfigContentsDirectory{Path: dirPath},
				IncludePaths: con.IncludePaths,
				ExcludePaths: con.ExcludePaths,
				LegalPaths:   con.LegalPaths,
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
			newCon.Path = ctldir.EntireDirPath

			result.Directories = append(result.Directories, ctldir.Config{
				Path:     path,
				Contents: []ctldir.ConfigContents{newCon},
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

func (c Config) checkOverlappingPaths() error {
	paths := []string{}

	for _, dir := range c.Directories {
		for _, con := range dir.Contents {
			paths = append(paths, filepath.Join(dir.Path, con.Path))
		}
	}

	for i, path := range paths {
		for i2, path2 := range paths {
			if i != i2 && strings.Contains(path2, path) {
				return fmt.Errorf("Expected to not "+
					"manage overlapping paths: '%s' and '%s'", path2, path)
			}
		}
	}

	return nil
}
