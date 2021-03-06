package pivnet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type SubscriptionGroupsService struct {
	client Client
}

type SubscriptionGroupsResponse struct {
	SubscriptionGroups []SubscriptionGroup `json:"subscription_groups,omitempty"`
}

type SubscriptionGroupMember struct {
	ID      int    `json:"id,omitempty" yaml:"id,omitempty"`
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	Email   string `json:"email,omitempty" yaml:"email,omitempty"`
	IsAdmin bool   `json:"admin" yaml:"admin"`
}

type SubscriptionGroupMemberNoAdmin struct {
	ID    int    `json:"id,omitempty" yaml:"id,omitempty"`
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

type SubscriptionGroupMemberEmail struct {
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

type subscriptionGroupMemberToAdd struct {
	Member SubscriptionGroupMember `json:"member,omitempty"`
}

type subscriptionGroupMemberNoAdminToAdd struct {
	Member SubscriptionGroupMemberNoAdmin `json:"member,omitempty"`
}

type subscriptionGroupMemberToRemove struct {
	Member SubscriptionGroupMemberEmail `json:"member"`
}

type SubscriptionGroup struct {
	ID                 int                             `json:"id,omitempty" yaml:"id,omitempty"`
	Name               string                          `json:"name,omitempty" yaml:"name,omitempty"`
	Members            []SubscriptionGroupMember       `json:"members" yaml:"members"`
	PendingInvitations []string                        `json:"pending_invitations" yaml:"pending_invitations"`
	Subscriptions      []SubscriptionGroupSubscription `json:"subscriptions" yaml:"subscriptions"`
}

type SubscriptionGroupSubscription struct {
	ID   int    `json:"id,omitempty" yaml:"id,omitempty"`
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
}

func (c SubscriptionGroupsService) List() ([]SubscriptionGroup, error) {
	url := "/subscription_groups"

	var response SubscriptionGroupsResponse
	resp, err := c.client.MakeRequest(
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

	return response.SubscriptionGroups, nil
}

func (c SubscriptionGroupsService) Get(subscriptionGroupID int) (SubscriptionGroup, error) {
	url := fmt.Sprintf("/subscription_groups/%d", subscriptionGroupID)

	var response SubscriptionGroup
	resp, err := c.client.MakeRequest(
		"GET",
		url,
		http.StatusOK,
		nil,
	)
	if err != nil {
		return SubscriptionGroup{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return SubscriptionGroup{}, err
	}

	return response, nil
}

func (c SubscriptionGroupsService) AddMember(
	subscriptionGroupID int,
	memberEmailAddress string,
	isAdmin string,
) (SubscriptionGroup, error) {
	url := fmt.Sprintf("/subscription_groups/%d/add_member", subscriptionGroupID)

	var b []byte
	var err error

	if len(strings.TrimSpace(isAdmin)) == 0 {
		addSubscriptionGroupMemberBody := subscriptionGroupMemberNoAdminToAdd{
			SubscriptionGroupMemberNoAdmin{
				Email: memberEmailAddress,
			},
		}

		b, err = json.Marshal(addSubscriptionGroupMemberBody)
		if err != nil {
			return SubscriptionGroup{}, err
		}
	} else {
		isAdmin, err := strconv.ParseBool(isAdmin)
		if err != nil {
			return SubscriptionGroup{}, errors.New("parameter admin should be true or false")
		}

		addSubscriptionGroupMemberBody := subscriptionGroupMemberToAdd{
			SubscriptionGroupMember{
				Email:   memberEmailAddress,
				IsAdmin: isAdmin,
			},
		}

		b, err = json.Marshal(addSubscriptionGroupMemberBody)
		if err != nil {
			return SubscriptionGroup{}, err
		}
	}

	body := bytes.NewReader(b)

	var response SubscriptionGroup
	resp, err := c.client.MakeRequest(
		"PATCH",
		url,
		http.StatusOK,
		body,
	)
	if err != nil {
		return SubscriptionGroup{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return SubscriptionGroup{}, err
	}

	return response, nil
}

func (c SubscriptionGroupsService) RemoveMember(
	subscriptionGroupID int,
	memberEmailAddress string,
) (SubscriptionGroup, error) {
	url := fmt.Sprintf("/subscription_groups/%d/remove_member", subscriptionGroupID)

	addSubscriptionGroupMemberBody := subscriptionGroupMemberToRemove{
		SubscriptionGroupMemberEmail{
			Email: memberEmailAddress,
		},
	}

	b, err := json.Marshal(addSubscriptionGroupMemberBody)
	if err != nil {
		return SubscriptionGroup{}, err
	}

	body := bytes.NewReader(b)

	var response SubscriptionGroup
	resp, err := c.client.MakeRequest(
		"PATCH",
		url,
		http.StatusOK,
		body,
	)
	if err != nil {
		return SubscriptionGroup{}, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return SubscriptionGroup{}, err
	}

	return response, nil
}
