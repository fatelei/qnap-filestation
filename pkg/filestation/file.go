package filestation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// ListResponse represents the response from QNAP list operations
type ListResponse struct {
	Total     int    `json:"total"`
	RealTotal int    `json:"real_total,omitempty"`
	Datas     []File `json:"datas"`
}

// ListFiles lists files in a directory
func (fs *FileStationService) ListFiles(ctx context.Context, path string, options *ListOptions) ([]File, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "get_list",
		"sid":       sid,
		"path":      path,
		"list_mode": "all",
		"is_iso":    "0",
		"start":     "0",
		"limit":     "100",
		"sort":      "filename",
		"dir":       "ASC",
	}

	if options != nil {
		if options.Offset > 0 {
			params["start"] = fmt.Sprintf("%d", options.Offset)
		}
		if options.Limit > 0 {
			params["limit"] = fmt.Sprintf("%d", options.Limit)
		}
		if options.SortBy != "" {
			params["sort"] = options.SortBy
		}
		if options.SortOrder != "" {
			params["dir"] = options.SortOrder
		}
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

	var listResp ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	return listResp.Datas, nil
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
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			_ = cerr
		}
	}()

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
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			_ = cerr
		}
	}()

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
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			_ = cerr
		}
	}()

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
		"api":              "SYNO.FileStation.CopyMove",
		"method":           "copy",
		"version":          "2",
		"path":             source,
		"dest_folder_path": dest,
	}

	if options != nil && options.Overwrite {
		params["overwrite"] = "true"
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
		"api":              "SYNO.FileStation.CopyMove",
		"method":           "move",
		"version":          "2",
		"path":             source,
		"dest_folder_path": dest,
	}

	if options != nil && options.Overwrite {
		params["overwrite"] = "true"
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

// UtilRequestResponse represents the response from utilRequest.cgi operations
type UtilRequestResponse struct {
	Status  int    `json:"status"`
	Success string `json:"success"`
}

// DeleteFiles deletes one or more files using the utilRequest.cgi endpoint
func (fs *FileStationService) DeleteFiles(ctx context.Context, sourcePath string, sourceFiles []string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if len(sourceFiles) == 0 {
		return api.NewAPIError(api.ErrInvalidPath, "at least one source file is required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"

	// Build parameters - source_file can be repeated for multiple files
	params := map[string]string{
		"func":        "delete",
		"sid":         sid,
		"source_path": sourcePath,
	}

	// Add each source file (will be sent as repeated parameters)
	for i, file := range sourceFiles {
		key := fmt.Sprintf("source_file[%d]", i)
		params[key] = file
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

	var result UtilRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if result.Status != 1 || result.Success != "true" {
		return api.NewAPIError(api.ErrUnknown, "delete operation failed")
	}

	return nil
}

// RenameFileUtil renames a file using the utilRequest.cgi endpoint
func (fs *FileStationService) RenameFileUtil(ctx context.Context, path, sourceName, destName string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if sourceName == "" || destName == "" {
		return api.NewAPIError(api.ErrInvalidPath, "source name and destination name are required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":        "rename",
		"sid":         sid,
		"path":        path,
		"source_name": sourceName,
		"dest_name":   destName,
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

	var result UtilRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if result.Status != 1 || result.Success != "true" {
		return api.NewAPIError(api.ErrUnknown, "rename operation failed")
	}

	return nil
}

// CopyFilesUtil copies one or more files using the utilRequest.cgi endpoint
func (fs *FileStationService) CopyFilesUtil(ctx context.Context, sourcePath, destPath string, sourceFiles []string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if len(sourceFiles) == 0 {
		return api.NewAPIError(api.ErrInvalidPath, "at least one source file is required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"

	// Build parameters - source_file can be repeated for multiple files
	params := map[string]string{
		"func":        "copy",
		"sid":         sid,
		"source_path": sourcePath,
		"dest_path":   destPath,
	}

	// Add each source file (will be sent as repeated parameters)
	for i, file := range sourceFiles {
		key := fmt.Sprintf("source_file[%d]", i)
		params[key] = file
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

	var result UtilRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if result.Status != 1 || result.Success != "true" {
		return api.NewAPIError(api.ErrUnknown, "copy operation failed")
	}

	return nil
}

// MoveFilesUtil moves one or more files using the utilRequest.cgi endpoint
func (fs *FileStationService) MoveFilesUtil(ctx context.Context, sourcePath, destPath string, sourceFiles []string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if len(sourceFiles) == 0 {
		return api.NewAPIError(api.ErrInvalidPath, "at least one source file is required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"

	// Build parameters - source_file can be repeated for multiple files
	params := map[string]string{
		"func":        "move",
		"sid":         sid,
		"source_path": sourcePath,
		"dest_path":   destPath,
	}

	// Add each source file (will be sent as repeated parameters)
	for i, file := range sourceFiles {
		key := fmt.Sprintf("source_file[%d]", i)
		params[key] = file
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

	var result UtilRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if result.Status != 1 || result.Success != "true" {
		return api.NewAPIError(api.ErrUnknown, "move operation failed")
	}

	return nil
}
