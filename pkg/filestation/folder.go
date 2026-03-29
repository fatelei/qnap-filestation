package filestation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// ListFolders lists folders in a directory
func (fs *FileStationService) ListFolders(ctx context.Context, path string, options *ListOptions) ([]File, error) {
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
		"filetype":    "dir",
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

// CreateFolder creates a new folder
func (fs *FileStationService) CreateFolder(ctx context.Context, path string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/filestation/file.cgi"
	params := map[string]string{
		"api":         "SYNO.FileStation.CreateFolder",
		"method":      "create",
		"version":     "2",
		"folder_path": path,
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

// DeleteFolder deletes a folder
func (fs *FileStationService) DeleteFolder(ctx context.Context, path string) error {
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

// RenameFolder renames a folder
func (fs *FileStationService) RenameFolder(ctx context.Context, oldPath, newPath string) error {
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
