package pivnet

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/pivotal-cf/go-pivnet/v7/download"
	"github.com/pivotal-cf/go-pivnet/v7/logger"
)

const (
	DefaultHost         = "https://network.tanzu.vmware.com"
	apiVersion          = "/api/v2"
	concurrentDownloads = 10
)

type Client struct {
	baseURL       string
	token         AccessTokenService
	userAgent     string
	logger        logger.Logger
	usingUAAToken bool

	HTTP *http.Client

	downloader download.Client

	Auth                  *AuthService
	EULA                  *EULAsService
	ProductFiles          *ProductFilesService
	ArtifactReferences    *ArtifactReferencesService
	FederationToken       *FederationTokenService
	FileGroups            *FileGroupsService
	Releases              *ReleasesService
	Products              *ProductsService
	UserGroups            *UserGroupsService
	SubscriptionGroups    *SubscriptionGroupsService
	ReleaseTypes          *ReleaseTypesService
	ReleaseDependencies   *ReleaseDependenciesService
	DependencySpecifiers  *DependencySpecifiersService
	ReleaseUpgradePaths   *ReleaseUpgradePathsService
	UpgradePathSpecifiers *UpgradePathSpecifiersService
	PivnetVersions        *PivnetVersionsService
}

type AccessTokenOrLegacyToken struct {
	host              string
	refreshToken      string
	skipSSLValidation bool
	userAgent         string
}

type QueryParameter struct {
	Key   string
	Value string
}

func (o AccessTokenOrLegacyToken) AccessToken() (string, error) {
	const legacyAPITokenLength = 20
	if len(o.refreshToken) > legacyAPITokenLength {
		baseURL := fmt.Sprintf("%s%s", o.host, apiVersion)
		tokenFetcher := NewTokenFetcher(baseURL, o.refreshToken, o.skipSSLValidation, o.userAgent)

		accessToken, err := tokenFetcher.GetToken()
		if err != nil {
			log.Panicf("Exiting with error: %s", err)
			return "", err
		}
		return accessToken, nil
	} else {
		return o.refreshToken, nil
	}
}

func AuthorizationHeader(accessToken string) (string, error) {
	const legacyAPITokenLength = 20
	if len(accessToken) > legacyAPITokenLength {
		return fmt.Sprintf("Bearer %s", accessToken), nil
	} else {
		return fmt.Sprintf("Token %s", accessToken), nil
	}
}

type ClientConfig struct {
	Host              string
	UserAgent         string
	SkipSSLValidation bool
}

//go:generate counterfeiter . AccessTokenService
type AccessTokenService interface {
	AccessToken() (string, error)
}

func NewAccessTokenOrLegacyToken(token string, host string, skipSSLValidation bool, userAgentOptional ...string) AccessTokenOrLegacyToken {
	var userAgent = ""
	if len(userAgentOptional) > 0 {
		userAgent = userAgentOptional[0]
	}
	return AccessTokenOrLegacyToken{
		refreshToken:      token,
		host:              host,
		skipSSLValidation: skipSSLValidation,
		userAgent:         userAgent,
	}
}

func NewClient(
	token AccessTokenService,
	config ClientConfig,
	logger logger.Logger,
) Client {
	baseURL := fmt.Sprintf("%s%s", config.Host, apiVersion)

	httpClient := &http.Client{
		Timeout: 10 * time.Minute,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.SkipSSLValidation,
			},
			Proxy: http.ProxyFromEnvironment,
		},
	}

	downloadClient := &http.Client{
		Timeout: 0,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.SkipSSLValidation,
			},
			Proxy: http.ProxyFromEnvironment,
		},
	}

	ranger := download.NewRanger(concurrentDownloads)
	downloader := download.Client{
		HTTPClient: downloadClient,
		Ranger:     ranger,
		Logger:     logger,
		Timeout:    30 * time.Second,
	}

	client := Client{
		baseURL:    baseURL,
		token:      token,
		userAgent:  config.UserAgent,
		logger:     logger,
		downloader: downloader,
		HTTP:       httpClient,
	}

	client.Auth = &AuthService{client: client}
	client.EULA = &EULAsService{client: client}
	client.ProductFiles = &ProductFilesService{client: client}
	client.ArtifactReferences = &ArtifactReferencesService{client: client}
	client.FederationToken = &FederationTokenService{client: client}
	client.FileGroups = &FileGroupsService{client: client}
	client.Releases = &ReleasesService{client: client, l: logger}
	client.Products = &ProductsService{client: client, l: logger}
	client.UserGroups = &UserGroupsService{client: client}
	client.SubscriptionGroups = &SubscriptionGroupsService{client: client}
	client.ReleaseTypes = &ReleaseTypesService{client: client}
	client.ReleaseDependencies = &ReleaseDependenciesService{client: client}
	client.DependencySpecifiers = &DependencySpecifiersService{client: client}
	client.ReleaseUpgradePaths = &ReleaseUpgradePathsService{client: client}
	client.UpgradePathSpecifiers = &UpgradePathSpecifiersService{client: client}
	client.PivnetVersions = &PivnetVersionsService{client: client}

	return client
}

