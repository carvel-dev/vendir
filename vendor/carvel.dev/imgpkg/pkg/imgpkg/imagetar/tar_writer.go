// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package imagetar

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"carvel.dev/imgpkg/pkg/imgpkg/imagedesc"
	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
)

type Logger interface {
	Logf(str string, args ...interface{})
}

type TarWriterOpts struct {
	Concurrency int
}

type TarWriter struct {
	ids       *imagedesc.ImageRefDescriptors
	dstOpener func() (io.WriteCloser, error)

	dst           io.WriteCloser
	tf            *tar.Writer
	layersToWrite []imagedesc.ImageLayerDescriptor

	opts                  TarWriterOpts
	logger                Logger
	imageLayerWriterCheck ImageLayerWriterFilter
	layersFromOtherSource []regv1.Layer
}

// NewTarWriter constructor returning a mechanism to write image refs / layers to a tarball on disk.
func NewTarWriter(ids *imagedesc.ImageRefDescriptors, dstOpener func() (io.WriteCloser, error),
	opts TarWriterOpts, logger Logger, imageLayerWriterCheck ImageLayerWriterFilter,
	layersFromOtherSource []regv1.Layer) *TarWriter {
	return &TarWriter{
		ids:                   ids,
		dstOpener:             dstOpener,
		opts:                  opts,
		logger:                logger,
		imageLayerWriterCheck: imageLayerWriterCheck,
		layersFromOtherSource: layersFromOtherSource,
	}
}

func (w *TarWriter) Write() error {
	var err error

	w.dst, err = w.dstOpener()
	if err != nil {
		return err
	}
	defer w.dst.Close()

	w.tf = tar.NewWriter(w.dst)
	defer w.tf.Close()

	idsBytes, err := w.ids.AsBytes()
	if err != nil {
		return err
	}

	err = w.writeTarEntry(w.tf, "manifest.json", bytes.NewReader(idsBytes), int64(len(idsBytes)))
	if err != nil {
		return err
	}

	for _, td := range w.ids.Descriptors() {
		switch {
		case td.Image != nil:
			err := w.writeImage(*td.Image)
			if err != nil {
				return err
			}

		case td.ImageIndex != nil:
			err := w.writeImageIndex(*td.ImageIndex)
			if err != nil {
				return err
			}

		default:
			panic("Unknown item")
		}
	}

	return w.writeLayers()
}

