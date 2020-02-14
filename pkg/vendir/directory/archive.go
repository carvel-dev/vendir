package directory

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Archive struct {
	path string
}

func (t Archive) Unpack(dstPath string) error {
	contentExtractorFuncs := []func(string, string) (bool, error){
		t.tryZip,
		t.tryTgz,
		t.tryTar,
	}

	for _, f := range contentExtractorFuncs {
		ok, err := f(t.path, dstPath)
		if ok {
			return err
		}
	}

	return fmt.Errorf("Expected known archive type (zip, tgz, tar)")
}

func (t Archive) writeIntoFile(srcFile io.Reader, dstPath, additionalPath string) error {
	dstFilePath := filepath.Join(dstPath, additionalPath)

	err := os.MkdirAll(filepath.Dir(dstFilePath), 0700)
	if err != nil {
		return fmt.Errorf("Making intermediate dir: %s", err)
	}

	dstFile, err := os.Create(dstFilePath)
	if err != nil {
		return fmt.Errorf("Creating dst file: %s", err)
	}

	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("Copying into dst file: %s", err)
	}

	return nil
}

func (t Archive) writeIntoFileAndClose(srcFile io.ReadCloser, dstPath, additionalPath string) error {
	defer srcFile.Close()
	return t.writeIntoFile(srcFile, dstPath, additionalPath)
}

func (t Archive) tryZip(path, dstPath string) (bool, error) {
	zipArchive, err := zip.OpenReader(path)
	if err != nil {
		return false, fmt.Errorf("Opening zip archive: %s", err)
	}

	defer zipArchive.Close()

	for _, f := range zipArchive.File {
		if strings.HasSuffix(f.Name, "/") {
			// TODO should we make empty directories?
			continue
		}

		srcZipFile, err := f.Open()
		if err != nil {
			return true, fmt.Errorf("Opening zip file: %s", err)
		}

		err = t.writeIntoFileAndClose(srcZipFile, dstPath, f.Name)
		if err != nil {
			return true, err
		}
	}

	return true, nil
}

func (t Archive) tryTgz(path, dstPath string) (bool, error) {
	return t.tryTarWithGzip(path, dstPath, true)
}

func (t Archive) tryTar(path, dstPath string) (bool, error) {
	return t.tryTarWithGzip(path, dstPath, false)
}

func (t Archive) tryTarWithGzip(path, dstPath string, gzipped bool) (bool, error) {
	plainFile, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("Opening archive: %s", err)
	}

	defer plainFile.Close()

	var fileReader io.Reader

	if gzipped {
		gzipFile, err := gzip.NewReader(plainFile)
		if err != nil {
			return false, fmt.Errorf("Opening gzip archive: %s", err)
		}
		fileReader = gzipFile
	} else {
		fileReader = plainFile
	}

	tarReader := tar.NewReader(fileReader)
	firstFile := true

	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return firstFile, fmt.Errorf("Reading next tar header: %s", err)
		}

		firstFile = false

		switch header.Typeflag {
		case tar.TypeDir:
			// TODO should we make empty directories?
			continue

		case tar.TypeReg:
			err = t.writeIntoFile(tarReader, dstPath, header.Name)
			if err != nil {
				return true, err
			}

		default:
			return false, fmt.Errorf("Unknown file '%s' (%d)", header.Name, header.Typeflag)
		}
	}

	return true, nil
}
