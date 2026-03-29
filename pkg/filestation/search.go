package filestation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// SearchResponse represents the response from QNAP search operations
type SearchResponse struct {
	Total int    `json:"total"`
	Datas []File `json:"datas"`
}

// Search searches for files and folders
func (fs *FileStationService) Search(ctx context.Context, path string, options *SearchOptions) ([]File, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":        "search_ext",
		"sid":         sid,
		"folders":     path,
		"folderCount": "1",
		"keyword":     "",
		"searchType":  "0",
		"start":       "0",
		"limit":       "100",
		"sort":        "filename",
		"dir":         "ASC",
		"v":           "1",
	}

	if options != nil {
		if options.Pattern != "" {
			params["keyword"] = options.Pattern
		}
		if options.FileType != "" {
			// Map file types: MUSIC=1, VIDEO=2, PHOTO=3
			typeMap := map[string]string{
				"MUSIC": "1",
				"VIDEO": "2",
				"PHOTO": "3",
			}
			if typeCode, ok := typeMap[options.FileType]; ok {
				params["searchType"] = typeCode
			}
		}
		if len(options.Extension) > 0 {
			params["extensionName"] = options.Extension[0]
		}
		if options.SizeMin > 0 {
			params["fileSizeType"] = "5" // Greater than
			params["fileSize"] = fmt.Sprintf("%d", options.SizeMin)
		}
		if options.SizeMax > 0 {
			params["fileSizeType"] = "6" // Less than
			params["fileSize"] = fmt.Sprintf("%d", options.SizeMax)
		}
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse search response", err)
	}

	return searchResp.Datas, nil
}

// SearchByPattern searches for files by pattern (simplified interface)
func (fs *FileStationService) SearchByPattern(ctx context.Context, path, pattern string) ([]File, error) {
	return fs.Search(ctx, path, &SearchOptions{
		Pattern: pattern,
	})
}

// SearchResult represents the response from async search operations
type SearchResult struct {
	PID     string `json:"pid"`     // Process ID for async operation
	Status  string `json:"status"`  // Status: running, finished, failed
	Total   int    `json:"total"`   // Total results found
	Results []File `json:"results"` // Search results (when available)
}

// SearchAsync starts an async search operation
func (fs *FileStationService) SearchAsync(ctx context.Context, path string, options *SearchOptions) (string, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return "", api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":        "search_start",
		"sid":         sid,
		"folders":     path,
		"folderCount": "1",
		"keyword":     "",
		"searchType":  "0",
		"start":       "0",
		"limit":       "100",
		"sort":        "filename",
		"dir":         "ASC",
		"v":           "1",
	}

	if options != nil {
		if options.Pattern != "" {
			params["keyword"] = options.Pattern
		}
		if options.FileType != "" {
			// Map file types: MUSIC=1, VIDEO=2, PHOTO=3
			typeMap := map[string]string{
				"MUSIC": "1",
				"VIDEO": "2",
				"PHOTO": "3",
			}
			if typeCode, ok := typeMap[options.FileType]; ok {
				params["searchType"] = typeCode
			}
		}
		if len(options.Extension) > 0 {
			params["extensionName"] = options.Extension[0]
		}
		if options.SizeMin > 0 {
			params["fileSizeType"] = "5" // Greater than
			params["fileSize"] = fmt.Sprintf("%d", options.SizeMin)
		}
		if options.SizeMax > 0 {
			params["fileSizeType"] = "6" // Less than
			params["fileSize"] = fmt.Sprintf("%d", options.SizeMax)
		}
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		api.BaseResponse
		Status int    `json:"status"`
		Error  string `json:"error,omitempty"`
		PID    string `json:"pid"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", api.WrapAPIError(api.ErrUnknown, "failed to parse search response", err)
	}

	ok := result.Status == 1 || result.IsSuccess()
	if !ok {
		msg := result.Message
		if msg == "" {
			msg = result.Error
		}
		return "", &api.APIError{Code: result.GetErrorCode(), Message: msg}
	}

	return result.PID, nil
}

// GetSearchResult gets async search results by PID
func (fs *FileStationService) GetSearchResult(ctx context.Context, pid string) (*SearchResult, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if strings.TrimSpace(pid) == "" {
		return nil, api.NewAPIError(api.ErrInvalidParams, "process ID required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_search_result",
		"sid":  sid,
		"pid":  pid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		api.BaseResponse
		Status int          `json:"status"`
		Error  string       `json:"error,omitempty"`
		Data   SearchResult `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse search result response", err)
	}

	ok := result.Status == 1 || result.IsSuccess()
	if !ok {
		msg := result.Message
		if msg == "" {
			msg = result.Error
		}
		return nil, &api.APIError{Code: result.GetErrorCode(), Message: msg}
	}

	return &result.Data, nil
}

// StopSearch cancels an active search operation
func (fs *FileStationService) StopSearch(ctx context.Context, pid string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if strings.TrimSpace(pid) == "" {
		return api.NewAPIError(api.ErrInvalidParams, "process ID required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "search_stop",
		"sid":  sid,
		"pid":  pid,
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		api.BaseResponse
		Status int    `json:"status"`
		Error  string `json:"error,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	ok := result.Status == 1 || result.IsSuccess()
	if !ok {
		msg := result.Message
		if msg == "" {
			msg = result.Error
		}
		return &api.APIError{Code: result.GetErrorCode(), Message: msg}
	}

	return nil
}
