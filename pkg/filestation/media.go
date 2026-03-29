package filestation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// GetThumbResponse represents the response from get_thumb
type GetThumbResponse struct {
	api.BaseResponse
	Data struct {
		ThumbnailURL string `json:"thumbnail_url"`
	} `json:"data"`
}

// GetThumbOptions contains options for getting thumbnails
type GetThumbOptions struct {
	Size    string `json:"size,omitempty"`    // Size: small, medium, large
	Width   int    `json:"width,omitempty"`   // Custom width
	Height  int    `json:"height,omitempty"`  // Custom height
	Rotate  int    `json:"rotate,omitempty"`  // Rotation angle
	Effect  string `json:"effect,omitempty"`  // Effect: grayscale, sepia, etc.
	Buffer  bool   `json:"buffer,omitempty"`  // Return as base64
	Timeout int    `json:"timeout,omitempty"` // Timeout in seconds
}

// GetThumb gets a thumbnail for a file
func (fs *FileStationService) GetThumb(ctx context.Context, path string, options *GetThumbOptions) (*GetThumbResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_thumb",
		"sid":  sid,
		"path": path,
	}

	if options != nil {
		if options.Size != "" {
			params["size"] = options.Size
		}
		if options.Width > 0 {
			params["width"] = fmt.Sprintf("%d", options.Width)
		}
		if options.Height > 0 {
			params["height"] = fmt.Sprintf("%d", options.Height)
		}
		if options.Rotate > 0 {
			params["rotate"] = fmt.Sprintf("%d", options.Rotate)
		}
		if options.Effect != "" {
			params["effect"] = options.Effect
		}
		if options.Buffer {
			params["buffer"] = "1"
		}
		if options.Timeout > 0 {
			params["timeout"] = fmt.Sprintf("%d", options.Timeout)
		}
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetThumbResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse get thumb response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// ForceThumbResponse represents the response from force_thumb
type ForceThumbResponse struct {
	api.BaseResponse
	Data struct {
		PID     string `json:"pid"` // Process ID
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
	} `json:"data"`
}

// ForceThumbOptions contains options for forcing thumbnail generation
type ForceThumbOptions struct {
	Size   string `json:"size,omitempty"`   // Size: small, medium, large
	Width  int    `json:"width,omitempty"`  // Custom width
	Height int    `json:"height,omitempty"` // Custom height
}

// ForceThumb forces generation of a thumbnail
func (fs *FileStationService) ForceThumb(ctx context.Context, path string, options *ForceThumbOptions) (*ForceThumbResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "force_thumb",
		"sid":  sid,
		"path": path,
	}

	if options != nil {
		if options.Size != "" {
			params["size"] = options.Size
		}
		if options.Width > 0 {
			params["width"] = fmt.Sprintf("%d", options.Width)
		}
		if options.Height > 0 {
			params["height"] = fmt.Sprintf("%d", options.Height)
		}
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result ForceThumbResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse force thumb response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// RemoteThumbResponse represents the response from remote_thumb
type RemoteThumbResponse struct {
	api.BaseResponse
	Data struct {
		ThumbnailURL string `json:"thumbnail_url"`
	} `json:"data"`
}

// RemoteThumbOptions contains options for remote thumbnail
type RemoteThumbOptions struct {
	Size   string `json:"size,omitempty"`   // Size: small, medium, large
	Width  int    `json:"width,omitempty"`  // Custom width
	Height int    `json:"height,omitempty"` // Custom height
	Buffer bool   `json:"buffer,omitempty"` // Return as base64
}

// RemoteThumb gets a thumbnail from a remote URL
func (fs *FileStationService) RemoteThumb(ctx context.Context, url string, options *RemoteThumbOptions) (*RemoteThumbResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "remote_thumb",
		"sid":  sid,
		"url":  url,
	}

	if options != nil {
		if options.Size != "" {
			params["size"] = options.Size
		}
		if options.Width > 0 {
			params["width"] = fmt.Sprintf("%d", options.Width)
		}
		if options.Height > 0 {
			params["height"] = fmt.Sprintf("%d", options.Height)
		}
		if options.Buffer {
			params["buffer"] = "1"
		}
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result RemoteThumbResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse remote thumb response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// SupportPdfThumbResponse represents the response from support_pdf_thumb
type SupportPdfThumbResponse struct {
	api.BaseResponse
	Data struct {
		Supported bool `json:"supported"`
		Enabled   bool `json:"enabled"`
	} `json:"data"`
}

// SupportPdfThumb checks if PDF thumbnail is supported
func (fs *FileStationService) SupportPdfThumb(ctx context.Context) (*SupportPdfThumbResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "support_pdf_thumb",
		"sid":  sid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result SupportPdfThumbResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse support pdf thumb response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// GetSupportPdfThumbResponse represents the response from get_support_pdf_thumb
type GetSupportPdfThumbResponse struct {
	api.BaseResponse
	Data struct {
		ThumbnailURL string `json:"thumbnail_url"`
		Page         int    `json:"page"`
	} `json:"data"`
}

// GetSupportPdfThumbOptions contains options for getting PDF thumbnail
type GetSupportPdfThumbOptions struct {
	Page   int    `json:"page,omitempty"`   // Page number
	Size   string `json:"size,omitempty"`   // Size: small, medium, large
	Width  int    `json:"width,omitempty"`  // Custom width
	Height int    `json:"height,omitempty"` // Custom height
	Buffer bool   `json:"buffer,omitempty"` // Return as base64
}

// GetSupportPdfThumb gets a thumbnail for a PDF file
func (fs *FileStationService) GetSupportPdfThumb(ctx context.Context, path string, options *GetSupportPdfThumbOptions) (*GetSupportPdfThumbResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_support_pdf_thumb",
		"sid":  sid,
		"path": path,
	}

	if options != nil {
		if options.Page > 0 {
			params["page"] = fmt.Sprintf("%d", options.Page)
		}
		if options.Size != "" {
			params["size"] = options.Size
		}
		if options.Width > 0 {
			params["width"] = fmt.Sprintf("%d", options.Width)
		}
		if options.Height > 0 {
			params["height"] = fmt.Sprintf("%d", options.Height)
		}
		if options.Buffer {
			params["buffer"] = "1"
		}
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetSupportPdfThumbResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse get support pdf thumb response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// EnableThumbnailResponse represents the response from enable_thumbnail
type EnableThumbnailResponse struct {
	api.BaseResponse
	Data struct {
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
	} `json:"data"`
}

// EnableThumbnailOptions contains options for enabling thumbnail
type EnableThumbnailOptions struct {
	Path    string `json:"path,omitempty"`    // Path to enable thumbnail for
	Rebuild bool   `json:"rebuild,omitempty"` // Rebuild existing thumbnails
}

// EnableThumbnail enables thumbnail generation
func (fs *FileStationService) EnableThumbnail(ctx context.Context, options *EnableThumbnailOptions) (*EnableThumbnailResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "enable_thumbnail",
		"sid":  sid,
	}

	if options != nil {
		if options.Path != "" {
			params["path"] = options.Path
		}
		if options.Rebuild {
			params["rebuild"] = "1"
		}
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result EnableThumbnailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse enable thumbnail response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// SetSmbThumbResponse represents the response from set_smb_thumb
type SetSmbThumbResponse struct {
	api.BaseResponse
	Data struct {
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
	} `json:"data"`
}

// SetSmbThumbOptions contains options for setting SMB thumbnail
type SetSmbThumbOptions struct {
	Enabled bool   `json:"enabled,omitempty"` // Enable/disable SMB thumbnail
	Path    string `json:"path,omitempty"`    // Path to apply setting
}

// SetSmbThumb sets SMB thumbnail settings
func (fs *FileStationService) SetSmbThumb(ctx context.Context, options *SetSmbThumbOptions) (*SetSmbThumbResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "set_smb_thumb",
		"sid":  sid,
	}

	if options != nil {
		if options.Enabled {
			params["enabled"] = "1"
		} else {
			params["enabled"] = "0"
		}
		if options.Path != "" {
			params["path"] = options.Path
		}
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result SetSmbThumbResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse set smb thumb response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// GetViewerResponse represents the response from get_viewer
type GetViewerResponse struct {
	api.BaseResponse
	Data struct {
		Viewers []ViewerInfo `json:"viewers"`
		Total   int          `json:"total"`
	} `json:"data"`
}

// ViewerInfo represents information about a viewer
type ViewerInfo struct {
	Name        string   `json:"name"`        // Viewer name
	Type        string   `json:"type"`        // Viewer type: image, video, pdf, etc.
	Description string   `json:"description"` // Description
	Enabled     bool     `json:"enabled"`     // Whether enabled
	Extensions  []string `json:"extensions"`  // Supported file extensions
}

// GetViewer gets available viewers
func (fs *FileStationService) GetViewer(ctx context.Context) (*GetViewerResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_viewer",
		"sid":  sid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetViewerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse get viewer response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// GetViewerSupportFormatResponse represents the response from get_viewer_support_format_t
type GetViewerSupportFormatResponse struct {
	api.BaseResponse
	Data struct {
		Formats []ViewerFormat `json:"formats"`
		Total   int            `json:"total"`
	} `json:"data"`
}

// ViewerFormat represents a viewer format
type ViewerFormat struct {
	Viewer     string   `json:"viewer"`     // Viewer name
	Extensions []string `json:"extensions"` // Supported extensions
	MimeTypes  []string `json:"mime_types"` // Supported MIME types
}

// GetViewerSupportFormat gets supported formats for viewers
func (fs *FileStationService) GetViewerSupportFormat(ctx context.Context) (*GetViewerSupportFormatResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_viewer_support_format_t",
		"sid":  sid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetViewerSupportFormatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse get viewer support format response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// GetTextFileResponse represents the response from get_text_file
type GetTextFileResponse struct {
	api.BaseResponse
	Data struct {
		Content string `json:"content"`
		Size    int64  `json:"size"`
		Path    string `json:"path"`
	} `json:"data"`
}

// GetTextFileOptions contains options for getting text file
type GetTextFileOptions struct {
	Encoding string `json:"encoding,omitempty"` // File encoding
	Offset   int64  `json:"offset,omitempty"`   // Start offset
	Limit    int64  `json:"limit,omitempty"`    // Number of bytes to read
}

// GetTextFile gets content of a text file
func (fs *FileStationService) GetTextFile(ctx context.Context, path string, options *GetTextFileOptions) (*GetTextFileResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_text_file",
		"sid":  sid,
		"path": path,
	}

	if options != nil {
		if options.Encoding != "" {
			params["encoding"] = options.Encoding
		}
		if options.Offset > 0 {
			params["offset"] = fmt.Sprintf("%d", options.Offset)
		}
		if options.Limit > 0 {
			params["limit"] = fmt.Sprintf("%d", options.Limit)
		}
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetTextFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse get text file response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// SaveTextFileResponse represents the response from save_text_file
type SaveTextFileResponse struct {
	api.BaseResponse
	Data struct {
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
		Path    string `json:"path"`
	} `json:"data"`
}

// SaveTextFileOptions contains options for saving text file
type SaveTextFileOptions struct {
	Encoding string `json:"encoding,omitempty"` // File encoding
	Mode     string `json:"mode,omitempty"`     // Write mode: overwrite, append
}

// SaveTextFile saves content to a text file
func (fs *FileStationService) SaveTextFile(ctx context.Context, path, content string, options *SaveTextFileOptions) (*SaveTextFileResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":    "save_text_file",
		"sid":     sid,
		"path":    path,
		"content": content,
	}

	if options != nil {
		if options.Encoding != "" {
			params["encoding"] = options.Encoding
		}
		if options.Mode != "" {
			params["mode"] = options.Mode
		}
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result SaveTextFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse save text file response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}
