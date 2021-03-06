package pivnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pivotal-cf/go-pivnet/v7/download"
	"github.com/pivotal-cf/go-pivnet/v7/logger"
)

type ProductFilesService struct {
	client Client
}

type CreateProductFileConfig struct {
	ProductSlug        string
	AWSObjectKey       string
	Description        string
	DocsURL            string
	FileType           string
	FileVersion        string
	IncludedFiles      []string
	SHA256             string
	MD5                string
	Name               string
	Platforms          []string
	ReleasedAt         string
	SystemRequirements []string
}

type ProductFilesResponse struct {
	ProductFiles []ProductFile `json:"product_files,omitempty"`
}

type ProductFileResponse struct {
	ProductFile ProductFile `json:"product_file,omitempty"`
}

type ProductFile struct {
	ID                 int      `json:"id,omitempty" yaml:"id,omitempty"`
	AWSObjectKey       string   `json:"aws_object_key,omitempty" yaml:"aws_object_key,omitempty"`
	Description        string   `json:"description,omitempty" yaml:"description,omitempty"`
	DocsURL            string   `json:"docs_url,omitempty" yaml:"docs_url,omitempty"`
	FileTransferStatus string   `json:"file_transfer_status,omitempty" yaml:"file_transfer_status,omitempty"`
	FileType           string   `json:"file_type,omitempty" yaml:"file_type,omitempty"`
	FileVersion        string   `json:"file_version,omitempty" yaml:"file_version,omitempty"`
	HasSignatureFile   bool     `json:"has_signature_file,omitempty" yaml:"has_signature_file,omitempty"`
	IncludedFiles      []string `json:"included_files,omitempty" yaml:"included_files,omitempty"`
	SHA256             string   `json:"sha256,omitempty" yaml:"sha256,omitempty"`
	MD5                string   `json:"md5,omitempty" yaml:"md5,omitempty"`
	Name               string   `json:"name,omitempty" yaml:"name,omitempty"`
	Platforms          []string `json:"platforms,omitempty" yaml:"platforms,omitempty"`
	ReadyToServe       bool     `json:"ready_to_serve,omitempty" yaml:"ready_to_serve,omitempty"`
	ReleasedAt         string   `json:"released_at,omitempty" yaml:"released_at,omitempty"`
	Size               int      `json:"size,omitempty" yaml:"size,omitempty"`
	SystemRequirements []string `json:"system_requirements,omitempty" yaml:"system_requirements,omitempty"`
	Links              *Links   `json:"_links,omitempty" yaml:"_links,omitempty"`
}

func (p ProductFile) DownloadLink() (string, error) {
	if p.Links == nil {
		return "", fmt.Errorf("Could not determine download link - links map is empty")
	}

	return p.Links.Download["href"], nil
}

const (
	FileTypeSoftware          = "Software"
	FileTypeDocumentation     = "Documentation"
	FileTypeOpenSourceLicense = "Open Source License"
)

func (p ProductFilesService) List(productSlug string) ([]ProductFile, error) {
	url := fmt.Sprintf("/products/%s/product_files", productSlug)

	var response ProductFilesResponse
	resp, err := p.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return []ProductFile{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return []ProductFile{}, err
	}

	return response.ProductFiles, nil
}

func (p ProductFilesService) ListForRelease(productSlug string, releaseID int) ([]ProductFile, error) {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/product_files",
		productSlug,
		releaseID,
	)

	var response ProductFilesResponse
	resp, err := p.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return []ProductFile{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return []ProductFile{}, err
	}

	return response.ProductFiles, nil
}

func (p ProductFilesService) Get(productSlug string, productFileID int) (ProductFile, error) {
	url := fmt.Sprintf(
		"/products/%s/product_files/%d",
		productSlug,
		productFileID,
	)

	var response ProductFileResponse
	resp, err := p.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return ProductFile{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return ProductFile{}, err
	}

	return response.ProductFile, nil
}

func (p ProductFilesService) GetForRelease(productSlug string, releaseID int, productFileID int) (ProductFile, error) {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/product_files/%d",
		productSlug,
		releaseID,
		productFileID,
	)

	var response ProductFileResponse
	resp, err := p.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return ProductFile{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return ProductFile{}, err
	}

	return response.ProductFile, nil
}

