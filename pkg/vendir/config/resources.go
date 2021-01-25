// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	kyaml "k8s.io/apimachinery/pkg/util/yaml"
)

type resource struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}

func parseResources(paths []string, resourceFunc func([]byte) error) error {
	for _, path := range paths {
		var bs []byte
		var err error

		if path == "-" {
			bs, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("Reading config from stdin: %s", err)
			}
		} else {
			bs, err = ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("Reading config '%s': %s", path, err)
			}
		}

		reader := kyaml.NewYAMLReader(bufio.NewReaderSize(bytes.NewReader(bs), 4096))

		for {
			docBytes, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("Parsing config '%s': %s", path, err)
			}
			err = resourceFunc(docBytes)
			if err != nil {
				return fmt.Errorf("Parsing resource config '%s': %s", path, err)
			}
		}
	}
	return nil
}
