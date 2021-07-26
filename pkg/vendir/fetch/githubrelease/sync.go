// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package githubrelease

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	ctlver "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/versions/v1alpha1"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar"
	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
)

type Sync struct {
	opts            ctlconf.DirectoryContentsGithubRelease
	defaultAPIToken string
	refFetcher      ctlfetch.RefFetcher
}

func NewSync(opts ctlconf.DirectoryContentsGithubRelease,
	defaultAPIToken string, refFetcher ctlfetch.RefFetcher) Sync {

	return Sync{opts, defaultAPIToken, refFetcher}
}

func (d Sync) Desc() (string, error) {
	desc := ""

	switch {
	case len(d.opts.URL) > 0:
		desc = d.opts.URL
	case len(d.opts.Tag) > 0:
		desc = d.opts.Slug + "@" + d.opts.Tag
	case d.opts.TagSelection != nil:
		desc = d.opts.Slug + "@"
		switch {
		case d.opts.TagSelection.Semver != nil:
			desc += fmt.Sprintf("[%s]", d.opts.TagSelection.Semver.Constraints)
		}
	case d.opts.Latest:
		desc = d.opts.Slug + "@latest"
	default:
		return "", fmt.Errorf("Expected to have non-empty tag, tagSelection, latest or url")
	}
	return desc, nil
}

func (d Sync) URL() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", d.opts.Slug)

	switch {
	case len(d.opts.URL) > 0:
		url = d.opts.URL
	case len(d.opts.Tag) > 0:
		url += "/tags/" + d.opts.Tag
	case d.opts.TagSelection != nil:
		tag, err := d.fetchTagSelection()
		if err != nil {
			return "", err
		}
		url += "/tags/" + tag
	case d.opts.Latest:
		url += "/latest"
	default:
		return "", fmt.Errorf("Expected to have non-empty tag, tagSelection, latest or url")
	}
	return url, nil
}

func (d Sync) Sync(dstPath string, tempArea ctlfetch.TempArea) (ctlconf.LockDirectoryContentsGithubRelease, error) {
	lockConf := ctlconf.LockDirectoryContentsGithubRelease{}

	incomingTmpPath, err := tempArea.NewTempDir("github-release")
	if err != nil {
		return lockConf, err
	}

	defer os.RemoveAll(incomingTmpPath)

	authToken, err := d.authToken()
	if err != nil {
		return lockConf, err
	}

	releaseAPI, err := d.downloadRelease(authToken)
	if err != nil {
		return lockConf, fmt.Errorf("Downloading release info: %s", err)
	}

	fileChecksums := map[string]string{}
	matchedAssets := []ReleaseAssetAPI{}

	for _, asset := range releaseAPI.Assets {
		matched, err := d.matchesAssetName(asset.Name)
		if err != nil {
			return lockConf, fmt.Errorf("Matching asset name '%s': %s", asset.Name, err)
		}
		if matched {
			matchedAssets = append(matchedAssets, asset)
		}
	}

	if len(d.opts.Checksums) > 0 {
		fileChecksums = d.opts.Checksums
	} else {
		if !d.opts.DisableAutoChecksumValidation {
			fileChecksums, err = ReleaseNotesChecksums{}.Find(matchedAssets, releaseAPI.Body)
			if err != nil {
				return lockConf, fmt.Errorf("Finding checksums in release notes: %s", err)
			}
		}
	}

	for _, asset := range matchedAssets {
		path := filepath.Join(incomingTmpPath, asset.Name)

		err = d.downloadFile(asset.URL, path, authToken)
		if err != nil {
			return lockConf, fmt.Errorf("Downloading asset '%s': %s", asset.Name, err)
		}

		err = d.checkFileSize(path, asset.Size)
		if err != nil {
			return lockConf, fmt.Errorf("Checking asset '%s' size: %s", asset.Name, err)
		}

		if len(fileChecksums) > 0 {
			err = d.checkFileChecksum(path, fileChecksums[asset.Name])
			if err != nil {
				return lockConf, fmt.Errorf("Checking asset '%s' checksum: %s", asset.Name, err)
			}
		}
	}

	if d.opts.UnpackArchive != nil {
		newIncomingTmpPath, err := tempArea.NewTempDir("github-release-unpack")
		if err != nil {
			return lockConf, err
		}

		defer os.RemoveAll(newIncomingTmpPath)

		final, err := ctlfetch.NewArchive(filepath.Join(incomingTmpPath, d.opts.UnpackArchive.Path), false, "").Unpack(newIncomingTmpPath)
		if err != nil {
			return lockConf, fmt.Errorf("Unpacking archive '%s': %s", d.opts.UnpackArchive.Path, err)
		}
		if !final {
			return lockConf, fmt.Errorf("Expected known archive type (zip, tgz, tar)")
		}

		incomingTmpPath = newIncomingTmpPath
	}

	err = os.RemoveAll(dstPath)
	if err != nil {
		return lockConf, fmt.Errorf("Deleting dir %s: %s", dstPath, err)
	}

	err = os.Rename(incomingTmpPath, dstPath)
	if err != nil {
		return lockConf, fmt.Errorf("Moving directory '%s' to staging dir: %s", incomingTmpPath, err)
	}

	lockConf.URL = releaseAPI.URL

	return lockConf, nil
}

