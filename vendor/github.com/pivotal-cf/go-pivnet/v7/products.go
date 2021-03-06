package pivnet

import (
	"fmt"
	"net/http"

	"encoding/json"
	"github.com/pivotal-cf/go-pivnet/v7/logger"
)

type ProductsService struct {
	client Client
	l      logger.Logger
}

type S3Directory struct {
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
}

type Product struct {
	ID            int          `json:"id,omitempty" yaml:"id,omitempty"`
	Slug          string       `json:"slug,omitempty" yaml:"slug,omitempty"`
	Name          string       `json:"name,omitempty" yaml:"name,omitempty"`
	S3Directory   *S3Directory `json:"s3_directory,omitempty" yaml:"s3_directory,omitempty"`
	InstallsOnPks bool         `json:"installs_on_pks,omitempty" yaml:"installs_on_pks,omitempty"`
}

type SlugAliasResponse struct {
	Slugs       []string `json:"slugs" yaml:"slugs"`
	CurrentSlug string   `json:"current_slug" yaml:"current_slug"`
}

type ProductsResponse struct {
	Products []Product `json:"products,omitempty"`
}

func (p ProductsService) List() ([]Product, error) {
	url := "/products"

	var response ProductsResponse
	resp, err := p.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return []Product{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return []Product{}, err
	}

	return response.Products, nil
}

func (p ProductsService) Get(slug string) (Product, error) {
	url := fmt.Sprintf("/products/%s", slug)

	var response Product
	resp, err := p.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return Product{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return Product{}, err
	}

	return response, nil
}

func (p ProductsService) SlugAlias(slug string) (SlugAliasResponse, error) {
	url := fmt.Sprintf("/products/%s/slug_alias", slug)

	var response SlugAliasResponse
	resp, err := p.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return SlugAliasResponse{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return SlugAliasResponse{}, err
	}

	return response, nil
}
