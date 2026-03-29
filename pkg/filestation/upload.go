package filestation

import (
	"bytes"
	"context"
	"encoding/json"
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

// UploadFile uploads a local file to the QNAP device
func (fs *FileStationService) UploadFile(ctx context.Context, localPath, remotePath string, options *UploadOptions) error {
	file, err := os.Open(localPath)
	if err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to open local file", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to get file info", err)
	}

	return fs.UploadReader(ctx, file, remotePath, fileInfo.Name(), fileInfo.Size(), options)
}

// UploadReader uploads content from an io.Reader to the QNAP device
func (fs *FileStationService) UploadReader(ctx context.Context, reader io.Reader, remotePath, filename string, size int64, options *UploadOptions) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to create form file", err)
	}

	if _, err := io.Copy(part, reader); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to copy file data", err)
	}

	if err := writer.WriteField("path", remotePath); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to write path field", err)
	}

	if options != nil && options.Overwrite {
		if err := writer.WriteField("overwrite", "true"); err != nil {
			return api.WrapAPIError(api.ErrUnknown, "failed to write overwrite field", err)
		}
	}

	if err := writer.Close(); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to close multipart writer", err)
	}

	endpoint := "/filestation/upload.cgi"
	params := map[string]string{
		"api":     "SYNO.FileStation.Upload",
		"method":  "upload",
		"version": "2",
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Set content type
	if resp.Request != nil {
		resp.Request.Header.Set("Content-Type", writer.FormDataContentType())
	}

	var uploadResp UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse upload response", err)
	}

	if !uploadResp.IsSuccess() {
		return &api.APIError{
			Code:    uploadResp.GetErrorCode(),
			Message: uploadResp.Message,
		}
	}

	return nil
}
