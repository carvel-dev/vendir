// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	ctlimg "carvel.dev/imgpkg/pkg/imgpkg/image"
	"carvel.dev/imgpkg/pkg/imgpkg/imageset"
	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	"carvel.dev/imgpkg/pkg/imgpkg/lockconfig"
	plainimg "carvel.dev/imgpkg/pkg/imgpkg/plainimage"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
)

const (
	BundleConfigLabel = "dev.carvel.imgpkg.bundle"
)

// Logger Interface used for logging
type Logger interface {
	Errorf(msg string, args ...interface{})
	Warnf(msg string, args ...interface{})
	Debugf(msg string, args ...interface{})
	Tracef(msg string, args ...interface{})
	Logf(msg string, args ...interface{})
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImagesLockReader
type ImagesLockReader interface {
	Read(img regv1.Image) (lockconfig.ImagesLock, error)
}

type ImagesMetadata interface {
	Get(regname.Reference) (*regremote.Descriptor, error)
	Image(regname.Reference) (regv1.Image, error)
	Digest(regname.Reference) (regv1.Hash, error)
	FirstImageExists(digests []string) (string, error)
}

// GraphNode Node information of a Bundle
type GraphNode struct {
	Path          string
	ImageRef      string
	NestedBundles []GraphNode
}

// Bundle struct that represents a bundle
type Bundle struct {
	plainImg         *plainimg.PlainImage
	imgRetriever     ImagesMetadata
	imagesLockReader ImagesLockReader
	bundleFetcher    Fetcher

	// cachedNestedBundleGraph stores a graph with all the nested
	// bundles associated with the current bundle
	cachedNestedBundleGraph []GraphNode

	// cachedImageRefs stores set of ImageRefs that were
	// discovered as part of reading the bundle.
	// Includes refs only directly referenced by the bundle.
	cachedImageRefs *imageRefCache
}

// NewBundleFromPlainImage Creates a new Bundle with a PlainImage and uses Registry Fetcher
func NewBundleFromPlainImage(plainImg *plainimg.PlainImage, imagesMetadata ImagesMetadata) *Bundle {
	imagesLockReader := NewImagesLockReader()
	return &Bundle{plainImg: plainImg, imgRetriever: imagesMetadata,
		imagesLockReader: imagesLockReader, bundleFetcher: NewRegistryFetcher(imagesMetadata, imagesLockReader),
		cachedImageRefs: newImageRefCache()}
}

// NewBundle Creates a new Bundle
func NewBundle(plainImg *plainimg.PlainImage, imagesMetadata ImagesMetadata, imagesLockReader ImagesLockReader, bundleFetcher Fetcher) *Bundle {
	return &Bundle{plainImg: plainImg, imgRetriever: imagesMetadata,
		imagesLockReader: imagesLockReader, bundleFetcher: bundleFetcher,
		cachedImageRefs: newImageRefCache()}
}

// NewBundleFromRef Creates a new Bundle from an image full reference
func NewBundleFromRef(ref string, imagesMetadata ImagesMetadata, imagesLockReader ImagesLockReader, bundleFetcher Fetcher) *Bundle {
	return NewBundle(plainimg.NewPlainImage(ref, imagesMetadata), imagesMetadata, imagesLockReader, bundleFetcher)
}

// DigestRef Bundle full location including registry, repository and digest
func (o *Bundle) DigestRef() string { return o.plainImg.DigestRef() }

// Digest Bundle Digest
func (o *Bundle) Digest() string { return o.plainImg.Digest() }

// Repo Bundle registry and Repository
func (o *Bundle) Repo() string { return o.plainImg.Repo() }

// Tag Bundle Tag
func (o *Bundle) Tag() string { return o.plainImg.Tag() }

// NestedBundles Provides information about the Graph of nested bundles associated with the current bundle
func (o *Bundle) NestedBundles() []GraphNode { return o.cachedNestedBundleGraph }

func (o *Bundle) findCachedImageRef(digestRef string) (ImageRef, bool) {
	ref, found := o.cachedImageRefs.ImageRef(digestRef)
	if found {
		return ref.DeepCopy(), true
	}

	for _, imgRef := range o.cachedImageRefs.All() {
		for _, loc := range imgRef.Locations() {
			if loc == digestRef {
				return imgRef.DeepCopy(), true
			}
		}
	}

	return ImageRef{}, false
}

// NoteCopy writes an image-location representing the bundle / images that have been copied
func (o *Bundle) NoteCopy(processedImages *imageset.ProcessedImages, reg ImagesMetadataWriter, ui util.LoggerWithLevels) error {
	locationsCfg := ImageLocationsConfig{
		APIVersion: LocationAPIVersion,
		Kind:       ImageLocationsKind,
	}
	foundImages := map[string]bool{}
	var bundleProcessedImage imageset.ProcessedImage
	for _, image := range processedImages.All() {
		imgDigest, err := regname.NewDigest(image.UnprocessedImageRef.DigestRef)
		if err != nil {
			panic(fmt.Sprintf("Internal inconsistency: Image '%s' is not a valid Digest Reference", err))
		}

		ref, found := o.findCachedImageRef(image.UnprocessedImageRef.DigestRef)
		if found {
			if _, ok := foundImages[imgDigest.DigestStr()]; !ok {
				locationsCfg.Images = append(locationsCfg.Images, ImageLocation{
					Image:    ref.Image,
					IsBundle: *ref.IsBundle,
				})
				foundImages[imgDigest.DigestStr()] = true
			}
		}

		if imgDigest.DigestStr() == o.Digest() {
			bundleProcessedImage = image
		}
	}

	if len(locationsCfg.Images) != o.cachedImageRefs.Size() {
		panic(fmt.Sprintf("Expected: on bundle %s %d images to be written to Location OCI. Actual: %d were written", o.DigestRef(), o.cachedImageRefs.Size(), len(locationsCfg.Images)))
	}

	destinationRef, err := regname.NewDigest(bundleProcessedImage.DigestRef)
	if err != nil {
		panic(fmt.Sprintf("Internal inconsistency: '%s' have to be a digest", bundleProcessedImage.DigestRef))
	}

	ui.Debugf("creating Locations OCI Image\n")

	// Using NewNoopLevelLogger because we do not want to have output from this push
	return NewLocations(ui).Save(reg, destinationRef, locationsCfg, util.NewNoopLevelLogger())
}

// Pull Downloads bundle image to disk and checks if it can update the ImagesLock file
func (o *Bundle) Pull(outputPath string, logger Logger, pullNestedBundles bool) (bool, error) {
	isRootBundleRelocated, err := o.pull(outputPath, logger, pullNestedBundles, "", map[string]bool{}, 0)
	if err != nil {
		return false, err
	}

	logger.Logf("\nLocating image lock file images...\n")
	if isRootBundleRelocated {
		logger.Logf("The bundle repo (%s) is hosting every image specified in the bundle's Images Lock file (.imgpkg/images.yml)\n", o.Repo())
	} else {
		logger.Logf("One or more images not found in bundle repo; skipping lock file update\n")
	}
	return isRootBundleRelocated, nil
}

func (o *Bundle) pull(baseOutputPath string, logger Logger, pullNestedBundles bool, bundlePath string, imagesProcessed map[string]bool, numSubBundles int) (bool, error) {
	img, err := o.checkedImage()
	if err != nil {
		return false, err
	}

	if o.rootBundle(bundlePath) {
		logger.Logf("Pulling bundle '%s'\n", o.DigestRef())
	} else {
		logger.Logf("Pulling nested bundle '%s'\n", o.DigestRef())
	}

	bundleDigestRef, err := regname.NewDigest(o.plainImg.DigestRef())
	if err != nil {
		return false, err
	}

	err = ctlimg.NewDirImage(filepath.Join(baseOutputPath, bundlePath), img, util.NewIndentedLevelLogger(logger)).AsDirectory()
	if err != nil {
		return false, fmt.Errorf("Extracting bundle into directory: %s", err)
	}

	imagesLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(baseOutputPath, bundlePath, ImgpkgDir, ImagesLockFile))
	if err != nil {
		return false, err
	}

	bundleImageRefs, err := NewImageRefsFromImagesLock(imagesLock, LocationsConfig{
		logger:          logger,
		imgRetriever:    o.imgRetriever,
		bundleDigestRef: bundleDigestRef,
	})
	if err != nil {
		return false, err
	}

	isRelocatedToBundle, err := bundleImageRefs.UpdateRelativeToRepo(o.imgRetriever, o.Repo())
	if err != nil {
		return false, err
	}

	if pullNestedBundles {
		for _, bundleImgRef := range bundleImageRefs.ImageRefs() {
			if isBundle, alreadyProcessedImage := imagesProcessed[bundleImgRef.Image]; alreadyProcessedImage {
				if isBundle {
					util.NewIndentedLevelLogger(logger).Logf("Pulling nested bundle '%s'\n", bundleImgRef.Image)
					util.NewIndentedLevelLogger(logger).Logf("Skipped, already downloaded\n")
				}
				continue
			}

			subBundle := NewBundleFromRef(bundleImgRef.PrimaryLocation(), o.imgRetriever, o.imagesLockReader, o.bundleFetcher)

			var isBundle bool
			if bundleImgRef.IsBundle != nil {
				isBundle = *bundleImgRef.IsBundle
			} else {
				isBundle, err = subBundle.IsBundle()
				if err != nil {
					return false, err
				}
			}

			imagesProcessed[bundleImgRef.Image] = isBundle

			if !isBundle {
				continue
			}

			numSubBundles++

			if o.shouldPrintNestedBundlesHeader(bundlePath, numSubBundles) {
				logger.Logf("\nNested bundles\n")
			}
			bundleDigest, err := regname.NewDigest(bundleImgRef.Image)
			if err != nil {
				return false, err
			}
			_, err = subBundle.pull(baseOutputPath, util.NewIndentedLevelLogger(logger), pullNestedBundles, o.subBundlePath(bundleDigest), imagesProcessed, numSubBundles)
			if err != nil {
				return false, err
			}

			o.cachedNestedBundleGraph = append(o.cachedNestedBundleGraph, GraphNode{
				Path:          filepath.Join(baseOutputPath, o.subBundlePath(bundleDigest)),
				ImageRef:      bundleImgRef.PrimaryLocation(),
				NestedBundles: subBundle.cachedNestedBundleGraph,
			})
		}
	}

	if isRelocatedToBundle {
		err := bundleImageRefs.ImagesLock().WriteToPath(filepath.Join(baseOutputPath, bundlePath, ImgpkgDir, ImagesLockFile))
		if err != nil {
			return false, fmt.Errorf("Rewriting image lock file: %s", err)
		}
	}

	return isRelocatedToBundle, nil
}

