package filestation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// ListResponse represents the response from list operations
type ListResponse struct {
	api.BaseResponse
	Data struct {
		Items  []File `json:"items"`
		Total  int    `json:"total"`
		Offset int    `json:"offset"`
	} `json:"data"`
}

// ListFiles lists files in a directory
func (fs *FileStationService) ListFiles(ctx context.Context, path string, options *ListOptions) ([]File, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/filestation/list.cgi"
	params := map[string]string{
		"api":         "SYNO.FileStation.List",
		"method":      "get",
		"version":     "2",
		"folder_path": path,
	}

	if options != nil {
		if options.Offset > 0 {
			params["offset"] = fmt.Sprintf("%d", options.Offset)
		}
		if options.Limit > 0 {
			params["limit"] = fmt.Sprintf("%d", options.Limit)
		}
		if options.SortBy != "" {
			params["sort_by"] = options.SortBy
		}
		if options.SortOrder != "" {
			params["sort_order"] = options.SortOrder
		}
		if options.FileType != "" {
			params["filetype"] = options.FileType
		}
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var listResp ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if !listResp.IsSuccess() {
		return nil, &api.APIError{
			Code:    listResp.GetErrorCode(),
			Message: listResp.Message,
		}
	}

	return listResp.Data.Items, nil
}

// GetFileInfo gets information about a specific file
func (fs *FileStationService) GetFileInfo(ctx context.Context, path string) (*File, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/filestation/stat.cgi"
	params := map[string]string{
		"api":     "SYNO.FileStation.Info",
		"method":  "getinfo",
		"version": "2",
		"path":    path,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		api.BaseResponse
		Data struct {
			Files []File `json:"files"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	if len(result.Data.Files) == 0 {
		return nil, api.NewAPIError(api.ErrNotFound, "file not found")
	}

	return &result.Data.Files[0], nil
}

// DeleteFile deletes a file
func (fs *FileStationService) DeleteFile(ctx context.Context, path string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/filestation/file.cgi"
	params := map[string]string{
		"api":     "SYNO.FileStation.Delete",
		"method":  "delete",
		"version": "2",
		"path":    path,
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result api.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if !result.IsSuccess() {
		return &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return nil
}

// RenameFile renames a file
func (fs *FileStationService) RenameFile(ctx context.Context, oldPath, newPath string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/filestation/file.cgi"
	params := map[string]string{
		"api":     "SYNO.FileStation.Rename",
		"method":  "rename",
		"version": "2",
		"path":    oldPath,
		"name":    newPath,
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result api.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if !result.IsSuccess() {
		return &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return nil
}

// CopyFile copies a file to a destination
func (fs *FileStationService) CopyFile(ctx context.Context, source, dest string, options *CopyMoveOptions) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/filestation/file.cgi"
	params := map[string]string{
		"api":            "SYNO.FileStation.CopyMove",
		"method":         "copy",
		"version":        "2",
		"path":           source,
		"dest_folder_path": dest,
	}

	if options != nil && options.Overwrite {
		params["overwrite"] = "true"
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result api.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if !result.IsSuccess() {
		return &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return nil
}

// MoveFile moves a file to a destination
func (fs *FileStationService) MoveFile(ctx context.Context, source, dest string, options *CopyMoveOptions) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/filestation/file.cgi"
	params := map[string]string{
		"api":            "SYNO.FileStation.CopyMove",
		"method":         "move",
		"version":        "2",
		"path":           source,
		"dest_folder_path": dest,
	}

	if options != nil && options.Overwrite {
		params["overwrite"] = "true"
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result api.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if !result.IsSuccess() {
		return &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return nil
}
