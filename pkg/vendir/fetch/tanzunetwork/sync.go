package tanzunetwork

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"

	semver "github.com/hashicorp/go-version"
	"github.com/pivotal-cf/go-pivnet/v7"
	"github.com/pivotal-cf/go-pivnet/v7/download"
	"github.com/pivotal-cf/go-pivnet/v7/logger"
	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
)

type Sync struct {
	client pivnet.Client
	opts   ctlconf.DirectoryContentsTanzuNetwork
}

func NewSync(host, token, userAgent string, skipSSLValidation bool, opts ctlconf.DirectoryContentsTanzuNetwork) *Sync {
	if host == "" {
		host = pivnet.DefaultHost
	}

	tokenSource := pivnet.NewAccessTokenOrLegacyToken(token, host, skipSSLValidation, userAgent)
	config := pivnet.ClientConfig{
		Host:              host,
		UserAgent:         userAgent,
		SkipSSLValidation: skipSSLValidation,
	}

	return &Sync{pivnet.NewClient(tokenSource, config, noopLogger{}), opts}
}

func (t *Sync) Sync(dstPath string, tempArea ctlfetch.TempArea) (conf ctlconf.LockDirectoryContentsTanzuNetwork, err error) {
	lockConf := ctlconf.LockDirectoryContentsTanzuNetwork{
		Slug:    t.opts.Slug,
		Version: t.opts.Version,
	}

	var releaseID int

	if t.opts.ReleaseID != nil {
		releaseID = *t.opts.ReleaseID
	} else {
		releases, err := t.client.Releases.List(t.opts.Slug)
		if err != nil {
			return lockConf, err
		}

		release, err := highestReleaseMatchingVersion(t.opts.Version, releases)
		if err != nil {
			return lockConf, err
		}

		releaseID = release.ID
		lockConf.Version = release.Version
	}

	lockConf.ReleaseID = releaseID

	productFiles, err := t.client.ProductFiles.ListForRelease(t.opts.Slug, releaseID)
	if err != nil {
		return lockConf, err
	}

	productFiles, err = filesToFetch(t.opts.Files, productFiles)
	if err != nil {
		return lockConf, err
	}

	incomingTmpPath, err := tempArea.NewTempDir("tanzuNetwork")
	if err != nil {
		return lockConf, err
	}
	defer os.RemoveAll(incomingTmpPath)

	for _, file := range productFiles {
		lock, err := downloadProductFileAndCheckSum(t.client, t.opts.Slug, incomingTmpPath, releaseID, file)

		if err != nil {
			return lockConf, fmt.Errorf("Failed to fetch %s for product %s: %w", fileName(file), t.opts.Slug, err)
		}

		lockConf.Files = append(lockConf.Files, lock)
	}

	err = ctlfetch.MoveDir(incomingTmpPath, dstPath)
	if err != nil {
		return lockConf, err
	}

	return lockConf, nil
}

func filesToFetch(files []ctlconf.DirectoryContentsTanzuNetworkFile, productFiles []pivnet.ProductFile) ([]pivnet.ProductFile, error) {
	withID := func(id int) (pivnet.ProductFile, bool) {
		for _, f := range productFiles {
			if f.ID == id {
				return f, true
			}
		}
		return pivnet.ProductFile{}, false
	}

	withName := func(pattern string) (pivnet.ProductFile, bool) {
		for _, productFile := range productFiles {
			matched, err := filepath.Match(pattern, fileName(productFile))
			if err != nil {
				continue
			}
			if matched {
				return productFile, true
			}
		}

		return pivnet.ProductFile{}, false
	}

	existingNames := func() []string {
		list := make([]string, len(productFiles))
		for i, f := range productFiles {
			list[i] = fileName(f)
		}
		return list
	}

	toFetch := make([]pivnet.ProductFile, len(files))
	for i, file := range files {

		if file.ID != nil {
			f, ok := withID(*file.ID)
			if !ok {
				return nil, fmt.Errorf("File with id %d not found", *file.ID)
			}

			toFetch[i] = f
			continue
		}

		f, ok := withName(file.Name)
		if !ok {
			return nil, fmt.Errorf("No file with name matching %q (file names: %v)", file.Name, existingNames())
		}

		toFetch[i] = f
	}

	return toFetch, nil
}

func highestReleaseMatchingVersion(constraint string, releases []pivnet.Release) (pivnet.Release, error) {
	if constraint == "" && len(releases) > 0 {
		return releases[0], nil
	}

	var (
		filteredReleases = releases[:0]
		versions         semver.Collection
	)

	for _, release := range releases {
		v, err := semver.NewVersion(release.Version)
		if err != nil {
			continue
		}
		filteredReleases = append(filteredReleases, release)
		versions = append(versions, v)
	}
	releases = filteredReleases

	sort.Sort(sort.Reverse(sorter{
		len: len(releases),

		less: func(i, j int) bool { return versions[i].LessThan(versions[j]) },

		swap: func(i, j int) {
			versions[i], versions[j] = versions[j], versions[i]
			releases[i], releases[j] = releases[j], releases[i]
		},
	}))

	releaseConstraint, err := semver.NewConstraint(constraint)
	if err != nil {
		return pivnet.Release{}, fmt.Errorf("Product release contstraint invalid: %w", err)
	}

	for i, v := range versions {
		if releaseConstraint.Check(v) {
			return releases[i], nil // product found :)
		}
	}

	return pivnet.Release{}, fmt.Errorf("Release version matching %q not found", constraint)
}

// downloadProductFileAndCheckSum callers should wrap returned errors
func downloadProductFileAndCheckSum(client pivnet.Client, slug, localDir string, releaseID int, productFile pivnet.ProductFile) (ctlconf.LockDirectoryContentsTanzuNetworkFile, error) {
	var lock ctlconf.LockDirectoryContentsTanzuNetworkFile
	incomingFilePath := filepath.Join(localDir, fileName(productFile))

	f, err := os.Create(incomingFilePath)
	if err != nil {
		return lock, err
	}
	defer f.Close()

	fi, err := download.NewFileInfo(f)
	if err != nil {
		return lock, err
	}

	err = client.ProductFiles.DownloadForRelease(fi, slug, releaseID, productFile.ID, ioutil.Discard)
	if err != nil {
		return lock, err
	}

	_, _ = f.Seek(0, 0)

	bf := bufio.NewReader(f)

	digestDst := sha256.New()
	_, err = io.Copy(digestDst, bf)
	if err != nil {
		return lock, err
	}

	if gotSum := fmt.Sprintf("%x", digestDst.Sum(nil)); gotSum != productFile.SHA256 {
		return lock, fmt.Errorf("expected sha256sum %q got %q", productFile.SHA256, gotSum)
	}

	return ctlconf.LockDirectoryContentsTanzuNetworkFile{
		Name:      fileName(productFile),
		ID:        productFile.ID,
		SHA256Sum: productFile.SHA256,
	}, nil
}

func fileName(file pivnet.ProductFile) string {
	return path.Base(file.AWSObjectKey)
}

type noopLogger struct{}

func (noopLogger) Debug(string, ...logger.Data) {}
func (noopLogger) Info(string, ...logger.Data)  {}

type sorter struct {
	len  int
	swap func(i, j int)
	less func(i, j int) bool
}

func (x sorter) Len() int           { return x.len }
func (x sorter) Swap(i, j int)      { x.swap(i, j) }
func (x sorter) Less(i, j int) bool { return x.less(i, j) }
