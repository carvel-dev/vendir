// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"archive/tar"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	"carvel.dev/imgpkg/pkg/imgpkg/plainimage"
	"github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

const (
	locationsTagFmt string = "%s-%s.image-locations.imgpkg"
)

type LocationsNotFound struct {
	image string
}

var imageNotFoundStatusCode = map[int]struct{}{
	http.StatusNotFound:     {},
	http.StatusUnauthorized: {},
	http.StatusForbidden:    {},
}

func (n LocationsNotFound) Error() string {
	return fmt.Sprintf("Locations image in %s could not be found", n.image)
}

type LocationsConfigs struct {
	reader LocationImageReader
	ui     util.LoggerWithLevels
}

type LocationImageReader interface {
	Read(img regv1.Image) (ImageLocationsConfig, error)
}

// NewLocations constructor for creating a LocationsConfigs
func NewLocations(ui util.LoggerWithLevels) *LocationsConfigs {
	return NewLocationsWithReader(&locationsSingleLayerReader{}, ui)
}

// NewLocationsWithReader constructor for LocationsConfigs
func NewLocationsWithReader(reader LocationImageReader, ui util.LoggerWithLevels) *LocationsConfigs {
	return &LocationsConfigs{reader: reader, ui: ui}
}

// Fetch Retrieve the ImageLocationsConfig for a particular Bundle
func (r *LocationsConfigs) Fetch(registry ImagesMetadata, bundleRef name.Digest) (ImageLocationsConfig, error) {
	r.ui.Tracef("Fetching Locations OCI Images for bundle: %s\n", bundleRef)

	locRef, err := r.locationsRefFromBundleRef(bundleRef)
	if err != nil {
		return ImageLocationsConfig{}, fmt.Errorf("Calculating locations image tag: %s", err)
	}

	img, err := registry.Image(locRef)
	if err != nil {
		if terr, ok := err.(*transport.Error); ok {
			if _, ok := imageNotFoundStatusCode[terr.StatusCode]; ok {
				r.ui.Debugf("Did not find Locations OCI Image for bundle: %s\n", bundleRef)
				return ImageLocationsConfig{}, &LocationsNotFound{image: locRef.Name()}
			}
		}
		return ImageLocationsConfig{}, fmt.Errorf("Fetching location image: %s", err)
	}

	r.ui.Tracef("Reading the locations configuration file\n")

	cfg, err := r.reader.Read(img)
	if err != nil {
		return ImageLocationsConfig{}, fmt.Errorf("Reading fetched location image: %s", err)
	}

	return cfg, err
}

// LocationsImageDigest Retrieve the Locations OCI Image Digest
func (r LocationsConfigs) LocationsImageDigest(registry ImagesMetadata, bundleRef name.Digest) (name.Digest, error) {
	r.ui.Tracef("Fetching Locations OCI Images for bundle: %s\n", bundleRef)

	locRef, err := r.locationsRefFromBundleRef(bundleRef)
	if err != nil {
		return name.Digest{}, fmt.Errorf("Calculating locations image tag: %s", err)
	}

	digest, err := registry.Digest(locRef)
	if err != nil {
		if terr, ok := err.(*transport.Error); ok {
			if _, ok := imageNotFoundStatusCode[terr.StatusCode]; ok {
				r.ui.Debugf("Did not find Locations OCI Image for bundle: %s\n", bundleRef)
				return name.Digest{}, &LocationsNotFound{image: locRef.Name()}
			}
		}
		return name.Digest{}, fmt.Errorf("Fetching location image: %s", err)
	}
	return bundleRef.Digest(digest.String()), nil
}

// Save the locations information for the bundle in the registry
// This function will create an OCI Image that contains the Location information of all the images that are part of the Bundle
func (r LocationsConfigs) Save(reg ImagesMetadataWriter, bundleRef name.Digest, config ImageLocationsConfig, logger Logger) error {
	r.ui.Tracef("saving Locations OCI Image for bundle: %s\n", bundleRef.Name())

	locRef, err := r.locationsRefFromBundleRef(bundleRef)
	if err != nil {
		return fmt.Errorf("Calculating locations image tag: %s", err)
	}

	tmpDir, err := os.MkdirTemp("", "imgpkg-bundle-locations")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	err = config.WriteToPath(filepath.Join(tmpDir, LocationFilepath))
	if err != nil {
		return err
	}

	r.ui.Tracef("Pushing image\n")

	_, err = plainimage.NewContents([]string{tmpDir}, nil, false).Push(locRef, nil, reg.CloneWithLogger(util.NewNoopProgressBar()), logger)
	if err != nil {
		// Immutable tag errors within registries are not standardized.
		// Assume word "immutable" would be present in most cases.
		// Example:
		//    TAG_INVALID: The image tag 'sha256-81c592...289f6.image-locations.imgpkg'
		//    already exists and cannot be overwritten because the repository is immutable
		if strings.Contains(err.Error(), "immutable") {
			if _, fetchErr := r.Fetch(reg, bundleRef); fetchErr == nil {
				// Ignore failed write if existing ImageLocations record is present.
				// (ImageLocations should be used as a cache and not an authoritative record.)
				// Failure to write may happen if:
				// - registry has immutable tags functionality _and_ we are modifying tag to a new digest
				//   which means that existing ImageLocations record does not match new record.
				//   That may happen if we have previously written an "incorrect" record (e.g. due to a bug)
				//   or because we changed format of ImageLocations record.
				// imgpkg should be backwards compatible to read previously written ImageLocations
				// hence write of a new ImageLocations is best-effort.
				return nil
			}
		}
		return fmt.Errorf("Pushing locations image to '%s': %s", locRef.Name(), err)
	}

	return nil
}

func (r LocationsConfigs) locationsRefFromBundleRef(bundleRef name.Digest) (name.Tag, error) {
	hash, err := regv1.NewHash(bundleRef.DigestStr())
	if err != nil {
		return name.Tag{}, err
	}

	tag, err := name.NewTag(bundleRef.Context().Name())
	if err != nil {
		return name.Tag{}, err
	}

	return tag.Tag(fmt.Sprintf(locationsTagFmt, hash.Algorithm, hash.Hex)), nil
}

type locationsSingleLayerReader struct{}

func (o *locationsSingleLayerReader) Read(img regv1.Image) (ImageLocationsConfig, error) {
	conf := ImageLocationsConfig{}

	layers, err := img.Layers()
	if err != nil {
		return conf, err
	}

	if len(layers) != 1 {
		return conf, fmt.Errorf("Expected locations OCI Image to only have a single layer, got %d", len(layers))
	}

	layer := layers[0]

	mediaType, err := layer.MediaType()
	if err != nil {
		return conf, err
	}

	if mediaType != types.DockerLayer {
		return conf, fmt.Errorf("Expected layer to have docker layer media type, was %s", mediaType)
	}

	// here we know layer is .tgz so decompress and read tar headers
	unzippedReader, err := layer.Uncompressed()
	if err != nil {
		return conf, fmt.Errorf("Could not read locations image layer contents: %v", err)
	}

	tarReader := tar.NewReader(unzippedReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				return conf, fmt.Errorf("Expected to find image-locations.yml in location image")
			}
			return conf, fmt.Errorf("Reading tar: %v", err)
		}

		basename := filepath.Base(header.Name)
		if basename == LocationFilepath {
			break
		}
	}

	bs, err := io.ReadAll(tarReader)
	if err != nil {
		return conf, fmt.Errorf("Reading image-locations.yml from layer: %s", err)
	}

	return NewLocationConfigFromBytes(bs)
}
