package filestation

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// ListFolders lists folders in a directory
func (fs *FileStationService) ListFolders(ctx context.Context, path string, options *ListOptions) ([]File, error) {
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
	defer func() { _ = resp.Body.Close() }()

	var listResp ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	// Filter only folders
	var folders []File
	for _, item := range listResp.Datas {
		if item.IsFolder == 1 {
			folders = append(folders, item)
		}
	}

	return folders, nil
}

// CreateFolder creates a new folder
func (fs *FileStationService) CreateFolder(ctx context.Context, path string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	// Parse path to get parent directory and folder name
	// path format: /parent/foldername
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 1 {
		return api.WrapAPIError(api.ErrUnknown, "invalid path", nil)
	}

	folderName := parts[len(parts)-1]
	destPath := "/" + strings.Join(parts[:len(parts)-1], "/")
	if destPath == "/" {
		destPath = "" // Root path case
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":        "createdir",
		"sid":         sid,
		"dest_folder": folderName,
		"dest_path":   destPath,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Status  int    `json:"status"`
		Success string `json:"success"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if result.Status != 1 || result.Success != "true" {
		return api.NewAPIError(api.ErrUnknown, "failed to create folder")
	}

	return nil
}

// DeleteFolder deletes a folder
func (fs *FileStationService) DeleteFolder(ctx context.Context, path string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	// Split path into parent directory and folder name for QNAP API
	dir, name := filepath.Dir(path), filepath.Base(path)

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":       "delete",
		"sid":        sid,
		"path":       dir,
		"file_total": "1",
		"file_name":  name,
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	var result UtilRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if result.Status != 1 || result.Success != "true" {
		return api.NewAPIError(api.ErrUnknown, "delete operation failed")
	}

	return nil
}

// RenameFolder renames a folder using the QNAP utilRequest.cgi API
func (fs *FileStationService) RenameFolder(ctx context.Context, oldPath, newPath string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	// Split path into parent directory and folder name
	dir := filepath.Dir(oldPath)
	sourceName := filepath.Base(oldPath)
	destName := filepath.Base(newPath)

	return fs.RenameFileUtil(ctx, dir, sourceName, destName)
}
