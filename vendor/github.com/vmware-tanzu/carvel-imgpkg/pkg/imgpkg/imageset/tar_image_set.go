// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imageset

import (
	"fmt"
	"io"
	"os"

	regname "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imagedesc"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imagetar"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
)

type TarImageSet struct {
	imageSet    ImageSet
	concurrency int
	logger      Logger
}

// NewTarImageSet provides export/import operations on a tarball for a set of images
func NewTarImageSet(imageSet ImageSet, concurrency int, logger Logger) TarImageSet {
	return TarImageSet{imageSet, concurrency, logger}
}

// Export Creates a Tar with the provided Images
func (i TarImageSet) Export(foundImages *UnprocessedImageRefs, outputPath string, registry registry.ImagesReaderWriter, imageLayerWriterCheck imagetar.ImageLayerWriterFilter, resume bool) (d *imagedesc.ImageRefDescriptors, err error) {
	ids, err := i.imageSet.Export(foundImages, registry)
	if err != nil {
		return nil, err
	}

	var outputFile *os.File
	var alreadyDownloadedLayers []v1.Layer

	// this temporary file is used only in the case were we are resuming the copy of an image to a tar
	// we are creating a temporary copy of the existing tar. This is done to be able to read the layers
	// when we are filling up the destination tar.
	var tmpFile *os.File
	if resume {
		// If the file cannot be open we assume that this is not a resume action.
		// This will just follow the normal path of resume == false
		outputFile, err = os.Open(outputPath)
		if err == nil {
			tmpFile, err = os.CreateTemp("", "imgpkg-tar-imageset-")
			if err != nil {
				return nil, fmt.Errorf("Creating tmp folder: %s", err)
			}
			defer os.Remove(tmpFile.Name())

			var cErr error
			_, cErr = io.Copy(tmpFile, outputFile)
			err = tmpFile.Close()
			if err != nil {
				return nil, err
			}
			err = outputFile.Close()
			if err != nil {
				return nil, err
			}
			if cErr != nil {
				return nil, err
			}

			alreadyDownloadedLayers, err = imagetar.NewTarReader(tmpFile.Name()).PresentLayers()
			if err != nil {
				return nil, fmt.Errorf("Reading previously created tar '%s': %s", outputPath, err)
			}

			i.logger.Logf("Going to reuse %d layers from the tar already in disk\n", len(alreadyDownloadedLayers))
		}
	}

	outputFile, err = os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("Creating file '%s': %s", outputPath, err)
	}
	err = outputFile.Close()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			return
		}
		if tmpFile != nil {
			var err1 error
			outputFile, err1 = os.Open(outputPath)
			if err1 != nil {
				err = fmt.Errorf("original error: %s, post exit error: %s", err, err1)
				return
			}
			lTmpFile, err1 := os.Open(tmpFile.Name())
			if err1 != nil {
				outputFile.Close()
				err = fmt.Errorf("original error: %s, post exit error: %s", err, err1)
				return
			}

			_, err1 = io.Copy(outputFile, lTmpFile)
			outputFile.Close()
			lTmpFile.Close()
			if err1 != nil {
				err = fmt.Errorf("original error: %s, post exit error: %s", err, err1)
				return
			}
		}
	}()

	outputFileOpener := func() (io.WriteCloser, error) {
		return os.OpenFile(outputPath, os.O_RDWR, 0755)
	}

	i.logger.Logf("writing layers...\n")

	opts := imagetar.TarWriterOpts{Concurrency: i.concurrency}

	err = imagetar.NewTarWriter(ids, outputFileOpener, opts, i.logger, imageLayerWriterCheck, alreadyDownloadedLayers).Write()
	return ids, err
}

// Import Copy tar with Images to the Registry
func (i *TarImageSet) Import(path string, importRepo regname.Repository, registry registry.ImagesReaderWriter) (*ProcessedImages, error) {
	imgOrIndexes, err := imagetar.NewTarReader(path).Read()
	if err != nil {
		return nil, err
	}

	processedImages, err := i.imageSet.Import(imgOrIndexes, importRepo, registry)
	if err != nil {
		return nil, err
	}

	return processedImages, err
}
