package pivnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type UserGroupsService struct {
	client Client
}

type addRemoveUserGroupBody struct {
	UserGroup UserGroup `json:"user_group"`
}

type createUserGroupBody struct {
	UserGroup createUserGroup `json:"user_group"`
}

type updateUserGroupBody struct {
	UserGroup updateUserGroup `json:"user_group"`
}

type addRemoveMemberBody struct {
	Member member `json:"member"`
}

type UserGroupsResponse struct {
	UserGroups []UserGroup `json:"user_groups,omitempty"`
}

type UpdateUserGroupResponse struct {
	UserGroup UserGroup `json:"user_group,omitempty"`
}

type UserGroup struct {
	ID          int      `json:"id,omitempty" yaml:"id,omitempty"`
	Name        string   `json:"name,omitempty" yaml:"name,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Members     []string `json:"members,omitempty" yaml:"members,omitempty"`
	Admins      []string `json:"admins,omitempty" yaml:"admins,omitempty"`
}

type createUserGroup struct {
	ID          int      `json:"id,omitempty"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Members     []string `json:"members"` // do not omit empty to satisfy pivnet
}

type updateUserGroup struct {
	ID          int      `json:"id,omitempty"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Members     []string `json:"members,omitempty"`
}

type member struct {
	Email string `json:"email,omitempty"`
	Admin bool   `json:"admin,omitempty"`
}

func (u UserGroupsService) List() ([]UserGroup, error) {
	url := "/user_groups"

	var response UserGroupsResponse
	resp, err := u.client.MakeRequest(
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

	return response.UserGroups, nil
}

func (u UserGroupsService) ListForRelease(productSlug string, releaseID int) ([]UserGroup, error) {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/user_groups",
		productSlug,
		releaseID,
	)

	var response UserGroupsResponse
	resp, err := u.client.MakeRequest(
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

	return response.UserGroups, nil
}

func (u UserGroupsService) AddToRelease(productSlug string, releaseID int, userGroupID int) error {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/add_user_group",
		productSlug,
		releaseID,
	)

	body := addRemoveUserGroupBody{
		UserGroup: UserGroup{
			ID: userGroupID,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return err
	}

	resp, err := u.client.MakeRequest(
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

func (u UserGroupsService) RemoveFromRelease(productSlug string, releaseID int, userGroupID int) error {
	url := fmt.Sprintf(
		"/products/%s/releases/%d/remove_user_group",
		productSlug,
		releaseID,
	)

	body := addRemoveUserGroupBody{
		UserGroup: UserGroup{
			ID: userGroupID,
		},
	}

	b, err := json.Marshal(body)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return err
	}

	resp, err := u.client.MakeRequest(
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

func (u UserGroupsService) Get(userGroupID int) (UserGroup, error) {
	url := fmt.Sprintf("/user_groups/%d", userGroupID)

	var response UserGroup
	resp, err := u.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return UserGroup{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return UserGroup{}, err
	}

	return response, nil
}

func (u UserGroupsService) Create(name string, description string, members []string) (UserGroup, error) {
	url := "/user_groups"

	if members == nil {
		members = []string{}
	}

	createBody := createUserGroupBody{
		createUserGroup{
			Name:        name,
			Description: description,
			Members:     members,
		},
	}

	b, err := json.Marshal(createBody)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return UserGroup{}, err
	}

	body := bytes.NewReader(b)

	var response UserGroup
	resp, err := u.client.MakeRequest(
		"POST",
		url,
		http.StatusCreated,
		body,
	)
	if err != nil {
		return UserGroup{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return UserGroup{}, err
	}

	return response, nil
}

func (u UserGroupsService) Update(userGroup UserGroup) (UserGroup, error) {
	url := fmt.Sprintf("/user_groups/%d", userGroup.ID)

	createBody := updateUserGroupBody{
		updateUserGroup{
			Name:        userGroup.Name,
			Description: userGroup.Description,
		},
	}

	b, err := json.Marshal(createBody)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return UserGroup{}, err
	}

	body := bytes.NewReader(b)

	var response UpdateUserGroupResponse
	resp, err := u.client.MakeRequest(
		"PATCH",
		url,
		http.StatusOK,
		body,
	)
	if err != nil {
		return UserGroup{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return UserGroup{}, err
	}

	return response.UserGroup, nil
}

func (r UserGroupsService) Delete(userGroupID int) error {
	url := fmt.Sprintf("/user_groups/%d", userGroupID)

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

func (r UserGroupsService) AddMemberToGroup(
	userGroupID int,
	memberEmailAddress string,
	admin bool,
) (UserGroup, error) {
	url := fmt.Sprintf("/user_groups/%d/add_member", userGroupID)

	addRemoveMemberBody := addRemoveMemberBody{
		member{
			Email: memberEmailAddress,
			Admin: admin,
		},
	}

	b, err := json.Marshal(addRemoveMemberBody)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return UserGroup{}, err
	}

	body := bytes.NewReader(b)

	var response UpdateUserGroupResponse
	resp, err := r.client.MakeRequest(
		"PATCH",
		url,
		http.StatusOK,
		body,
	)
	if err != nil {
		return UserGroup{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return UserGroup{}, err
	}

	return response.UserGroup, nil
}

func (r UserGroupsService) RemoveMemberFromGroup(userGroupID int, memberEmailAddress string) (UserGroup, error) {
	url := fmt.Sprintf("/user_groups/%d/remove_member", userGroupID)

	addRemoveMemberBody := addRemoveMemberBody{
		member{
			Email: memberEmailAddress,
		},
	}

	b, err := json.Marshal(addRemoveMemberBody)
	if err != nil {
		// Untested as we cannot force an error because we are marshalling
		// a known-good body
		return UserGroup{}, err
	}

	body := bytes.NewReader(b)

	var response UpdateUserGroupResponse
	resp, err := r.client.MakeRequest(
		"PATCH",
		url,
		http.StatusOK,
		body,
	)
	if err != nil {
		return UserGroup{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return UserGroup{}, err
	}

	return response.UserGroup, nil
}
