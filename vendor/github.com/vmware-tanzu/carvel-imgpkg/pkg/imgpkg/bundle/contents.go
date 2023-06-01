// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/plainimage"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
)

const (
	ImgpkgDir      = ".imgpkg"
	BundlesDir     = "bundles"
	ImagesLockFile = "images.yml"
)

type Contents struct {
	paths               []string
	excludedPaths       []string
	preservePermissions bool
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImagesMetadataWriter
type ImagesMetadataWriter interface {
	ImagesMetadata
	WriteImage(regname.Reference, regv1.Image, chan regv1.Update) error
	WriteTag(ref regname.Tag, taggagle regremote.Taggable) error
	CloneWithLogger(logger util.ProgressLogger) registry.Registry
}

// NewContents creates Contents struct
func NewContents(paths []string, excludedPaths []string, preservePermissions bool) Contents {
	return Contents{paths: paths, excludedPaths: excludedPaths, preservePermissions: preservePermissions}
}

// Push the contents of the bundle to the registry as an OCI Image
func (b Contents) Push(uploadRef regname.Tag, registry ImagesMetadataWriter, logger Logger) (string, error) {
	err := b.validate()
	if err != nil {
		return "", err
	}

	labels := map[string]string{BundleConfigLabel: "true"}
	return plainimage.NewContents(b.paths, b.excludedPaths, b.preservePermissions).Push(uploadRef, labels, registry, logger)
}

// PresentsAsBundle checks if the provided folders have the needed structure to be a bundle
func (b Contents) PresentsAsBundle() (bool, error) {
	imgpkgDirs, err := b.findImgpkgDirs()
	if err != nil {
		return false, err
	}

	err = b.validateImgpkgDirs(imgpkgDirs)
	if _, ok := err.(bundleValidationError); ok {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (b Contents) validate() error {
	imgpkgDirs, err := b.findImgpkgDirs()
	if err != nil {
		return err
	}

	err = b.validateImgpkgDirs(imgpkgDirs)
	if err != nil {
		return err
	}

	return nil
}

func (b *Contents) findImgpkgDirs() ([]string, error) {
	var bundlePaths []string
	for _, path := range b.paths {
		err := filepath.Walk(path, func(currPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if filepath.Base(currPath) != ImgpkgDir {
				return nil
			}

			currPath, err = filepath.Abs(currPath)
			if err != nil {
				return err
			}

			bundlePaths = append(bundlePaths, currPath)

			return nil
		})

		if err != nil {
			return []string{}, err
		}
	}

	return bundlePaths, nil
}

func (b Contents) validateImgpkgDirs(imgpkgDirs []string) error {
	if len(imgpkgDirs) != 1 {
		imgpkgPath := filepath.Join(ImgpkgDir, ImagesLockFile)

		msg := fmt.Sprintf("This directory is not a bundle. It is missing %s", imgpkgPath)
		if len(imgpkgDirs) > 0 {
			msg = fmt.Sprintf("This directory contains multiple bundle definitions. Only a single instance of %s can be provided and instead these were provided %s", imgpkgPath, strings.Join(imgpkgDirs, ", "))
		}

		return bundleValidationError{msg}
	}

	// make sure it is a child of one input dir
	path := imgpkgDirs[0]
	for _, flagPath := range b.paths {
		flagPath, err := filepath.Abs(flagPath)
		if err != nil {
			return err
		}

		if filepath.Dir(path) == flagPath {
			imgpkgPath := filepath.Join(path, ImagesLockFile)
			if _, err := os.Stat(imgpkgPath); os.IsNotExist(err) {
				msg := fmt.Sprintf("The bundle expected .imgpkg/images.yml to exist, but it wasn't found in the path %s", imgpkgPath)

				return bundleValidationError{msg}
			}

			return nil
		}
	}

	msg := fmt.Sprintf("Expected '%s' directory, to be a direct child of one of: %s; was %s",
		ImgpkgDir, strings.Join(b.paths, ", "), path)

	return bundleValidationError{msg}
}

type bundleValidationError struct {
	msg string
}

func (b bundleValidationError) Error() string {
	return b.msg
}
