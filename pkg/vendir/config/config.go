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
	APIVersion  string `json:"apiVersion"`
	Kind        string
	Directories []ctldir.Config
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
