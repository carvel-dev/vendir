package pivnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type DependencySpecifiersService struct {
	client Client
}

type DependencySpecifiersResponse struct {
	DependencySpecifiers []DependencySpecifier `json:"dependency_specifiers,omitempty"`
}

type DependencySpecifierResponse struct {
	DependencySpecifier DependencySpecifier `json:"dependency_specifier,omitempty"`
}

type DependencySpecifier struct {
	ID        int     `json:"id,omitempty" yaml:"id,omitempty"`
	Product   Product `json:"product,omitempty" yaml:"product,omitempty"`
	Specifier string  `json:"specifier,omitempty" yaml:"specifier,omitempty"`
}

func (r DependencySpecifiersService) List(productSlug string, releaseID int) ([]DependencySpecifier, error) {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/dependency_specifiers",
		productSlug,
		releaseID,
	)

	var response DependencySpecifiersResponse
	resp, err := r.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.DependencySpecifiers, nil
}

func (r DependencySpecifiersService) Get(productSlug string, releaseID int, dependencySpecifierID int) (DependencySpecifier, error) {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/dependency_specifiers/%d",
		productSlug,
		releaseID,
		dependencySpecifierID,
	)

	resp, err := r.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return DependencySpecifier{}, err
	}
	defer resp.Body.Close()

	var response DependencySpecifierResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return DependencySpecifier{}, err
	}

	return response.DependencySpecifier, nil
}

func (r DependencySpecifiersService) Create(
	productSlug string,
	releaseID int,
	dependentProductSlug string,
	specifier string,
) (DependencySpecifier, error) {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/dependency_specifiers",
		productSlug,
		releaseID,
	)

	body := createDependencySpecifierBody{
		createDependencySpecifierBodyDependencySpecifier{
			ProductSlug: dependentProductSlug,
			Specifier:   specifier,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return DependencySpecifier{}, err
	}

	resp, err := r.client.MakeRequest(
		"POST",
		url,
		http.StatusCreated,
		bytes.NewReader(b),
	)
	if err != nil {
		return DependencySpecifier{}, err
	}
	defer resp.Body.Close()

	var response DependencySpecifierResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return DependencySpecifier{}, err
	}

	return response.DependencySpecifier, nil
}

func (r DependencySpecifiersService) Delete(
	productSlug string,
	releaseID int,
	dependencySpecifierID int,
) error {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/dependency_specifiers/%d",
		productSlug,
		releaseID,
		dependencySpecifierID,
	)

	resp, err := r.client.MakeRequest(
		"DELETE",
		url,
		http.StatusNoContent,
		nil,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

type createDependencySpecifierBody struct {
	DependencySpecifier createDependencySpecifierBodyDependencySpecifier `json:"dependency_specifier"`
}

type createDependencySpecifierBodyDependencySpecifier struct {
	ProductSlug string `json:"product_slug"`
	Specifier   string `json:"specifier"`
}
