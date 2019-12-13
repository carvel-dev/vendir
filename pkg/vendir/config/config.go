package config

import (
	"fmt"
	"io/ioutil"

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
	return nil
}
