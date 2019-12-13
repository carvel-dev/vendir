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

func (c LockConfig) WriteToFile(path string) error {
	bs, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("Marshaling lock config: %s", err)
	}

	err = ioutil.WriteFile(path, bs, 0700)
	if err != nil {
		return fmt.Errorf("Writing lock config: %s", err)
	}

	return nil
}
