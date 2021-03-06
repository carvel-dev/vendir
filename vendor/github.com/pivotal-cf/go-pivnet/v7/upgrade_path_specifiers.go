package pivnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type UpgradePathSpecifiersService struct {
	client Client
}

type UpgradePathSpecifiersResponse struct {
	UpgradePathSpecifiers []UpgradePathSpecifier `json:"upgrade_path_specifiers,omitempty"`
}

type UpgradePathSpecifierResponse struct {
	UpgradePathSpecifier UpgradePathSpecifier `json:"upgrade_path_specifier,omitempty"`
}

type UpgradePathSpecifier struct {
	ID        int    `json:"id,omitempty" yaml:"id,omitempty"`
	Specifier string `json:"specifier,omitempty" yaml:"specifier,omitempty"`
}

func (r UpgradePathSpecifiersService) List(productSlug string, releaseID int) ([]UpgradePathSpecifier, error) {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/upgrade_path_specifiers",
		productSlug,
		releaseID,
	)

	var response UpgradePathSpecifiersResponse
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

	return response.UpgradePathSpecifiers, nil
}

func (r UpgradePathSpecifiersService) Get(productSlug string, releaseID int, upgradePathSpecifierID int) (UpgradePathSpecifier, error) {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/upgrade_path_specifiers/%d",
		productSlug,
		releaseID,
		upgradePathSpecifierID,
	)

	resp, err := r.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return UpgradePathSpecifier{}, err
	}
	defer resp.Body.Close()

	var response UpgradePathSpecifierResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return UpgradePathSpecifier{}, err
	}

	return response.UpgradePathSpecifier, nil
}

func (r UpgradePathSpecifiersService) Create(productSlug string, releaseID int, specifier string) (UpgradePathSpecifier, error) {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/upgrade_path_specifiers",
		productSlug,
		releaseID,
	)

	body := createUpgradePathSpecifierBody{
		createUpgradePathSpecifierBodyUpgradePathSpecifier{
			Specifier: specifier,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return UpgradePathSpecifier{}, err
	}

	resp, err := r.client.MakeRequest(
		"POST",
		url,
		http.StatusCreated,
		bytes.NewReader(b),
	)
	if err != nil {
		return UpgradePathSpecifier{}, err
	}
	defer resp.Body.Close()

	var response UpgradePathSpecifierResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return UpgradePathSpecifier{}, err
	}

	return response.UpgradePathSpecifier, nil
}

func (r UpgradePathSpecifiersService) Delete(
	productSlug string,
	releaseID int,
	upgradePathSpecifierID int,
) error {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/upgrade_path_specifiers/%d",
		productSlug,
		releaseID,
		upgradePathSpecifierID,
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

type createUpgradePathSpecifierBody struct {
	UpgradePathSpecifier createUpgradePathSpecifierBodyUpgradePathSpecifier `json:"upgrade_path_specifier"`
}

type createUpgradePathSpecifierBodyUpgradePathSpecifier struct {
	Specifier string `json:"specifier"`
}
