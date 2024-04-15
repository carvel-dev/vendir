// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package plainimage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ctlimg "carvel.dev/imgpkg/pkg/imgpkg/image"
	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
)

// Contents of the OCI Image
type Contents struct {
	paths               []string
	excludedPaths       []string
	preservePermissions bool
}

// ImagesWriter defines the needed functions to write to the registry
type ImagesWriter interface {
	WriteImage(regname.Reference, regv1.Image, chan regv1.Update) error
	WriteTag(ref regname.Tag, taggagle regremote.Taggable) error
}

// NewContents creates the struct that represent an OCI Image based on the provided paths
func NewContents(paths []string, excludedPaths []string, preservePermissions bool) Contents {
	return Contents{paths: paths, excludedPaths: excludedPaths, preservePermissions: preservePermissions}
}

// Push the OCI Image to the registry
func (i Contents) Push(uploadRef regname.Tag, labels map[string]string, writer ImagesWriter, logger Logger) (string, error) {
	err := i.validate()
	if err != nil {
		return "", err
	}

	tarImg := ctlimg.NewTarImage(i.paths, i.excludedPaths, logger, i.preservePermissions)

	img, err := tarImg.AsFileImage(labels)
	if err != nil {
		return "", err
	}

	defer img.Remove()

	err = writer.WriteImage(uploadRef, img, nil)

	if err != nil {
		return "", fmt.Errorf("Writing '%s': %s", uploadRef.Name(), err)
	}

	digest, err := img.Digest()
	if err != nil {
		return "", err
	}

	uploadTagRef, err := util.BuildDefaultUploadTagRef(img, uploadRef.Repository)
	if err != nil {
		return "", fmt.Errorf("Building default upload tag image ref: %s", err)
	}

	err = writer.WriteTag(uploadTagRef, img)
	if err != nil {
		return "", fmt.Errorf("Writing Tag '%s': %s", uploadRef.Name(), err)
	}

	return fmt.Sprintf("%s@%s", uploadRef.Context(), digest), nil
}

func (i Contents) validate() error {
	return i.checkRepeatedPaths()
}

func (i Contents) checkRepeatedPaths() error {
	imageRootPaths := make(map[string][]string)
	for _, flagPath := range i.paths {
		err := filepath.Walk(flagPath, func(currPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			imageRootPath, err := filepath.Rel(flagPath, currPath)
			if err != nil {
				return err
			}

			if imageRootPath == "." {
				if info.IsDir() {
					return nil
				}
				imageRootPath = filepath.Base(flagPath)
			}
			imageRootPaths[imageRootPath] = append(imageRootPaths[imageRootPath], currPath)
			return nil
		})

		if err != nil {
			return err
		}
	}

	var repeatedPaths []string
	for _, v := range imageRootPaths {
		if len(v) > 1 {
			repeatedPaths = append(repeatedPaths, v...)
		}
	}
	if len(repeatedPaths) > 0 {
		return fmt.Errorf("Found duplicate paths: %s", strings.Join(repeatedPaths, ", "))
	}
	return nil
}
