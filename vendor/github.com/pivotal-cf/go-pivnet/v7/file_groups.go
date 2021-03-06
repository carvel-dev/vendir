package pivnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type FileGroupsService struct {
	client Client
}

type createFileGroupBody struct {
	FileGroup createFileGroup `json:"file_group"`
}

type updateFileGroupBody struct {
	FileGroup updateFileGroup `json:"file_group"`
}

type createFileGroup struct {
	Name string `json:"name,omitempty"`
}

type updateFileGroup struct {
	Name string `json:"name,omitempty"`
}

type CreateFileGroupConfig struct {
 ProductSlug        string
 Name               string
}

type FileGroup struct {
	ID           int              `json:"id,omitempty" yaml:"id,omitempty"`
	Name         string           `json:"name,omitempty" yaml:"name,omitempty"`
	Product      FileGroupProduct `json:"product,omitempty" yaml:"product,omitempty"`
	ProductFiles []ProductFile    `json:"product_files,omitempty" yaml:"product_files,omitempty"`
}

type FileGroupProduct struct {
	ID   int    `json:"id,omitempty" yaml:"id,omitempty"`
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
}

type FileGroupsResponse struct {
	FileGroups []FileGroup `json:"file_groups,omitempty"`
}

func (e FileGroupsService) List(productSlug string) ([]FileGroup, error) {
	url := fmt.Sprintf("/products/%s/file_groups", productSlug)

	var response FileGroupsResponse
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

	return response.FileGroups, nil
}

func (p FileGroupsService) Get(productSlug string, fileGroupID int) (FileGroup, error) {
	url := fmt.Sprintf("/products/%s/file_groups/%d",
		productSlug,
		fileGroupID,
	)

	var response FileGroup
	resp, err := p.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return FileGroup{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return FileGroup{}, err
	}

	return response, nil
}

func (p FileGroupsService) Create(config CreateFileGroupConfig) (FileGroup, error) {
	url := fmt.Sprintf(
		"/products/%s/file_groups",
		config.ProductSlug,
	)

	createBody := createFileGroupBody{
		createFileGroup{
			Name: config.Name,
		},
	}

	b, err := json.Marshal(createBody)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return FileGroup{}, err
	}

	body := bytes.NewReader(b)

	var response FileGroup
	resp, err := p.client.MakeRequest(
		"POST",
		url,
		http.StatusCreated,
		body,
	)
	if err != nil {
		return FileGroup{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return FileGroup{}, err
	}

	return response, nil
}

func (p FileGroupsService) Update(productSlug string, fileGroup FileGroup) (FileGroup, error) {
	url := fmt.Sprintf(
		"/products/%s/file_groups/%d",
		productSlug,
		fileGroup.ID,
	)

	updateBody := updateFileGroupBody{
		updateFileGroup{
			Name: fileGroup.Name,
		},
	}

	b, err := json.Marshal(updateBody)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return FileGroup{}, err
	}

	body := bytes.NewReader(b)

	var response FileGroup
	resp, err := p.client.MakeRequest(
		"PATCH",
		url,
		http.StatusOK,
		body,
	)
	if err != nil {
		return FileGroup{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return FileGroup{}, err
	}

	return response, nil
}

func (p FileGroupsService) Delete(productSlug string, id int) (FileGroup, error) {
	url := fmt.Sprintf(
		"/products/%s/file_groups/%d",
		productSlug,
		id,
	)

	var response FileGroup
	resp, err := p.client.MakeRequest(
		"DELETE",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return FileGroup{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return FileGroup{}, err
	}

	return response, nil
}

func (p FileGroupsService) ListForRelease(productSlug string, releaseID int) ([]FileGroup, error) {
	url := fmt.Sprintf("/products/%s/releases/%d/file_groups",
		productSlug,
		releaseID,
	)

	var response FileGroupsResponse
	resp, err := p.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return []FileGroup{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return []FileGroup{}, err
	}

	return response.FileGroups, nil
}

func (r FileGroupsService) AddToRelease(
	productSlug string,
	releaseID int,
	fileGroupID int,
) error {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/add_file_group",
		productSlug,
		releaseID,
	)

	body := addRemoveFileGroupBody{
		FileGroup: addRemoveFileGroupBodyFileGroup{
			ID: fileGroupID,
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

func (r FileGroupsService) RemoveFromRelease(
	productSlug string,
	releaseID int,
	fileGroupID int,
) error {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/remove_file_group",
		productSlug,
		releaseID,
	)

	body := addRemoveFileGroupBody{
		FileGroup: addRemoveFileGroupBodyFileGroup{
			ID: fileGroupID,
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

type addRemoveFileGroupBody struct {
	FileGroup addRemoveFileGroupBodyFileGroup `json:"file_group"`
}

type addRemoveFileGroupBodyFileGroup struct {
	ID int `json:"id"`
}
