package download

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/go-pivnet/v7/logger"
	"github.com/shirou/gopsutil/disk"
	"golang.org/x/sync/errgroup"
)

//go:generate counterfeiter -o ./fakes/ranger.go --fake-name Ranger . ranger
type ranger interface {
	BuildRange(contentLength int64) ([]Range, error)
}

//go:generate counterfeiter -o ./fakes/http_client.go --fake-name HTTPClient . httpClient
type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type downloadLinkFetcher interface {
	NewDownloadLink() (string, error)
}

//go:generate counterfeiter -o ./fakes/bar.go --fake-name Bar . bar
type bar interface {
	SetTotal(contentLength int64)
	SetOutput(output io.Writer)
	Add(totalWritten int) int
	Kickoff()
	Finish()
	NewProxyReader(reader io.Reader) io.Reader
}

type Client struct {
	HTTPClient httpClient
	Ranger     ranger
	Bar        bar
	Logger     logger.Logger
	Timeout    time.Duration
}

type FileInfo struct {
	Name string
	Mode os.FileMode
}

func NewFileInfo(file *os.File) (*FileInfo, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fileInfo := &FileInfo{
		Name: file.Name(),
		Mode: stat.Mode(),
	}

	return fileInfo, nil
}

func (c Client) Get(
	location *FileInfo,
	downloadLinkFetcher downloadLinkFetcher,
	progressWriter io.Writer,
) error {
	contentURL, err := downloadLinkFetcher.NewDownloadLink()
	if err != nil {
		return fmt.Errorf("could not create new download link in get: %s", err)
	}

	req, err := http.NewRequest("HEAD", contentURL, nil)
	if err != nil {
		return fmt.Errorf("failed to construct HEAD request: %s", err)
	}

	req.Header.Add("Referer", "https://go-pivnet.network.tanzu.vmware.com")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make HEAD request: %s", err)
	}

	c.Logger.Debug(fmt.Sprintf("HEAD response content size: %d", resp.ContentLength))

	contentURL = resp.Request.URL.String()

	if resp.ContentLength == -1 {
		return fmt.Errorf("failed to find file on remote filestore")
	}

	ranges, err := c.Ranger.BuildRange(resp.ContentLength)
	if err != nil {
		return fmt.Errorf("failed to construct range: %s", err)
	}

	diskStats, err := disk.Usage(path.Dir(location.Name))
	if err != nil {
		return fmt.Errorf("failed to get disk free space: %s", err)
	}

	if diskStats.Free < uint64(resp.ContentLength) {
		return fmt.Errorf("file is too big to fit on this drive: %d bytes required, %d bytes free", uint64(resp.ContentLength), diskStats.Free)
	}

	c.Bar.SetOutput(progressWriter)
	c.Bar.SetTotal(resp.ContentLength)
	c.Bar.Kickoff()

	defer c.Bar.Finish()

	var g errgroup.Group
	for _, r := range ranges {
		byteRange := r

		fileWriter, err := os.OpenFile(location.Name, os.O_RDWR, location.Mode)
		if err != nil {
			return fmt.Errorf("failed to open file %s for writing: %s", location.Name, err)
		}

		g.Go(func() error {
			err := c.retryableRequest(contentURL, byteRange.HTTPHeader, fileWriter, byteRange.Lower, downloadLinkFetcher, c.Timeout)
			if err != nil {
				return fmt.Errorf("failed during retryable request: %s", err)
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("problem while waiting for chunks to download: %s", err)
	}

	return nil
}

func (c Client) retryableRequest(contentURL string, rangeHeader http.Header, fileWriter *os.File, startingByte int64, downloadLinkFetcher downloadLinkFetcher, timeout time.Duration) error {
	currentURL := contentURL
	defer fileWriter.Close()

	var err error
Retry:
	_, err = fileWriter.Seek(startingByte, 0)
	if err != nil {
		return fmt.Errorf("failed to seek to correct byte of output file: %s", err)
	}

	req, err := http.NewRequest("GET", currentURL, nil)
	if err != nil {
		return fmt.Errorf("could not get new request: %s", err)
	}

	rangeHeader.Add("Referer", "https://go-pivnet.network.tanzu.vmware.com")
	req.Header = rangeHeader

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		if netErr, ok := err.(net.Error); ok {
			if netErr.Temporary() {
				goto Retry
			}
		}

		return fmt.Errorf("download request failed: %s", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		c.Logger.Debug("received unsuccessful status code: %d", logger.Data{"statusCode": resp.StatusCode})
		currentURL, err = downloadLinkFetcher.NewDownloadLink()
		if err != nil {
			return fmt.Errorf("could not get new download link: %s", err)
		}
		c.Logger.Debug("fetched new download url: %d", logger.Data{"url": currentURL})

		goto Retry
	}

	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("during GET unexpected status code was returned: %d", resp.StatusCode)
	}

	var proxyReader io.Reader
	proxyReader = c.Bar.NewProxyReader(resp.Body)

	var timeoutReader io.Reader
	timeoutReader = gbytes.TimeoutReader(proxyReader, timeout)

	bytesWritten, err := io.Copy(fileWriter, timeoutReader)
	if err != nil {
		if err == io.ErrUnexpectedEOF || err == gbytes.ErrTimeout {
			c.Logger.Debug(fmt.Sprintf("retrying %v", err))
			c.Bar.Add(int(-1 * bytesWritten))
			goto Retry
		}
		oe, _ := err.(*net.OpError)
		if strings.Contains(oe.Err.Error(), syscall.ECONNRESET.Error()) {
			c.Bar.Add(int(-1 * bytesWritten))
			goto Retry
		}
		return fmt.Errorf("failed to write file during io.Copy: %s", err)
	}

	return nil
}