func (*Bundle) subBundlePath(bundleDigest regname.Digest) string {
	return filepath.Join(ImgpkgDir, BundlesDir, strings.ReplaceAll(bundleDigest.DigestStr(), "sha256:", "sha256-"))
}

func (o *Bundle) shouldPrintNestedBundlesHeader(bundlePath string, bundlesProcessed int) bool {
	return o.rootBundle(bundlePath) && bundlesProcessed == 1
}

func (o *Bundle) rootBundle(bundlePath string) bool {
	return bundlePath == ""
}

func (o *Bundle) checkedImage() (regv1.Image, error) {
	isBundle, err := o.IsBundle()
	if err != nil {
		return nil, fmt.Errorf("Checking if image is bundle: %s", err)
	}
	if !isBundle {
		return nil, notABundleError{}
	}

	img, err := o.plainImg.Fetch()
	if err == nil && img == nil {
		panic("Unreachable")
	}
	return img, err
}

func newImageRefCache() *imageRefCache {
	return &imageRefCache{
		cache: map[string]ImageRef{},
		mutex: &sync.Mutex{},
	}
}

type imageRefCache struct {
	cache map[string]ImageRef
	mutex *sync.Mutex
}

// ImageRef retrieves the ImageRef associated with imageRef
func (i *imageRefCache) ImageRef(imageRef string) (ImageRef, bool) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	ref, err := regname.NewDigest(imageRef)
	if err != nil {
		panic(fmt.Sprintf("Internal inconsistency: Image '%s' needs to be a full reference", imageRef))
	}

	foundImgRef, found := i.cache[ref.DigestStr()]
	return foundImgRef, found
}

