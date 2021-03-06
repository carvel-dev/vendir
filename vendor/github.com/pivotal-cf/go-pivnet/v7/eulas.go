package pivnet

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type EULAsService struct {
	client Client
}

type EULA struct {
	Slug        string  `json:"slug,omitempty" yaml:"slug,omitempty"`
	ID          int     `json:"id,omitempty" yaml:"id,omitempty"`
	Name        string  `json:"name,omitempty" yaml:"name,omitempty"`
	Content     string  `json:"content,omitempty" yaml:"content,omitempty"`
	ArchivedAt  string  `json:"archived_at,omitempty" yaml:"archived_at,omitempty"`
	Links       *Links  `json:"_links,omitempty" yaml:"_links,omitempty"`
}

type EULAsResponse struct {
	EULAs []EULA `json:"eulas,omitempty"`
	Links *Links `json:"_links,omitempty"`
}

type EULAAcceptanceResponse struct {
	AcceptedAt string `json:"accepted_at,omitempty"`
	Links      *Links `json:"_links,omitempty"`
}

func (e EULAsService) List() ([]EULA, error) {
	url := "/eulas"

	var response EULAsResponse
	resp, err := e.client.MakeRequest(
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

	return response.EULAs, nil
}

func (e EULAsService) Get(eulaSlug string) (EULA, error) {
	url := fmt.Sprintf("/eulas/%s", eulaSlug)

	var response EULA
	resp, err := e.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return EULA{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return EULA{}, err
	}

	return response, nil
}

func (e EULAsService) Accept(productSlug string, releaseID int) error {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/pivnet_resource_eula_acceptance",
		productSlug,
		releaseID,
	)

	resp, err := e.client.MakeRequest(
		"POST",
		url,
		http.StatusOK,
		strings.NewReader(`{}`),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
