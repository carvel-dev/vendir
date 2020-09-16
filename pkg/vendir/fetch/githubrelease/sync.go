package githubrelease

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	ctlconf "github.com/k14s/vendir/pkg/vendir/config"
	ctlfetch "github.com/k14s/vendir/pkg/vendir/fetch"
)

type Sync struct {
	opts     ctlconf.DirectoryContentsGithubRelease
	apiToken string
}

func NewSync(opts ctlconf.DirectoryContentsGithubRelease, apiToken string) Sync {
	return Sync{opts, apiToken}
}

func (d Sync) DescAndURL() (string, string, error) {
	desc := ""
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", d.opts.Slug)

	switch {
	case len(d.opts.URL) > 0:
		desc = d.opts.URL
		url = d.opts.URL
	case len(d.opts.Tag) > 0:
		desc = d.opts.Slug + "@" + d.opts.Tag
		url += "/tags/" + d.opts.Tag
	case d.opts.Latest:
		desc = d.opts.Slug + "@latest"
		url += "/latest"
	default:
		return "", "", fmt.Errorf("Expected to have non-empty tag, latest or url")
	}
	return desc, url, nil
}

func (d Sync) Sync(dstPath string, tempArea ctlfetch.TempArea) (ctlconf.LockDirectoryContentsGithubRelease, error) {
	lockConf := ctlconf.LockDirectoryContentsGithubRelease{}

	incomingTmpPath, err := tempArea.NewTempDir("github-release")
	if err != nil {
		return lockConf, err
	}

	defer os.RemoveAll(incomingTmpPath)

	releaseAPI, err := d.downloadRelease()
	if err != nil {
		return lockConf, fmt.Errorf("Downloading release info: %s", err)
	}

	fileChecksums := map[string]string{}

	if len(d.opts.Checksums) > 0 {
		fileChecksums = d.opts.Checksums
	} else {
		if !d.opts.DisableAutoChecksumValidation {
			fileChecksums, err = ReleaseNotesChecksums{}.Find(releaseAPI.AssetNames(), releaseAPI.Body)
			if err != nil {
				return lockConf, fmt.Errorf("Finding checksums in release notes: %s", err)
			}
		}
	}

	for _, asset := range releaseAPI.Assets {
		path := filepath.Join(incomingTmpPath, asset.Name)

		err := d.downloadFile(asset.URL, path)
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

func (d Sync) downloadRelease() (GithubReleaseAPI, error) {
	releaseAPI := GithubReleaseAPI{}

	_, url, err := d.DescAndURL()
	if err != nil {
		return releaseAPI, err
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return releaseAPI, err
	}

	if len(d.apiToken) > 0 {
		req.Header.Add("Authorization", "token "+d.apiToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return releaseAPI, err
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
			errMsg += " " + hintMsg
		}
		return releaseAPI, fmt.Errorf(errMsg)
	}

	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return releaseAPI, err
	}

	err = json.Unmarshal(bs, &releaseAPI)
	if err != nil {
		return releaseAPI, err
	}

	return releaseAPI, nil
}

func (d Sync) downloadFile(url string, dstPath string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// Forces Github to redirect to asset contents
	req.Header.Add("Accept", "application/octet-stream")

	if len(d.apiToken) > 0 {
		req.Header.Add("Authorization", "token "+d.apiToken)
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
			bs, _ := ioutil.ReadAll(resp.Body)
			errMsg += fmt.Sprintf(" (body: '%s')", bs)
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

type GithubReleaseAPI struct {
	URL    string `json:"url"`
	Body   string
	Assets []GithubReleaseAssetAPI
}

type GithubReleaseAssetAPI struct {
	URL  string
	Name string
	Size int64
	// This URL does not work for private repo assets
	// BrowserDownloadURL string `json:"browser_download_url"`
}

func (a GithubReleaseAPI) AssetNames() []string {
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
