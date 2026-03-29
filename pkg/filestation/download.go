package filestation

import (
	"context"
	"encoding/json"
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

// DownloadResponse represents the response from download operations
type DownloadResponse struct {
	api.BaseResponse
	Data struct {
		DownloadID string `json:"download_id"`
		URL        string `json:"url"`
	} `json:"data"`
}

// DownloadFile downloads a file from the QNAP device to local storage
// Endpoint: /cgi-bin/filemanager/utilRequest.cgi
// Params: func=download, sid, isfolder=0, compress=0, source_path, source_file, source_total
func (fs *FileStationService) DownloadFile(ctx context.Context, remotePath, localPath string, options *DownloadOptions) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":         "download",
		"sid":          sid,
		"isfolder":     "0",
		"compress":     "0",
		"source_path":  filepath.Dir(remotePath),
		"source_file":  filepath.Base(remotePath),
		"source_total": "1",
	}

	baseURL := fs.client.GetBaseURL()
	u := baseURL.ResolveReference(&url.URL{Path: endpoint})
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
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
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			_ = cerr
		}
	}()

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
	defer func() {
		if cerr := file.Close(); cerr != nil {
			_ = cerr
		}
	}()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to download file", err)
	}

	return nil
}

// DownloadReader returns an io.ReadCloser for downloading a file
// Endpoint: /cgi-bin/filemanager/utilRequest.cgi
// Params: func=download, sid, isfolder=0, compress=0, source_path, source_file, source_total
func (fs *FileStationService) DownloadReader(ctx context.Context, remotePath string) (io.ReadCloser, int64, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, 0, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":         "download",
		"sid":          sid,
		"isfolder":     "0",
		"compress":     "0",
		"source_path":  filepath.Dir(remotePath),
		"source_file":  filepath.Base(remotePath),
		"source_total": "1",
	}

	baseURL := fs.client.GetBaseURL()
	u := baseURL.ResolveReference(&url.URL{Path: endpoint})
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
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
		if cerr := resp.Body.Close(); cerr != nil {
			_ = cerr
		}
		return nil, 0, api.NewAPIError(api.ErrUnknown, fmt.Sprintf("download failed with status %d", resp.StatusCode))
	}

	size := resp.ContentLength
	return resp.Body, size, nil
}

// DownloadFileAsync starts an asynchronous download and returns the download ID
// Endpoint: /cgi-bin/filemanager/utilRequest.cgi
// Params: func=download, sid, isfolder=0, compress=0, source_path, source_file, source_total
// Returns: download_id
func (fs *FileStationService) DownloadFileAsync(ctx context.Context, remotePath string) (*DownloadResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":         "download",
		"sid":          sid,
		"isfolder":     "0",
		"compress":     "0",
		"source_path":  filepath.Dir(remotePath),
		"source_file":  filepath.Base(remotePath),
		"source_total": "1",
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			_ = cerr
		}
	}()

	var result DownloadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse download response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}