func (c Client) CreateRequest(
	requestType string,
	endpoint string,
	body io.Reader,
) (*http.Request, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}

	endpoint = c.stripHostPrefix(endpoint)

	u.Path = u.Path + endpoint

	req, err := http.NewRequest(requestType, u.String(), body)
	if err != nil {
		return nil, err
	}

	if !isVersionsEndpoint(endpoint) {
		accessToken, err := c.token.AccessToken()
		if err != nil {
			return nil, err
		}

		authorizationHeader, err := AuthorizationHeader(accessToken)
		if err != nil {
			return nil, fmt.Errorf("could not create authorization header: %s", err)
		}

		req.Header.Add("Authorization", authorizationHeader)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", c.userAgent)

	return req, nil
}

func (c Client) MakeRequest(
	requestType string,
	endpoint string,
	expectedStatusCode int,
	body io.Reader,
) (*http.Response, error) {
	req, err := c.CreateRequest(requestType, endpoint, body)
	if err != nil {
		return nil, err
	}

	reqBytes, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("Making request", logger.Data{"request": string(reqBytes)})

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("Response status code", logger.Data{"status code": resp.StatusCode})
	c.logger.Debug("Response headers", logger.Data{"headers": resp.Header})

	if expectedStatusCode > 0 && resp.StatusCode != expectedStatusCode {
		return nil, c.handleUnexpectedResponse(resp)
	}

	return resp, nil
}

func (c Client) MakeRequestWithParams(
	requestType string,
	endpoint string,
	expectedStatusCode int,
	params []QueryParameter,
	body io.Reader,
) (*http.Response, error) {
	req, err := c.CreateRequest(requestType, endpoint, body)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	for _, param := range params {
		q.Add(param.Key, param.Value)
	}
	req.URL.RawQuery = q.Encode()

	reqBytes, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("Making request", logger.Data{"request": string(reqBytes)})

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("Response status code", logger.Data{"status code": resp.StatusCode})
	c.logger.Debug("Response headers", logger.Data{"headers": resp.Header})

	if expectedStatusCode > 0 && resp.StatusCode != expectedStatusCode {
		return nil, c.handleUnexpectedResponse(resp)
	}

	return resp, nil
}

func (c Client) stripHostPrefix(downloadLink string) string {
	if strings.HasPrefix(downloadLink, apiVersion) {
		return downloadLink
	}
	sp := strings.Split(downloadLink, apiVersion)
	return sp[len(sp)-1]
}

func (c Client) handleUnexpectedResponse(resp *http.Response) error {
	var pErr pivnetErr

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return newErrTooManyRequests()
	}

	// We have to handle 500 differently because it has a different structure
	if resp.StatusCode == http.StatusInternalServerError {
		var internalServerError pivnetInternalServerErr
		err = json.Unmarshal(b, &internalServerError)
		if err != nil {
			return err
		}

		pErr = pivnetErr{
			Message: internalServerError.Error,
		}
	} else {
		err = json.Unmarshal(b, &pErr)
		if err != nil {
			return fmt.Errorf("could not parse json [%q] \n%s", b, err)
		}
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return newErrUnauthorized(pErr.Message)
	case http.StatusNotFound:
		return newErrNotFound(pErr.Message)
	case http.StatusUnavailableForLegalReasons:
		return newErrUnavailableForLegalReasons(pErr.Message)
	default:
		return ErrPivnetOther{
			ResponseCode: resp.StatusCode,
			Message:      pErr.Message,
			Errors:       pErr.Errors,
		}
	}
}

func isVersionsEndpoint(endpoint string) bool {
	return endpoint == "/versions"
}
