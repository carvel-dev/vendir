package pivnet

import (
	"net/http"
)

type ProductFileLinkFetcher struct {
	downloadLink string
	client       Client
}

func NewProductFileLinkFetcher(downloadLink string, client Client) ProductFileLinkFetcher {
	return ProductFileLinkFetcher{downloadLink: downloadLink, client: client}
}

func (p ProductFileLinkFetcher) NewDownloadLink() (string, error) {
	p.client.HTTP.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := p.client.MakeRequest("POST", p.downloadLink, http.StatusFound, nil)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	p.client.HTTP.CheckRedirect = nil

	return resp.Header.Get("Location"), nil
}
