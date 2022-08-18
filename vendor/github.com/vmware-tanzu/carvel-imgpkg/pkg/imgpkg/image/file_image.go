// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type FileImage struct {
	v1.Image
	path string
}

func NewFileImage(path string, labels map[string]string) (*FileImage, error) {
	sha256, err := sha256Path(path)
	if err != nil {
		return nil, err
	}

	layer, err := partial.UncompressedToLayer(&UncompressedFileLayer{
		diffID:    v1.Hash{Algorithm: "sha256", Hex: sha256},
		mediaType: types.DockerLayer,
		path:      path,
	})
	if err != nil {
		return nil, err
	}

	add := mutate.Addendum{
		Layer: layer,
		History: v1.History{
			Author:    "imgpkg",
			CreatedBy: "imgpkg",
			Created:   v1.Time{}, // static
		},
	}

	img, err := mutate.Append(empty.Image, add)
	if err != nil {
		return nil, err
	}

	if len(labels) > 0 {
		cfg, err := img.ConfigFile()
		if err != nil {
			return nil, fmt.Errorf("Fetching image config: %s", err)
		}

		cfg.Config.Labels = labels

		img, err = mutate.ConfigFile(img, cfg)
		if err != nil {
			return nil, err
		}
	}

	return &FileImage{img, path}, nil
}

func (i *FileImage) Remove() error {
	return os.Remove(i.path)
}

func sha256Path(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer file.Close()

	hash := sha256.New()

	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
