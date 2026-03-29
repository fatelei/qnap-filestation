package filestation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// CloudStatusResponse represents the response from cloud status query
type CloudStatusResponse struct {
	api.BaseResponse
	Data struct {
		IsEnabled       bool    `json:"is_enabled"`
		ConnectedClouds []Cloud `json:"connected_clouds,omitempty"`
		Total           int     `json:"total"`
	} `json:"data"`
}

// Cloud represents a cloud storage connection
type Cloud struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`   // e.g., "dropbox", "google_drive", "onedrive"
	Status       string `json:"status"` // e.g., "connected", "disconnected"
	Account      string `json:"account,omitempty"`
	LastSyncTime string `json:"last_sync_time,omitempty"`
}

// CloudStatus gets the cloud storage status
func (fs *FileStationService) CloudStatus(ctx context.Context) (*CloudStatusResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "cloud",
		"sid":  sid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result CloudStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse cloud status response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// RemoteFolderSubfunc specifies the sub-function for remote folder operations
type RemoteFolderSubfunc string

const (
	// RemoteFolderSubfuncCreateShare creates a remote folder share
	RemoteFolderSubfuncCreateShare RemoteFolderSubfunc = "create_share"
	// RemoteFolderSubfuncModify modifies a remote folder
	RemoteFolderSubfuncModify RemoteFolderSubfunc = "modify"
	// RemoteFolderSubfuncDelete deletes a remote folder
	RemoteFolderSubfuncDelete RemoteFolderSubfunc = "delete"
)

// RemoteFolderOptions contains options for remote folder operations
type RemoteFolderOptions struct {
	Subfunc   RemoteFolderSubfunc `json:"subfunc"`
	Path      string              `json:"path,omitempty"`       // Required for create_share, modify, delete
	Name      string              `json:"name,omitempty"`       // Required for create_share, modify
	CloudType string              `json:"cloud_type,omitempty"` // e.g., "dropbox", "google_drive"
	ShareID   string              `json:"share_id,omitempty"`   // Required for modify, delete
}

// RemoteFolderResponse represents the response from remote folder operations
type RemoteFolderResponse struct {
	api.BaseResponse
	Data struct {
		ShareID   string `json:"share_id,omitempty"`
		Path      string `json:"path,omitempty"`
		Name      string `json:"name,omitempty"`
		CloudType string `json:"cloud_type,omitempty"`
		URL       string `json:"url,omitempty"`
	} `json:"data"`
}

// RemoteFolder performs remote folder operations
func (fs *FileStationService) RemoteFolder(ctx context.Context, options *RemoteFolderOptions) (*RemoteFolderResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if options == nil {
		return nil, api.NewAPIError(api.ErrInvalidParams, "options are required")
	}

	if options.Subfunc == "" {
		return nil, api.NewAPIError(api.ErrInvalidParams, "subfunc is required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":    "remote_folder",
		"subfunc": string(options.Subfunc),
		"sid":     sid,
	}

	if options.Path != "" {
		params["path"] = options.Path
	}
	if options.Name != "" {
		params["name"] = options.Name
	}
	if options.CloudType != "" {
		params["cloud_type"] = options.CloudType
	}
	if options.ShareID != "" {
		params["share_id"] = options.ShareID
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result RemoteFolderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse remote folder response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// CloudSyncStatus represents the status of cloud sync
type CloudSyncStatus struct {
	CloudID      string  `json:"cloud_id"`
	CloudName    string  `json:"cloud_name"`
	CloudType    string  `json:"cloud_type"`
	Status       string  `json:"status"`   // e.g., "syncing", "synced", "error"
	Progress     float64 `json:"progress"` // 0-100
	TotalFiles   int     `json:"total_files"`
	SyncedFiles  int     `json:"synced_files"`
	FailedFiles  int     `json:"failed_files"`
	LastSyncTime string  `json:"last_sync_time"`
	NextSyncTime string  `json:"next_sync_time,omitempty"`
	ErrorMessage string  `json:"error_message,omitempty"`
}

// GetCloudSyncStatusResponse represents the response from cloud sync status query
type GetCloudSyncStatusResponse struct {
	api.BaseResponse
	Data struct {
		SyncStatus []CloudSyncStatus `json:"sync_status"`
		Total      int               `json:"total"`
	} `json:"data"`
}

// GetCloudSyncStatus gets the cloud sync status
func (fs *FileStationService) GetCloudSyncStatus(ctx context.Context) (*GetCloudSyncStatusResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_cloud_sync_status",
		"sid":  sid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetCloudSyncStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse cloud sync status response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// MountIsoOptions contains options for mounting ISO files
type MountIsoOptions struct {
	ISOPath    string `json:"iso_path"`              // Required: path to ISO file
	MountPoint string `json:"mount_point,omitempty"` // Optional: custom mount point
}

// MountIsoResponse represents the response from mount ISO operation
type MountIsoResponse struct {
	api.BaseResponse
	Data struct {
		MountPoint string `json:"mount_point"`
		Device     string `json:"device,omitempty"`
		Status     string `json:"status"`
	} `json:"data"`
}

// MountIso mounts an ISO file
func (fs *FileStationService) MountIso(ctx context.Context, options *MountIsoOptions) (*MountIsoResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if options == nil || options.ISOPath == "" {
		return nil, api.NewAPIError(api.ErrInvalidPath, "ISO path is required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":     "mount_iso",
		"sid":      sid,
		"iso_path": options.ISOPath,
	}

	if options.MountPoint != "" {
		params["mount_point"] = options.MountPoint
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result MountIsoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse mount ISO response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// UnmountIsoResponse represents the response from unmount ISO operation
type UnmountIsoResponse struct {
	api.BaseResponse
	Data struct {
		MountPoint string `json:"mount_point"`
		Status     string `json:"status"`
	} `json:"data"`
}

// UnmountIso unmounts an ISO file
func (fs *FileStationService) UnmountIso(ctx context.Context, mountPoint string) (*UnmountIsoResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if mountPoint == "" {
		return nil, api.NewAPIError(api.ErrInvalidPath, "mount point is required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":        "unmount_iso",
		"sid":         sid,
		"mount_point": mountPoint,
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result UnmountIsoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse unmount ISO response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// MountQdffOptions contains options for mounting QDFF files
type MountQdffOptions struct {
	QdffPath   string `json:"qdff_path"`             // Required: path to QDFF file
	MountPoint string `json:"mount_point,omitempty"` // Optional: custom mount point
	ReadOnly   bool   `json:"read_only,omitempty"`   // Optional: mount as read-only
}

// MountQdffResponse represents the response from mount QDFF operation
type MountQdffResponse struct {
	api.BaseResponse
	Data struct {
		MountPoint string `json:"mount_point"`
		Device     string `json:"device,omitempty"`
		Status     string `json:"status"`
	} `json:"data"`
}

// MountQdff mounts a QDFF file
func (fs *FileStationService) MountQdff(ctx context.Context, options *MountQdffOptions) (*MountQdffResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if options == nil || options.QdffPath == "" {
		return nil, api.NewAPIError(api.ErrInvalidPath, "QDFF path is required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "mount_qdff",
		"sid":       sid,
		"qdff_path": options.QdffPath,
	}

	if options.MountPoint != "" {
		params["mount_point"] = options.MountPoint
	}
	if options.ReadOnly {
		params["read_only"] = "true"
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result MountQdffResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse mount QDFF response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// UnmountQdffResponse represents the response from unmount QDFF operation
type UnmountQdffResponse struct {
	api.BaseResponse
	Data struct {
		MountPoint string `json:"mount_point"`
		Status     string `json:"status"`
	} `json:"data"`
}

// UnmountQdff unmounts a QDFF file
func (fs *FileStationService) UnmountQdff(ctx context.Context, mountPoint string) (*UnmountQdffResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if mountPoint == "" {
		return nil, api.NewAPIError(api.ErrInvalidPath, "mount point is required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":        "unmount_qdff",
		"sid":         sid,
		"mount_point": mountPoint,
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result UnmountQdffResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse unmount QDFF response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// ExternalDiskDisconnectResponse represents the response from external disk disconnect operation
type ExternalDiskDisconnectResponse struct {
	api.BaseResponse
	Data struct {
		DiskPath string `json:"disk_path"`
		Status   string `json:"status"`
	} `json:"data"`
}

// ExternalDiskDisconnect disconnects an external disk
func (fs *FileStationService) ExternalDiskDisconnect(ctx context.Context, diskPath string) (*ExternalDiskDisconnectResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if diskPath == "" {
		return nil, api.NewAPIError(api.ErrInvalidPath, "disk path is required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "external_disk_disconnect",
		"sid":       sid,
		"disk_path": diskPath,
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result ExternalDiskDisconnectResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse external disk disconnect response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// HostType represents a host type
type HostType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"` // e.g., "windows", "mac", "linux"
}

