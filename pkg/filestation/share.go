package filestation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// ShareListResponse represents the response from listing share links
type ShareListResponse struct {
	api.BaseResponse
	Data struct {
		Shares []ShareLink `json:"shares"`
		Total  int         `json:"total"`
	} `json:"data"`
}

// ShareCreateResponse represents the response from creating a share link
type ShareCreateResponse struct {
	api.BaseResponse
	Data struct {
		Shares []ShareLink `json:"shares"`
	} `json:"data"`
}

// ListShareLinks lists all share links
func (fs *FileStationService) ListShareLinks(ctx context.Context) ([]ShareLink, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/filestation/share.cgi"
	params := map[string]string{
		"api":     "SYNO.FileStation.Sharing",
		"method":  "list",
		"version": "2",
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var listResp ShareListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse share list response", err)
	}

	if !listResp.IsSuccess() {
		return nil, &api.APIError{
			Code:    listResp.GetErrorCode(),
			Message: listResp.Message,
		}
	}

	return listResp.Data.Shares, nil
}

// CreateShareLink creates a new share link
func (fs *FileStationService) CreateShareLink(ctx context.Context, path string, options *ShareLinkOptions) (*ShareLink, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/filestation/share.cgi"
	params := map[string]string{
		"api":     "SYNO.FileStation.Sharing",
		"method":  "create",
		"version": "2",
		"path":    path,
	}

	if options != nil {
		if !options.Expires.IsZero() {
			params["expired_date"] = fmt.Sprintf("%d", options.Expires.Unix())
		}
		if options.Password != "" {
			params["password"] = options.Password
		}
		if options.Writeable {
			params["writeable"] = "true"
		}
		if options.Validity > 0 {
			params["validity"] = fmt.Sprintf("%d", options.Validity)
		}
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var createResp ShareCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse create share response", err)
	}

	if !createResp.IsSuccess() {
		return nil, &api.APIError{
			Code:    createResp.GetErrorCode(),
			Message: createResp.Message,
		}
	}

	if len(createResp.Data.Shares) == 0 {
		return nil, api.NewAPIError(api.ErrUnknown, "no share link created")
	}

	return &createResp.Data.Shares[0], nil
}

// DeleteShareLink deletes a share link
func (fs *FileStationService) DeleteShareLink(ctx context.Context, id string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/filestation/share.cgi"
	params := map[string]string{
		"api":     "SYNO.FileStation.Sharing",
		"method":  "delete",
		"version": "2",
		"id":      id,
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result api.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse delete share response", err)
	}

	if !result.IsSuccess() {
		return &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return nil
}
