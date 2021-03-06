package pivnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pivotal-cf/go-pivnet/v7/logger"
)

type ReleasesService struct {
	client Client
	l      logger.Logger
}

type createReleaseBody struct {
	Release      Release `json:"release"`
	CopyMetadata bool    `json:"copy_metadata"`
}

type ReleasesResponse struct {
	Releases []Release `json:"releases,omitempty"`
}

type CreateReleaseResponse struct {
	Release Release `json:"release,omitempty"`
}

type Release struct {
	ID                     int         `json:"id,omitempty" yaml:"id,omitempty"`
	Availability           string      `json:"availability,omitempty" yaml:"availability,omitempty"`
	EULA                   *EULA       `json:"eula,omitempty" yaml:"eula,omitempty"`
	OSSCompliant           string      `json:"oss_compliant,omitempty" yaml:"oss_compliant,omitempty"`
	ReleaseDate            string      `json:"release_date,omitempty" yaml:"release_date,omitempty"`
	ReleaseType            ReleaseType `json:"release_type,omitempty" yaml:"release_type,omitempty"`
	Version                string      `json:"version,omitempty" yaml:"version,omitempty"`
	Links                  *Links      `json:"_links,omitempty" yaml:"_links,omitempty"`
	Description            string      `json:"description,omitempty" yaml:"description,omitempty"`
	ReleaseNotesURL        string      `json:"release_notes_url,omitempty" yaml:"release_notes_url,omitempty"`
	Controlled             bool        `json:"controlled,omitempty" yaml:"controlled,omitempty"`
	ECCN                   string      `json:"eccn,omitempty" yaml:"eccn,omitempty"`
	LicenseException       string      `json:"license_exception,omitempty" yaml:"license_exception,omitempty"`
	EndOfSupportDate       string      `json:"end_of_support_date,omitempty" yaml:"end_of_support_date,omitempty"`
	EndOfGuidanceDate      string      `json:"end_of_guidance_date,omitempty" yaml:"end_of_guidance_date,omitempty"`
	EndOfAvailabilityDate  string      `json:"end_of_availability_date,omitempty" yaml:"end_of_availability_date,omitempty"`
	UpdatedAt              string      `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
	SoftwareFilesUpdatedAt string      `json:"software_files_updated_at,omitempty" yaml:"software_files_updated_at,omitempty"`
	UserGroupsUpdatedAt    string      `json:"user_groups_updated_at,omitempty" yaml:"user_groups_updated_at,omitempty"`
}

type CreateReleaseConfig struct {
	ProductSlug           string
	Version               string
	ReleaseType           string
	ReleaseDate           string
	EULASlug              string
	Description           string
	ReleaseNotesURL       string
	Controlled            bool
	ECCN                  string
	LicenseException      string
	EndOfSupportDate      string
	EndOfGuidanceDate     string
	EndOfAvailabilityDate string
	CopyMetadata          bool
}

func (r ReleasesService) List(productSlug string, params ...QueryParameter) ([]Release, error) {
	url := fmt.Sprintf("/products/%s/releases", productSlug)

	var response ReleasesResponse
	resp, err := r.client.MakeRequestWithParams("GET", url, http.StatusOK, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.Releases, nil
}

func (r ReleasesService) Get(productSlug string, releaseID int) (Release, error) {
	url := fmt.Sprintf("/products/%s/releases/%d", productSlug, releaseID)

	var response Release
	resp, err := r.client.MakeRequest("GET", url, http.StatusOK, nil)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return Release{}, err
	}

	return response, nil
}

func (r ReleasesService) Create(config CreateReleaseConfig) (Release, error) {
	url := fmt.Sprintf("/products/%s/releases", config.ProductSlug)

	body := createReleaseBody{
		Release: Release{
			Availability: "Admins Only",
			EULA: &EULA{
				Slug: config.EULASlug,
			},
			OSSCompliant:          "confirm",
			ReleaseDate:           config.ReleaseDate,
			ReleaseType:           ReleaseType(config.ReleaseType),
			Version:               config.Version,
			Description:           config.Description,
			ReleaseNotesURL:       config.ReleaseNotesURL,
			Controlled:            config.Controlled,
			ECCN:                  config.ECCN,
			LicenseException:      config.LicenseException,
			EndOfSupportDate:      config.EndOfSupportDate,
			EndOfGuidanceDate:     config.EndOfGuidanceDate,
			EndOfAvailabilityDate: config.EndOfAvailabilityDate,
		},
		CopyMetadata: config.CopyMetadata,
	}

	if config.ReleaseDate == "" {
		body.Release.ReleaseDate = time.Now().Format("2006-01-02")
		r.l.Info(
			"No release date found - using default release date",
			logger.Data{"release date": body.Release.ReleaseDate})
	}

	b, err := json.Marshal(body)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return Release{}, err
	}

	var response CreateReleaseResponse
	resp, err := r.client.MakeRequest(
		"POST",
		url,
		http.StatusCreated,
		bytes.NewReader(b),
	)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return Release{}, err
	}

	return response.Release, nil
}

func (r ReleasesService) Update(productSlug string, release Release) (Release, error) {
	url := fmt.Sprintf(
		"/products/%s/releases/%d",
		productSlug,
		release.ID,
	)

	release.OSSCompliant = "confirm"

	var updatedRelease = createReleaseBody{
		Release: release,
	}

	body, err := json.Marshal(updatedRelease)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return Release{}, err
	}

	var response CreateReleaseResponse
	resp, err := r.client.MakeRequest(
		"PATCH",
		url,
		http.StatusOK,
		bytes.NewReader(body),
	)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return Release{}, err
	}

	return response.Release, nil
}

func (r ReleasesService) Delete(productSlug string, release Release) error {
	url := fmt.Sprintf(
		"/products/%s/releases/%d",
		productSlug,
		release.ID,
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
