package pivnet

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type AuthResp struct {
	Token string `json:"access_token"`
}

type TokenFetcher struct {
	Endpoint          string
	RefreshToken      string
	SkipSSLValidation bool
	UserAgent         string
}

func NewTokenFetcher(endpoint, refreshToken string, skipSSLValidation bool, userAgent string) *TokenFetcher {
	return &TokenFetcher{endpoint, refreshToken, skipSSLValidation, userAgent }
}

func (t TokenFetcher) GetToken() (string, error) {
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: t.SkipSSLValidation,
			},
			Proxy: http.ProxyFromEnvironment,
		},
	}
	body := AuthBody{RefreshToken: t.RefreshToken}
	b, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal API token request body: %s", err.Error())
	}
	req, err := http.NewRequest("POST", t.Endpoint+"/authentication/access_tokens", bytes.NewReader(b))
	req.Header.Add("Content-Type", "application/json")

	if t.UserAgent != "" {
		req.Header.Add("User-Agent", t.UserAgent)
	}

	if err != nil {
		return "", fmt.Errorf("failed to construct API token request: %s", err.Error())
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("API token request failed: %s", err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch API token - received status %v", resp.StatusCode)
	}

	var response AuthResp
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", fmt.Errorf("failed to decode API token response: %s", err.Error())
	}

	return response.Token, nil
}
