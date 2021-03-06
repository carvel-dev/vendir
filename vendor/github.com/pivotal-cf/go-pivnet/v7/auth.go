package pivnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type AuthService struct {
	client Client
}

type UAATokenResponse struct {
	Token string `json:"access_token"`
}

// Check returns:
// true,nil if the auth attempt was succesful,
// false,nil if the auth attempt failed for 401 or 403,
// false,err if the auth attempt failed for any other reason.
// It is guaranteed never to return true,err.
func (e AuthService) Check() (bool, error) {
	url := "/authentication"

	resp, err := e.client.MakeRequest(
		"GET",
		url,
		0,
		nil,
	)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusUnauthorized:
		fallthrough
	case http.StatusForbidden:
		return false, nil
	default:
		return false, e.client.handleUnexpectedResponse(resp)
	}
}

func (e AuthService) FetchUAAToken(refresh_token string) (UAATokenResponse, error) {
	url := "/authentication/access_tokens"

	body := AuthBody{RefreshToken: refresh_token}
	b, err := json.Marshal(body)
	if err != nil {
		return UAATokenResponse{}, err
	}

	resp, err := e.client.MakeRequest(
		"POST",
		url,
		0,
		bytes.NewReader(b),
	)
	if err != nil {
		return UAATokenResponse{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return UAATokenResponse{}, fmt.Errorf("failed to fetch UAA token")
	}

	var response UAATokenResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return UAATokenResponse{}, err
	}

	return response, err
}

type AuthBody struct {
	RefreshToken string `json:"refresh_token"`
}