// GetHostTypeListResponse represents the response from host type list query
type GetHostTypeListResponse struct {
	api.BaseResponse
	Data struct {
		HostTypes []HostType `json:"host_types"`
		Total     int        `json:"total"`
	} `json:"data"`
}

// GetHostTypeList gets the list of host types
func (fs *FileStationService) GetHostTypeList(ctx context.Context) (*GetHostTypeListResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_host_type_list",
		"sid":  sid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetHostTypeListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse host type list response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// DomainIP represents a domain IP
type DomainIP struct {
	ID       string `json:"id"`
	IP       string `json:"ip"`
	Hostname string `json:"hostname,omitempty"`
	Status   string `json:"status"` // e.g., "online", "offline"
}

// GetDomainIPListResponse represents the response from domain IP list query
type GetDomainIPListResponse struct {
	api.BaseResponse
	Data struct {
		DomainIPs []DomainIP `json:"domain_ips"`
		Total     int        `json:"total"`
	} `json:"data"`
}

// GetDomainIPList gets the list of domain IPs
func (fs *FileStationService) GetDomainIPList(ctx context.Context) (*GetDomainIPListResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_domain_ip_list",
		"sid":  sid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetDomainIPListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse domain IP list response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// DomainIPEx represents extended domain IP information
type DomainIPEx struct {
	ID         string `json:"id"`
	IP         string `json:"ip"`
	Hostname   string `json:"hostname,omitempty"`
	Status     string `json:"status"` // e.g., "online", "offline"
	MACAddress string `json:"mac_address,omitempty"`
	OS         string `json:"os,omitempty"` // Operating system
	Workgroup  string `json:"workgroup,omitempty"`
	Domain     string `json:"domain,omitempty"`
	LastSeen   string `json:"last_seen,omitempty"`
	IsOnline   bool   `json:"is_online"`
}

// GetDomainIPListExOptions contains options for extended domain IP list query
type GetDomainIPListExOptions struct {
	HostType string `json:"host_type,omitempty"` // Filter by host type
	Status   string `json:"status,omitempty"`    // Filter by status
	Limit    int    `json:"limit,omitempty"`     // Limit results
	Offset   int    `json:"offset,omitempty"`    // Offset for pagination
}

// GetDomainIPListExResponse represents the response from extended domain IP list query
type GetDomainIPListExResponse struct {
	api.BaseResponse
	Data struct {
		DomainIPs []DomainIPEx `json:"domain_ips"`
		Total     int          `json:"total"`
	} `json:"data"`
}

// GetDomainIPListEx gets extended domain IP information
func (fs *FileStationService) GetDomainIPListEx(ctx context.Context, options *GetDomainIPListExOptions) (*GetDomainIPListExResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_domain_ip_list_ex",
		"sid":  sid,
	}

	if options != nil {
		if options.HostType != "" {
			params["host_type"] = options.HostType
		}
		if options.Status != "" {
			params["status"] = options.Status
		}
		if options.Limit > 0 {
			params["limit"] = fmt.Sprintf("%d", options.Limit)
		}
		if options.Offset > 0 {
			params["offset"] = fmt.Sprintf("%d", options.Offset)
		}
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetDomainIPListExResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse domain IP list ex response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}
