package pivnet

import (
	"net/http"

	"encoding/json"
)

type PivnetVersionsService struct {
	client Client
}

type PivnetVersions struct {
	PivnetCliVersion       string  `json:"pivnet_cli,omitempty"`
	PivnetResourceVersion  string  `json:"pivnet_resource,omitempty"`
}

func (v PivnetVersionsService) List() (PivnetVersions, error) {
	url := "/versions"

	var response PivnetVersions
	resp, err := v.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return PivnetVersions{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return PivnetVersions{}, err
	}

	return response, nil
}