func (d Sync) matchesAssetName(name string) (bool, error) {
	if len(d.opts.AssetNames) == 0 {
		return true, nil
	}
	for _, pattern := range d.opts.AssetNames {
		ok, err := doublestar.PathMatch(pattern, name)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func (d Sync) fetchTagSelection() (string, error) {
	authToken, err := d.authToken()
	if err != nil {
		return "", err
	}
	tagList, err := d.downloadTags(authToken)
	if err != nil {
		return "", fmt.Errorf("Downloading tags info: %s", err)
	}
	tags := make([]string, len(tagList))
	for _, tag := range tagList {
		tags = append(tags, tag.Name)
	}
	tag, err := ctlver.HighestConstrainedVersion(tags, *d.opts.TagSelection)
	if err != nil {
		return "", fmt.Errorf("Failed to find tag matching tagSelection %v : %s", d.opts.TagSelection.Semver, err)
	}
	return tag, err
}

func (d Sync) downloadTags(authToken string) ([]TagAPI, error) {
	tagAPI := []TagAPI{}
	// CAUTION! Paginaton not implemented, more than 100 results not supported
	// Results seem to be sorted lexically in reverse (i.e. highest semver first),
	// rather than by descending date / commit order (as in the API)
	url := fmt.Sprintf("https://api.github.com/repos/%s/tags?per_page=100", d.opts.Slug)
	respBytes, err := d.downloadAPIResponse(url, authToken)
	if err != nil {
		return tagAPI, err
	}
	err = json.Unmarshal(respBytes, &tagAPI)
	if err != nil {
		return tagAPI, err
	}

	return tagAPI, nil
}

func (d Sync) downloadRelease(authToken string) (ReleaseAPI, error) {
	releaseAPI := ReleaseAPI{}

	url, err := d.URL()
	if err != nil {
		return releaseAPI, fmt.Errorf("getting release URL: %s", err)
	}
	respBytes, err := d.downloadAPIResponse(url, authToken)
	if err != nil {
		errMsg := err.Error()
		return releaseAPI, fmt.Errorf("error downloading release details from %s : %s", url, errMsg)
	}

	err = json.Unmarshal(respBytes, &releaseAPI)
	if err != nil {
		return releaseAPI, fmt.Errorf("error parsing response from: %s error: %s", url, err.Error())
	}

	return releaseAPI, nil
}

func (d Sync) downloadAPIResponse(url string, authToken string) ([]byte, error) {
	bs := []byte{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return bs, err
	}

	if len(authToken) > 0 {
		req.Header.Add("Authorization", "token "+authToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return bs, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errMsg := fmt.Sprintf("Expected response status 200, but was '%d'", resp.StatusCode)
		switch resp.StatusCode {
		case 401, 403:
			hintMsg := "(hint: consider setting VENDIR_GITHUB_API_TOKEN env variable to increase API rate limits)"
			bs, _ := ioutil.ReadAll(resp.Body)
			errMsg += fmt.Sprintf(" %s (body: '%s')", hintMsg, bs)
		case 404:
			hintMsg := "(hint: if you are using 'latest: true', there may not be any non-pre-release releases)"
			bs, _ := ioutil.ReadAll(resp.Body)
			errMsg += fmt.Sprintf(" %s (body: '%s')", hintMsg, bs)
		}
		return bs, fmt.Errorf(errMsg)
	}

	bs, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return bs, err
	}

	return bs, nil
}

func (d Sync) downloadFile(url, dstPath, authToken string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// Forces Github to redirect to asset contents
	req.Header.Add("Accept", "application/octet-stream")

	if len(authToken) > 0 {
		req.Header.Add("Authorization", "token "+authToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errMsg := fmt.Sprintf("Expected response status 200, but was '%d'", resp.StatusCode)
		switch resp.StatusCode {
		case 401, 403:
			hintMsg := "(hint: consider setting VENDIR_GITHUB_API_TOKEN env variable to increase API rate limits)"
			bs, _ := ioutil.ReadAll(resp.Body)
			errMsg += fmt.Sprintf(" %s (body: '%s')", hintMsg, bs)
		}
		return fmt.Errorf(errMsg)
	}

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (d Sync) checkFileSize(path string, expectedSize int64) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.Size() != expectedSize {
		return fmt.Errorf("Expected file size to be %d, but was %d", expectedSize, fi.Size())
	}
	return nil
}

func (d Sync) checkFileChecksum(path string, expectedChecksum string) error {
	if len(expectedChecksum) == 0 {
		panic("Expected non-empty checksum as argument")
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	hash := sha256.New()
	_, err = io.Copy(hash, f)
	if err != nil {
		return fmt.Errorf("Calculating checksum: %s", err)
	}

	actualChecksum := fmt.Sprintf("%x", hash.Sum(nil))

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("Expected file checksum to be '%s', but was '%s'",
			expectedChecksum, actualChecksum)
	}
	return nil
}

func (d Sync) authToken() (string, error) {
	token := ""

	if len(d.defaultAPIToken) > 0 {
		token = d.defaultAPIToken
	}

	if d.opts.SecretRef != nil {
		secret, err := d.refFetcher.GetSecret(d.opts.SecretRef.Name)
		if err != nil {
			return "", err
		}

		for name, val := range secret.Data {
			switch name {
			case ctlconf.SecretToken:
				token = string(val)
			default:
				return "", fmt.Errorf("Unknown secret field '%s' in secret '%s'", name, secret.Metadata.Name)
			}
		}
	}

	return token, nil
}

type TagAPI struct {
	Name string
}

/*
Example response:

[
	{
		"name": "0.0.1",
		"zipball_url": "https://api.github.com/repos/%s/zipball/refs/tags/0.05",
		"tarball_url": "https://api.github.com/repos/%s/tarball/refs/tags/0.05",
		"commit": {
			"sha": "{COMMIT_SHA}",
			"url": "https://api.github.com/repos/%s/commits/{COMMIT_ID}"
		},
		"node_id": "{{  NODE_ID }}"
	}
]
*/

type ReleaseAPI struct {
	URL    string `json:"url"`
	Body   string
	Assets []ReleaseAssetAPI
}

type ReleaseAssetAPI struct {
	URL  string
	Name string
	Size int64
	// This URL does not work for private repo assets
	// BrowserDownloadURL string `json:"browser_download_url"`
}

func (a ReleaseAPI) AssetNames() []string {
	var result []string
	for _, asset := range a.Assets {
		result = append(result, asset.Name)
	}
	return result
}

/*

Example response (not all fields present):

{
  "url": "https://api.github.com/repos/jgm/pandoc/releases/22608933",
  "id": 22608933,
  "node_id": "MDc6UmVsZWFzZTIyNjA4OTMz",
  "tag_name": "2.9.1.1",
  "target_commitish": "master",
  "name": "pandoc 2.9.1.1",
  "draft": false,
  "assets": [
    {
      "url": "https://api.github.com/repos/jgm/pandoc/releases/assets/17158996",
      "id": 17158996,
      "node_id": "MDEyOlJlbGVhc2VBc3NldDE3MTU4OTk2",
      "name": "pandoc-2.9.1.1-windows-x86_64.zip",
      "label": null,
      "content_type": "application/zip",
      "state": "uploaded",
      "size": 36132549,
      "download_count": 9236,
      "created_at": "2020-01-06T05:16:32Z",
      "updated_at": "2020-01-06T05:23:48Z",
      "browser_download_url": "https://github.com/jgm/pandoc/releases/download/2.9.1.1/pandoc-2.9.1.1-windows-x86_64.zip"
    }
  ],
  "body": "..."
}

*/
