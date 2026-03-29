package filestation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// SetACLControl sets ACL control for a share/folder
func (fs *FileStationService) SetACLControl(ctx context.Context, options *SetACLOptions) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "set_acl_control",
		"sid":       sid,
		"sharename": options.ShareName,
	}

	if options.Root != "" {
		params["root"] = options.Root
	}
	if options.Recursive {
		params["recursive"] = "1"
	}

	// Add ACL entries
	for i, acl := range options.ACLs {
		prefix := fmt.Sprintf("acl_%d_", i)
		params[prefix+"user"] = acl.User
		params[prefix+"domain"] = acl.Domain
		params[prefix+"isuser"] = fmt.Sprintf("%d", boolToInt(acl.IsUser))
		params[prefix+"right"] = acl.Right
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
		return api.WrapAPIError(api.ErrUnknown, "failed to parse set ACL response", err)
	}

	if result.Status != 1 || result.Success != "true" {
		return api.NewAPIError(api.ErrUnknown, "failed to set ACL control")
	}

	return nil
}

// GetACLControl gets ACL control settings for a share/folder
func (fs *FileStationService) GetACLControl(ctx context.Context, shareName, root string) (*ACLControl, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "get_acl_control",
		"sid":       sid,
		"sharename": shareName,
	}

	if root != "" {
		params["root"] = root
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var aclResp ACLControlResponse
	if err := json.NewDecoder(resp.Body).Decode(&aclResp); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse ACL control response", err)
	}

	if aclResp.Status != 1 {
		return nil, api.NewAPIError(api.ErrUnknown, "failed to get ACL control")
	}

	return &aclResp.Data, nil
}

// GetACLUserGroupList gets list of users and groups for ACL configuration
func (fs *FileStationService) GetACLUserGroupList(ctx context.Context, shareName string) (*ACLUserGroupList, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "get_acl_user_group_list_out",
		"sid":       sid,
		"sharename": shareName,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var listResp ACLUserGroupListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse ACL user/group list response", err)
	}

	if listResp.Status != 1 {
		return nil, api.NewAPIError(api.ErrUnknown, "failed to get ACL user/group list")
	}

	return &listResp.Data, nil
}

// SetPrivilege sets file/folder privileges
func (fs *FileStationService) SetPrivilege(ctx context.Context, options *SetPrivilegeOptions) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "set_privilege",
		"sid":       sid,
		"sharename": options.ShareName,
	}

	if options.Path != "" {
		params["path"] = options.Path
	}
	if options.Recursive {
		params["recursive"] = "1"
	}

	// Add privilege entries
	for i, priv := range options.Privileges {
		prefix := fmt.Sprintf("priv_%d_", i)
		params[prefix+"user"] = priv.User
		params[prefix+"domain"] = priv.Domain
		params[prefix+"isuser"] = fmt.Sprintf("%d", boolToInt(priv.IsUser))
		params[prefix+"right"] = priv.Right
		params[prefix+"isfile"] = fmt.Sprintf("%d", boolToInt(priv.IsFile))
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
		return api.WrapAPIError(api.ErrUnknown, "failed to parse set privilege response", err)
	}

	if result.Status != 1 || result.Success != "true" {
		return api.NewAPIError(api.ErrUnknown, "failed to set privilege")
	}

	return nil
}

// GetAccessRight gets access rights for a file/folder
func (fs *FileStationService) GetAccessRight(ctx context.Context, path string) (*AccessRight, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	// Validate and parse path to get share name and relative path
	if strings.TrimSpace(path) == "" || path == "/" {
		return nil, api.NewAPIError(api.ErrInvalidParams, "invalid path")
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	shareName := parts[0]
	if shareName == "" {
		return nil, api.NewAPIError(api.ErrInvalidParams, "invalid path")
	}
	relativePath := "/" + strings.Join(parts[1:], "/")
	if relativePath == "/" {
		relativePath = ""
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "get_access_right",
		"sid":       sid,
		"sharename": shareName,
	}

	if relativePath != "" {
		params["path"] = relativePath
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var accessResp AccessRightResponse
	if err := json.NewDecoder(resp.Body).Decode(&accessResp); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse access right response", err)
	}

	if accessResp.Status != 1 {
		return nil, api.NewAPIError(api.ErrUnknown, "failed to get access right")
	}

	return &accessResp.Data, nil
}

// SetProjectionType sets the projection type for a share
func (fs *FileStationService) SetProjectionType(ctx context.Context, shareName, projectionType string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":           "set_projection_type",
		"sid":            sid,
		"sharename":      shareName,
		"projectionType": projectionType,
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
		return api.WrapAPIError(api.ErrUnknown, "failed to parse set projection type response", err)
	}

	if result.Status != 1 || result.Success != "true" {
		return api.NewAPIError(api.ErrUnknown, "failed to set projection type")
	}

	return nil
}

// boolToInt converts boolean to integer (1 for true, 0 for false)
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