func (p ProductFilesService) Create(config CreateProductFileConfig) (ProductFile, error) {
	if config.AWSObjectKey == "" {
		return ProductFile{}, fmt.Errorf("AWS object key must not be empty")
	}

	url := fmt.Sprintf("/products/%s/product_files", config.ProductSlug)

	body := createUpdateProductFileBody{
		ProductFile: ProductFile{
			AWSObjectKey:       config.AWSObjectKey,
			Description:        config.Description,
			DocsURL:            config.DocsURL,
			FileType:           config.FileType,
			FileVersion:        config.FileVersion,
			IncludedFiles:      config.IncludedFiles,
			SHA256:             config.SHA256,
			MD5:                config.MD5,
			Name:               config.Name,
			Platforms:          config.Platforms,
			ReleasedAt:         config.ReleasedAt,
			SystemRequirements: config.SystemRequirements,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return ProductFile{}, err
	}

	var response ProductFileResponse
	resp, err := p.client.MakeRequest(
		"POST",
		url,
		http.StatusCreated,
		bytes.NewReader(b),
	)
	if err != nil {
		_, ok := err.(ErrTooManyRequests)
		if ok {
			return ProductFile{}, fmt.Errorf("You have hit the file creation limit. Please wait before creating more files. Contact pivnet-eng@pivotal.io with additional questions.")
		} else {
			return ProductFile{}, err
		}
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return ProductFile{}, err
	}

	return response.ProductFile, nil
}

func (p ProductFilesService) Update(productSlug string, productFile ProductFile) (ProductFile, error) {
	url := fmt.Sprintf("/products/%s/product_files/%d", productSlug, productFile.ID)

	body := createUpdateProductFileBody{
		ProductFile: ProductFile{
			Description:        productFile.Description,
			FileVersion:        productFile.FileVersion,
			SHA256:             productFile.SHA256,
			MD5:                productFile.MD5,
			Name:               productFile.Name,
			DocsURL:            productFile.DocsURL,
			SystemRequirements: productFile.SystemRequirements,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return ProductFile{}, err
	}

	var response ProductFileResponse
	resp, err := p.client.MakeRequest(
		"PATCH",
		url,
		http.StatusOK,
		bytes.NewReader(b),
	)
	if err != nil {
		return ProductFile{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return ProductFile{}, err
	}

	return response.ProductFile, nil
}

type createUpdateProductFileBody struct {
	ProductFile ProductFile `json:"product_file"`
}

func (p ProductFilesService) Delete(productSlug string, id int) (ProductFile, error) {
	url := fmt.Sprintf(
		"/products/%s/product_files/%d",
		productSlug,
		id,
	)

	var response ProductFileResponse
	resp, err := p.client.MakeRequest(
		"DELETE",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return ProductFile{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return ProductFile{}, err
	}

	return response.ProductFile, nil
}

func (p ProductFilesService) AddToRelease(
	productSlug string,
	releaseID int,
	productFileID int,
) error {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/add_product_file",
		productSlug,
		releaseID,
	)

	body := createUpdateProductFileBody{
		ProductFile: ProductFile{
			ID: productFileID,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return err
	}

	resp, err := p.client.MakeRequest(
		"PATCH",
		url,
		http.StatusNoContent,
		bytes.NewReader(b),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (p ProductFilesService) RemoveFromRelease(
	productSlug string,
	releaseID int,
	productFileID int,
) error {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/remove_product_file",
		productSlug,
		releaseID,
	)

	body := createUpdateProductFileBody{
		ProductFile: ProductFile{
			ID: productFileID,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return err
	}

	resp, err := p.client.MakeRequest(
		"PATCH",
		url,
		http.StatusNoContent,
		bytes.NewReader(b),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (p ProductFilesService) AddToFileGroup(
	productSlug string,
	fileGroupID int,
	productFileID int,
) error {
	url := fmt.Sprintf(
		"/products/%s/file_groups/%d/add_product_file",
		productSlug,
		fileGroupID,
	)

	body := createUpdateProductFileBody{
		ProductFile: ProductFile{
			ID: productFileID,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return err
	}

	resp, err := p.client.MakeRequest(
		"PATCH",
		url,
		http.StatusNoContent,
		bytes.NewReader(b),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (p ProductFilesService) RemoveFromFileGroup(
	productSlug string,
	fileGroupID int,
	productFileID int,
) error {
	url := fmt.Sprintf(
		"/products/%s/file_groups/%d/remove_product_file",
		productSlug,
		fileGroupID,
	)

	body := createUpdateProductFileBody{
		ProductFile: ProductFile{
			ID: productFileID,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return err
	}

	resp, err := p.client.MakeRequest(
		"PATCH",
		url,
		http.StatusNoContent,
		bytes.NewReader(b),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (p ProductFilesService) DownloadForRelease(
	location *download.FileInfo,
	productSlug string,
	releaseID int,
	productFileID int,
	progressWriter io.Writer,
) error {
	pf, err := p.GetForRelease(
		productSlug,
		releaseID,
		productFileID,
	)
	if err != nil {
		return fmt.Errorf("GetForRelease: %s", err)
	}

	downloadLink, err := pf.DownloadLink()
	if err != nil {
		return fmt.Errorf("DownloadLink: %s", err)
	}

	p.client.logger.Debug("Downloading file", logger.Data{"downloadLink": downloadLink})

	productFileDownloadLinkFetcher := NewProductFileLinkFetcher(downloadLink, p.client)

	p.client.downloader.Bar = download.NewBar()

	err = p.client.downloader.Get(
		location,
		productFileDownloadLinkFetcher,
		progressWriter,
	)
	if err != nil {
		return fmt.Errorf("Downloader.Get: %s", err)
	}

	return nil
}
