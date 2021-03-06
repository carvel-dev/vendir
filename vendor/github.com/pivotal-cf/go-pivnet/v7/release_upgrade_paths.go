package pivnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type ReleaseUpgradePathsService struct {
	client Client
}

type ReleaseUpgradePathsResponse struct {
	ReleaseUpgradePaths []ReleaseUpgradePath `json:"upgrade_paths,omitempty"`
}

type ReleaseUpgradePath struct {
	Release UpgradePathRelease `json:"release,omitempty" yaml:"release,omitempty"`
}

type UpgradePathRelease struct {
	ID      int    `json:"id,omitempty" yaml:"id,omitempty"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

func (r ReleaseUpgradePathsService) Get(productSlug string, releaseID int) ([]ReleaseUpgradePath, error) {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/upgrade_paths",
		productSlug,
		releaseID,
	)

	var response ReleaseUpgradePathsResponse
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

	return response.ReleaseUpgradePaths, nil
}

func (r ReleaseUpgradePathsService) Add(
	productSlug string,
	releaseID int,
	previousReleaseID int,
) error {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/add_upgrade_path",
		productSlug,
		releaseID,
	)

	body := addRemoveUpgradePathBody{
		UpgradePath: addRemoveUpgradePathBodyUpgradePath{
			ReleaseID: previousReleaseID,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return err
	}

	resp, err := r.client.MakeRequest(
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

func (r ReleaseUpgradePathsService) Remove(
	productSlug string,
	releaseID int,
	previousReleaseID int,
) error {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/remove_upgrade_path",
		productSlug,
		releaseID,
	)

	body := addRemoveUpgradePathBody{
		UpgradePath: addRemoveUpgradePathBodyUpgradePath{
			ReleaseID: previousReleaseID,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return err
	}

	resp, err := r.client.MakeRequest(
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

type addRemoveUpgradePathBody struct {
	UpgradePath addRemoveUpgradePathBodyUpgradePath `json:"upgrade_path"`
}

type addRemoveUpgradePathBodyUpgradePath struct {
	ReleaseID int `json:"release_id"`
}
