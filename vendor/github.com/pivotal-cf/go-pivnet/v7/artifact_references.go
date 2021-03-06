package pivnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type ArtifactReferencesService struct {
	client Client
}

type CreateArtifactReferenceConfig struct {
	ProductSlug        string
	Description        string
	DocsURL            string
	Digest             string
	Name               string
	ArtifactPath       string
	SystemRequirements []string
}

type ArtifactReferencesResponse struct {
	ArtifactReferences []ArtifactReference `json:"artifact_references,omitempty"`
}

type ArtifactReferenceResponse struct {
	ArtifactReference ArtifactReference `json:"artifact_reference,omitempty"`
}

type ReplicationStatus string

const (
	InProgress        ReplicationStatus = "in_progress"
	Complete          ReplicationStatus = "complete"
	FailedToReplicate ReplicationStatus = "failed_to_replicate"
)

type ArtifactReference struct {
	ID                 int               `json:"id,omitempty" yaml:"id,omitempty"`
	ArtifactPath       string            `json:"artifact_path,omitempty" yaml:"artifact_path,omitempty"`
	Description        string            `json:"description,omitempty" yaml:"description,omitempty"`
	Digest             string            `json:"digest,omitempty" yaml:"digest,omitempty"`
	DocsURL            string            `json:"docs_url,omitempty" yaml:"docs_url,omitempty"`
	Name               string            `json:"name,omitempty" yaml:"name,omitempty"`
	SystemRequirements []string          `json:"system_requirements,omitempty" yaml:"system_requirements,omitempty"`
	ReleaseVersions    []string          `json:"release_versions,omitempty" yaml:"release_versions,omitempty"`
	ReplicationStatus  ReplicationStatus `json:"replication_status,omitempty" yaml:"replication_status,omitempty"`
}

type createUpdateArtifactReferenceBody struct {
	ArtifactReference ArtifactReference `json:"artifact_reference"`
}

func (p ArtifactReferencesService) List(productSlug string) ([]ArtifactReference, error) {
	url := fmt.Sprintf("/products/%s/artifact_references", productSlug)

	var response ArtifactReferencesResponse
	resp, err := p.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return []ArtifactReference{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return []ArtifactReference{}, err
	}

	return response.ArtifactReferences, nil
}

func (p ArtifactReferencesService) ListForDigest(productSlug string, digest string) ([]ArtifactReference, error) {
	url := fmt.Sprintf("/products/%s/artifact_references", productSlug)
	params := []QueryParameter{
		{"digest", digest},
	}

	var response ArtifactReferencesResponse
	resp, err := p.client.MakeRequestWithParams(
		"GET",
		url,
		http.StatusOK,
		params,
		nil,
	)
	if err != nil {
		return []ArtifactReference{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return []ArtifactReference{}, err
	}

	return response.ArtifactReferences, nil
}

func (p ArtifactReferencesService) ListForRelease(productSlug string, releaseID int) ([]ArtifactReference, error) {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/artifact_references",
		productSlug,
		releaseID,
	)

	var response ArtifactReferencesResponse
	resp, err := p.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return []ArtifactReference{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return []ArtifactReference{}, err
	}

	return response.ArtifactReferences, nil
}

func (p ArtifactReferencesService) Get(productSlug string, artifactReferenceID int) (ArtifactReference, error) {
	url := fmt.Sprintf(
		"/products/%s/artifact_references/%d",
		productSlug,
		artifactReferenceID,
	)

	var response ArtifactReferenceResponse
	resp, err := p.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return ArtifactReference{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return ArtifactReference{}, err
	}

	return response.ArtifactReference, nil
}

func (p ArtifactReferencesService) Update(productSlug string, artifactReference ArtifactReference) (ArtifactReference, error) {
	url := fmt.Sprintf("/products/%s/artifact_references/%d", productSlug, artifactReference.ID)

	body := createUpdateArtifactReferenceBody{
		ArtifactReference: ArtifactReference{
			Description:        artifactReference.Description,
			Name:               artifactReference.Name,
			DocsURL:            artifactReference.DocsURL,
			SystemRequirements: artifactReference.SystemRequirements,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		return ArtifactReference{}, err
	}

	var response ArtifactReferenceResponse
	resp, err := p.client.MakeRequest(
		"PATCH",
		url,
		http.StatusOK,
		bytes.NewReader(b),
	)
	if err != nil {
		return ArtifactReference{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return ArtifactReference{}, err
	}

	return response.ArtifactReference, nil
}

func (p ArtifactReferencesService) GetForRelease(productSlug string, releaseID int, artifactReferenceID int) (ArtifactReference, error) {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/artifact_references/%d",
		productSlug,
		releaseID,
		artifactReferenceID,
	)

	var response ArtifactReferenceResponse
	resp, err := p.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return ArtifactReference{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return ArtifactReference{}, err
	}

	return response.ArtifactReference, nil
}

func (p ArtifactReferencesService) Create(config CreateArtifactReferenceConfig) (ArtifactReference, error) {
	url := fmt.Sprintf("/products/%s/artifact_references", config.ProductSlug)

	body := createUpdateArtifactReferenceBody{
		ArtifactReference: ArtifactReference{
			ArtifactPath:       config.ArtifactPath,
			Description:        config.Description,
			Digest:             config.Digest,
			DocsURL:            config.DocsURL,
			Name:               config.Name,
			SystemRequirements: config.SystemRequirements,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return ArtifactReference{}, err
	}

	var response ArtifactReferenceResponse
	resp, err := p.client.MakeRequest(
		"POST",
		url,
		http.StatusCreated,
		bytes.NewReader(b),
	)
	if err != nil {
		_, ok := err.(ErrTooManyRequests)
		if ok {
			return ArtifactReference{}, fmt.Errorf("You have hit the artifact reference creation limit. Please wait before creating more artifact references. Contact pivnet-eng@pivotal.io with additional questions.")
		} else {
			return ArtifactReference{}, err
		}
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return ArtifactReference{}, err
	}

	return response.ArtifactReference, nil
}

func (p ArtifactReferencesService) Delete(productSlug string, id int) (ArtifactReference, error) {
	url := fmt.Sprintf(
		"/products/%s/artifact_references/%d",
		productSlug,
		id,
	)

	var response ArtifactReferenceResponse
	resp, err := p.client.MakeRequest(
		"DELETE",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return ArtifactReference{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return ArtifactReference{}, err
	}

	return response.ArtifactReference, nil
}

func (p ArtifactReferencesService) AddToRelease(
	productSlug string,
	releaseID int,
	artifactReferenceID int,
) error {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/add_artifact_reference",
		productSlug,
		releaseID,
	)

	body := createUpdateArtifactReferenceBody{
		ArtifactReference: ArtifactReference{
			ID: artifactReferenceID,
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

func (p ArtifactReferencesService) RemoveFromRelease(
	productSlug string,
	releaseID int,
	artifactReferenceID int,
) error {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/remove_artifact_reference",
		productSlug,
		releaseID,
	)

	body := createUpdateArtifactReferenceBody{
		ArtifactReference: ArtifactReference{
			ID: artifactReferenceID,
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