func (w *TarWriter) writeImageIndex(td imagedesc.ImageIndexDescriptor) error {
	for _, idx := range td.Indexes {
		err := w.writeImageIndex(idx)
		if err != nil {
			return err
		}
	}

	for _, img := range td.Images {
		err := w.writeImage(img)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *TarWriter) writeImage(td imagedesc.ImageDescriptor) error {
	for _, imgLayer := range td.Layers {
		shouldLayerBeIncluded, err := w.imageLayerWriterCheck.ShouldLayerBeIncluded(imagedesc.NewDescribedCompressedLayer(imgLayer, nil))
		if err != nil {
			return err
		}
		if shouldLayerBeIncluded {
			w.layersToWrite = append(w.layersToWrite, imgLayer)
		}
	}
	return nil
}

type writtenLayer struct {
	Name   string
	Offset int64
	Layer  imagedesc.ImageLayerDescriptor
}

func (w *TarWriter) writeLayers() error {
	// Sort layers by digest to have deterministic archive
	sort.Slice(w.layersToWrite, func(i, j int) bool {
		return w.layersToWrite[i].Digest < w.layersToWrite[j].Digest
	})

	seekableDst, isSeekable := w.dst.(*os.File)
	isInflatable := (w.opts.Concurrency > 1) && isSeekable
	writtenLayers := map[string]writtenLayer{}

	// Inflate tar file so that multiple writes can happen in parallel
	for _, imgLayer := range w.layersToWrite {
		digest, err := regv1.NewHash(imgLayer.Digest)
		if err != nil {
			return err
		}

		name := digest.Algorithm + "-" + digest.Hex + ".tar.gz"

		// Dedup layers
		if _, found := writtenLayers[name]; found {
			continue
		}

		err = w.tf.Flush()
		if err != nil {
			return err
		}

		var stream io.Reader
		var currPos int64

		if isSeekable {
			currPos, err = seekableDst.Seek(0, 1)
			if err != nil {
				return fmt.Errorf("Find current pos: %s", err)
			}
		}

		if isInflatable {
			stream = nil
		} else {
			for _, layer := range w.layersFromOtherSource {
				d, err := layer.Digest()
				if err != nil {
					return fmt.Errorf("Retrieving digest: %s", err)
				}
				if d.String() == imgLayer.Digest {
					stream, err = layer.Compressed()
					if err != nil {
						return fmt.Errorf("Retrieve layer from file: %s", err)
					}
					break
				}
			}

			if stream == nil {
				foundLayer, err := w.ids.FindLayer(imgLayer)
				if err != nil {
					return err
				}

				stream, err = foundLayer.Open()
				if err != nil {
					return err
				}
			}
		}

		err = w.writeTarEntry(w.tf, name, stream, imgLayer.Size)
		if err != nil {
			return fmt.Errorf("Writing tar entry: %s", err)
		}

		writtenLayers[name] = writtenLayer{
			Name:   name,
			Layer:  imgLayer,
			Offset: currPos,
		}
	}

	err := w.tf.Flush()
	if err != nil {
		return err
	}

	if isInflatable {
		return w.fillInLayers(writtenLayers)
	}

	return nil
}

func (w *TarWriter) fillInLayers(writtenLayers map[string]writtenLayer) error {
	var sortedWrittenLayers []writtenLayer

	for _, writtenLayer := range writtenLayers {
		sortedWrittenLayers = append(sortedWrittenLayers, writtenLayer)
	}

	// Prefer larger sizes first
	sort.Slice(sortedWrittenLayers, func(i, j int) bool {
		return sortedWrittenLayers[i].Layer.Size >= sortedWrittenLayers[j].Layer.Size
	})

	errCh := make(chan error, len(writtenLayers))
	writeThrottle := util.NewThrottle(w.opts.Concurrency)

	// Fill in actual data
	for _, writtenLayer := range sortedWrittenLayers {
		writtenLayer := writtenLayer // copy

		go func() {
			writeThrottle.Take()
			defer writeThrottle.Done()

			errCh <- util.Retry(func() error {
				return w.fillInLayer(writtenLayer)
			})
		}()
	}

	for i := 0; i < len(writtenLayers); i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("Filling in a layer: %s", err)
		}
	}

	return nil
}

func (w *TarWriter) fillInLayer(wl writtenLayer) error {
	file, err := w.dstOpener()
	if err != nil {
		return err
	}

	defer file.Close()

	_, err = file.(*os.File).Seek(wl.Offset, 0)
	if err != nil {
		return fmt.Errorf("Seeking to offset: %s", err)
	}

	tw := tar.NewWriter(file)
	// Do not close tar writer as it would add unwanted footer

	foundLayer, err := w.ids.FindLayer(wl.Layer)
	if err != nil {
		return err
	}

	stream, err := foundLayer.Open()
	if err != nil {
		return err
	}

	err = w.writeTarEntry(tw, wl.Name, stream, wl.Layer.Size)
	if err != nil {
		return fmt.Errorf("Rewriting tar entry (%s): %s", wl.Name, err)
	}

	return tw.Flush()
}

func (w *TarWriter) writeTarEntry(tw *tar.Writer, path string, r io.Reader, size int64) error {
	var zerosFill bool

	if r == nil {
		zerosFill = true
		r = io.LimitReader(zeroReader{}, size)
	}

	hdr := &tar.Header{
		Mode:     0644,
		Typeflag: tar.TypeReg,
		Size:     size,
		Name:     path,
	}

	err := tw.WriteHeader(hdr)
	if err != nil {
		return fmt.Errorf("Writing header: %s", err)
	}

	t1 := time.Now()

	_, err = io.Copy(tw, r)
	if err != nil {
		return fmt.Errorf("Copying data: %s", err)
	}

	if !zerosFill {
		w.logger.Logf("done: file '%s' (%s)\n", path, time.Now().Sub(t1))
	}

	return nil
}

type zeroReader struct{}

func (r zeroReader) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}