// Size number of entries in the cache
func (i *imageRefCache) Size() int {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	return len(i.cache)
}

// All images from the cache, but it will panic if any ImageRef is not a digest/the error field is set
func (i *imageRefCache) All() []ImageRef {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	var result []ImageRef
	for _, ref := range i.cache {
		// if error is set panic
		if ref.Error != "" {
			panic(fmt.Sprintf("Internal consistency: function All called when there was an image '%s' that contains an error. Should call AllImagesWithErrors instead", ref.ImageRef.Image))
		}
		result = append(result, ref.DeepCopy())
	}
	return result
}

// AllImagesWithErrors images from the cache even when there is an error
func (i *imageRefCache) AllImagesWithErrors() []ImageRef {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	var result []ImageRef
	for _, ref := range i.cache {
		result = append(result, ref.DeepCopy())
	}
	return result
}

// StoreImageRef saves ImageRef into the cache
func (i *imageRefCache) StoreImageRef(imageRef ImageRef) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	key := ""
	if imageRef.Error != "" {
		// When an error happen while fetching artifact(signatures or other) images the imageRef.Image is not going to contain
		// the Digest to the image because we could not get it from the registry.
		// In this particular case it is ok to use has key whatever is present in the Image field
		key = imageRef.Image
	} else {
		ref, err := regname.NewDigest(imageRef.Image)
		if err != nil {
			panic(fmt.Sprintf("Internal inconsistency: Image '%s' needs to be a full reference", imageRef.Image))
		}

		key = ref.DigestStr()
	}

	i.cache[key] = imageRef
}
