package pivnet

import (
	"fmt"
	"net/http"
	"encoding/json"
)

type ReleaseTypesService struct {
	client Client
}

type ReleaseType string

type ReleaseTypesResponse struct {
	ReleaseTypes []ReleaseType `json:"release_types" yaml:"release_types"`
}

func (r ReleaseTypesService) Get() ([]ReleaseType, error) {
	url := fmt.Sprintf("/releases/release_types")

	var response ReleaseTypesResponse
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

	return response.ReleaseTypes, nil
}
