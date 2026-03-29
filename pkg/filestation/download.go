package filestation

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// DownloadOptions contains options for downloading files
type DownloadOptions struct {
	Offset   int64
	Length   int64
	Progress chan<- DownloadProgress
}

// DownloadFile downloads a file from the QNAP device to local storage
func (fs *FileStationService) DownloadFile(ctx context.Context, remotePath, localPath string, options *DownloadOptions) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/filestation/download.cgi"
	params := map[string]string{
		"api":     "SYNO.FileStation.Download",
		"method":  "download",
		"version": "2",
		"path":    remotePath,
		"mode":    "open",
	}

	baseURL := fs.client.GetBaseURL()
	u := baseURL.ResolveReference(&url.URL{Path: endpoint})
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	if sid != "" {
		q.Set("sid", sid)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := fs.client.GetHTTPClient().Do(req)
	if err != nil {
		return api.WrapAPIError(api.ErrNetwork, "download request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return api.NewAPIError(api.ErrUnknown, fmt.Sprintf("download failed with status %d", resp.StatusCode))
	}

	// Create local directory
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to create local directory", err)
	}

	file, err := os.Create(localPath)
	if err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to create local file", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to download file", err)
	}

	return nil
}

// DownloadReader returns an io.ReadCloser for downloading a file
func (fs *FileStationService) DownloadReader(ctx context.Context, remotePath string) (io.ReadCloser, int64, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, 0, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/filestation/download.cgi"
	params := map[string]string{
		"api":     "SYNO.FileStation.Download",
		"method":  "download",
		"version": "2",
		"path":    remotePath,
		"mode":    "open",
	}

	baseURL := fs.client.GetBaseURL()
	u := baseURL.ResolveReference(&url.URL{Path: endpoint})
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	if sid != "" {
		q.Set("sid", sid)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := fs.client.GetHTTPClient().Do(req)
	if err != nil {
		return nil, 0, api.WrapAPIError(api.ErrNetwork, "download request failed", err)
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, 0, api.NewAPIError(api.ErrUnknown, fmt.Sprintf("download failed with status %d", resp.StatusCode))
	}

	size := resp.ContentLength
	return resp.Body, size, nil
}
