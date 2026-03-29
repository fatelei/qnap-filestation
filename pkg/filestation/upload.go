package filestation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// UploadOptions contains options for uploading files
type UploadOptions struct {
	Overwrite bool
	Checksum  string
	Progress  chan<- UploadProgress
}

// UploadResponse represents the response from upload operations
type UploadResponse struct {
	api.BaseResponse
	Data struct {
		Uploaded []struct {
			Path     string `json:"path"`
			Name     string `json:"name"`
			Size     int64  `json:"size"`
			Checksum string `json:"checksum,omitempty"`
		} `json:"uploaded"`
	} `json:"data"`
}

// ChunkedUploadStartResponse represents the response from starting a chunked upload
type ChunkedUploadStartResponse struct {
	api.BaseResponse
	Data struct {
		UploadID string `json:"upload_id"`
	} `json:"data"`
}

// ChunkedUploadStatusResponse represents the status of a chunked upload
type ChunkedUploadStatusResponse struct {
	api.BaseResponse
	Data struct {
		UploadID string `json:"upload_id"`
		Status   string `json:"status"`
		Offset   int64  `json:"offset"`
		Size     int64  `json:"size"`
		FilePath string `json:"file_path"`
		FileName string `json:"file_name"`
	} `json:"data"`
}

// UploadFile uploads a local file to the QNAP device using the standard upload method
// Endpoint: /cgi-bin/filemanager/utilRequest.cgi
// Params: func=upload, type=standard, sid, dest_path, overwrite=1, progress
func (fs *FileStationService) UploadFile(ctx context.Context, localPath, destPath string, options *UploadOptions) (*UploadResponse, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to open local file", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			_ = cerr
		}
	}()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to get file info", err)
	}

	return fs.UploadReader(ctx, file, destPath, fileInfo.Name(), fileInfo.Size(), options)
}

// UploadReader uploads content from an io.Reader to the QNAP device
// Endpoint: /cgi-bin/filemanager/utilRequest.cgi
// Params: func=upload, type=standard, sid, dest_path, overwrite=1, progress
func (fs *FileStationService) UploadReader(ctx context.Context, reader io.Reader, destPath, filename string, size int64, options *UploadOptions) (*UploadResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to create form file", err)
	}

	if _, err := io.Copy(part, reader); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to copy file data", err)
	}

	// Close writer to finalize form
	if err := writer.Close(); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to close multipart writer", err)
	}

	// Build parameters
	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "upload",
		"type":      "standard",
		"sid":       sid,
		"dest_path": destPath,
		"overwrite": "1",
		"progress":  "1",
	}

	// Create request
	req, err := fs.client.DoRequest(ctx, "POST", endpoint, params, body)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := req.Body.Close(); cerr != nil {
			_ = cerr
		}
	}()

	// Set content type header
	req.Request.Header.Set("Content-Type", writer.FormDataContentType())

	var uploadResp UploadResponse
	if err := json.NewDecoder(req.Body).Decode(&uploadResp); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse upload response", err)
	}

	if !uploadResp.IsSuccess() {
		return nil, &api.APIError{
			Code:    uploadResp.GetErrorCode(),
			Message: uploadResp.Message,
		}
	}

	return &uploadResp, nil
}

// StartChunkedUpload starts a chunked upload session
// Endpoint: /cgi-bin/filemanager/utilRequest.cgi
// Params: func=start_chunked_upload, sid, upload_root_dir
// Returns: upload_id
func (fs *FileStationService) StartChunkedUpload(ctx context.Context, uploadRootDir string) (string, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return "", api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":            "start_chunked_upload",
		"sid":             sid,
		"upload_root_dir": uploadRootDir,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			_ = cerr
		}
	}()

	var result ChunkedUploadStartResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", api.WrapAPIError(api.ErrUnknown, "failed to parse start chunked upload response", err)
	}

	if !result.IsSuccess() {
		return "", &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	if result.Data.UploadID == "" {
		return "", api.NewAPIError(api.ErrUnknown, "no upload_id returned")
	}

	return result.Data.UploadID, nil
}

// ChunkedUpload uploads a chunk of data
// Endpoint: /cgi-bin/filemanager/utilRequest.cgi
// Params: func=chunked_upload, sid, upload_id, offset, size
// Method: POST with binary data
func (fs *FileStationService) ChunkedUpload(ctx context.Context, uploadID string, offset int64, data []byte) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "chunked_upload",
		"sid":       sid,
		"upload_id": uploadID,
		"offset":    fmt.Sprintf("%d", offset),
		"size":      fmt.Sprintf("%d", len(data)),
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			_ = cerr
		}
	}()

	var result api.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse chunked upload response", err)
	}

	if !result.IsSuccess() {
		return &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return nil
}

// GetChunkedUpload gets the status of a chunked upload
// Endpoint: /cgi-bin/filemanager/utilRequest.cgi
// Params: func=get_chunked_upload, sid, upload_id
// Returns: Chunked upload status
func (fs *FileStationService) GetChunkedUpload(ctx context.Context, uploadID string) (*ChunkedUploadStatusResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "get_chunked_upload",
		"sid":       sid,
		"upload_id": uploadID,
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

	var result ChunkedUploadStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse chunked upload status response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// DeleteChunkedUploadFile deletes an incomplete chunked upload
// Endpoint: /cgi-bin/filemanager/utilRequest.cgi
// Params: func=delete_chunked_upload_file, sid, upload_id
func (fs *FileStationService) DeleteChunkedUploadFile(ctx context.Context, uploadID string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "delete_chunked_upload_file",
		"sid":       sid,
		"upload_id": uploadID,
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			_ = cerr
		}
	}()

	var result api.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse delete chunked upload response", err)
	}

	if !result.IsSuccess() {
		return &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return nil
}
