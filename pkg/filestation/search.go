package filestation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// SearchResponse represents the response from search operations
type SearchResponse struct {
	api.BaseResponse
	Data struct {
		Items  []File `json:"items"`
		Total  int    `json:"total"`
		Offset int    `json:"offset"`
	} `json:"data"`
}

// Search searches for files and folders
func (fs *FileStationService) Search(ctx context.Context, path string, options *SearchOptions) ([]File, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/filestation/find.cgi"
	params := map[string]string{
		"api":         "SYNO.FileStation.Find",
		"method":      "start",
		"version":     "2",
		"folder_path": path,
	}

	if options != nil {
		if options.Pattern != "" {
			params["pattern"] = options.Pattern
		}
		if options.FileType != "" {
			params["filetype"] = options.FileType
		}
		if len(options.Extension) > 0 {
			for _, ext := range options.Extension {
				params["extension"] = fmt.Sprintf("%s,%s", params["extension"], ext)
			}
		}
		if options.SizeMin > 0 {
			params["size_min"] = fmt.Sprintf("%d", options.SizeMin)
		}
		if options.SizeMax > 0 {
			params["size_max"] = fmt.Sprintf("%d", options.SizeMax)
		}
		if options.Recursive {
			params["recursive"] = "true"
		}
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse search response", err)
	}

	if !searchResp.IsSuccess() {
		return nil, &api.APIError{
			Code:    searchResp.GetErrorCode(),
			Message: searchResp.Message,
		}
	}

	return searchResp.Data.Items, nil
}

// SearchByPattern searches files by pattern
func (fs *FileStationService) SearchByPattern(ctx context.Context, path, pattern string) ([]File, error) {
	return fs.Search(ctx, path, &SearchOptions{
		Pattern:   pattern,
		Recursive: true,
	})
}
