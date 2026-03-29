package filestation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// DaemonListResponse represents the response from daemon_list
type DaemonListResponse struct {
	api.BaseResponse
	Data struct {
		Daemons []DaemonInfo `json:"daemons"`
		Total   int          `json:"total"`
	} `json:"data"`
}

// DaemonInfo represents information about a daemon process
type DaemonInfo struct {
	PID       string `json:"pid"`        // Process ID
	Name      string `json:"name"`       // Daemon name
	Status    string `json:"status"`     // Status: running, stopped, etc.
	CPU       string `json:"cpu"`        // CPU usage
	Memory    string `json:"memory"`     // Memory usage
	Uptime    string `json:"uptime"`     // Uptime
	Command   string `json:"command"`    // Command line
	User      string `json:"user"`       // User running the daemon
	AutoStart bool   `json:"auto_start"` // Auto-start enabled
}

// DaemonList lists daemon processes
func (fs *FileStationService) DaemonList(ctx context.Context) (*DaemonListResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "daemon_list",
		"sid":  sid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result DaemonListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse daemon list response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// GetCayinMediaStatusResponse represents the response from get_cayin_media_status
type GetCayinMediaStatusResponse struct {
	api.BaseResponse
	Data struct {
		Enabled     bool   `json:"enabled"`      // Cayin media enabled
		Status      string `json:"status"`       // Status: online, offline, error
		Version     string `json:"version"`      // Software version
		DeviceID    string `json:"device_id"`    // Device ID
		IPAddress   string `json:"ip_address"`   // IP address
		MACAddress  string `json:"mac_address"`  // MAC address
		Uptime      string `json:"uptime"`       // Uptime
		LastSync    string `json:"last_sync"`    // Last sync time
		StorageUsed string `json:"storage_used"` // Storage used
		Description string `json:"description"`  // Description
	} `json:"data"`
}

// GetCayinMediaStatus gets Cayin media status
func (fs *FileStationService) GetCayinMediaStatus(ctx context.Context) (*GetCayinMediaStatusResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_cayin_media_status",
		"sid":  sid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetCayinMediaStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse cayin media status response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// QcloudNotifyInfoResponse represents the response from qcloud_notify_info
type QcloudNotifyInfoResponse struct {
	api.BaseResponse
	Data struct {
		Enabled      bool     `json:"enabled"`       // Notifications enabled
		EmailEnabled bool     `json:"email_enabled"` // Email notifications
		SMSEnabled   bool     `json:"sms_enabled"`   // SMS notifications
		PushEnabled  bool     `json:"push_enabled"`  // Push notifications
		Emails       []string `json:"emails"`        // Email addresses
		PhoneNumbers []string `json:"phone_numbers"` // Phone numbers
		EventTypes   []string `json:"event_types"`   // Event types to notify
		Frequency    string   `json:"frequency"`     // Notification frequency
		LastNotified string   `json:"last_notified"` // Last notification time
	} `json:"data"`
}

// QcloudNotifyInfo gets Qcloud notification info
func (fs *FileStationService) QcloudNotifyInfo(ctx context.Context) (*QcloudNotifyInfoResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "qcloud_notify_info",
		"sid":  sid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result QcloudNotifyInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse qcloud notify info response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// QcloudWopiUrlResponse represents the response from qcloud_wopi_url
type QcloudWopiUrlResponse struct {
	api.BaseResponse
	Data struct {
		URL        string `json:"url"`        // WOPI server URL
		Enabled    bool   `json:"enabled"`    // WOPI enabled
		Version    string `json:"version"`    // WOPI version
		Discovery  string `json:"discovery"`  // Discovery URL
		Zone       string `json:"zone"`       // Zone
		Expiration string `json:"expiration"` // Expiration time
	} `json:"data"`
}

// QcloudWopiUrlOptions contains options for getting WOPI URL
type QcloudWopiUrlOptions struct {
	FileID   string `json:"file_id,omitempty"`   // File ID
	FileName string `json:"file_name,omitempty"` // File name
	Action   string `json:"action,omitempty"`    // Action: view, edit
	Timeout  int    `json:"timeout,omitempty"`   // Timeout in seconds
}

// QcloudWopiUrl gets Qcloud WOPI URL
func (fs *FileStationService) QcloudWopiUrl(ctx context.Context, options *QcloudWopiUrlOptions) (*QcloudWopiUrlResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "qcloud_wopi_url",
		"sid":  sid,
	}

	if options != nil {
		if options.FileID != "" {
			params["file_id"] = options.FileID
		}
		if options.FileName != "" {
			params["file_name"] = options.FileName
		}
		if options.Action != "" {
			params["action"] = options.Action
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

	var result QcloudWopiUrlResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse qcloud wopi url response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// QdmcResponse represents the response from qdmc
type QdmcResponse struct {
	api.BaseResponse
	Data struct {
		Success   bool   `json:"success"`   // Operation success
		Message   string `json:"message"`   // Response message
		Operation string `json:"operation"` // Operation performed
		Status    string `json:"status"`    // Current status
	} `json:"data"`
}

// QdmcOptions contains options for QDMC operations
type QdmcOptions struct {
	Action   string `json:"action,omitempty"`   // Action to perform
	Target   string `json:"target,omitempty"`   // Target path/ID
	Mode     string `json:"mode,omitempty"`     // Operation mode
	Override bool   `json:"override,omitempty"` // Override existing settings
}

// Qdmc performs QDMC operations
func (fs *FileStationService) Qdmc(ctx context.Context, options *QdmcOptions) (*QdmcResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "qdmc",
		"sid":  sid,
	}

	if options != nil {
		if options.Action != "" {
			params["action"] = options.Action
		}
		if options.Target != "" {
			params["target"] = options.Target
		}
		if options.Mode != "" {
			params["mode"] = options.Mode
		}
		if options.Override {
			params["override"] = "1"
		}
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result QdmcResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse qdmc response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// QhamRetrieveResponse represents the response from qham_retrieve
type QhamRetrieveResponse struct {
	api.BaseResponse
	Data struct {
		Success     bool     `json:"success"`      // Operation success
		Message     string   `json:"message"`      // Response message
		Items       []string `json:"items"`        // Retrieved items
		Count       int      `json:"count"`        // Item count
		CacheStatus string   `json:"cache_status"` // Cache status
		Timestamp   string   `json:"timestamp"`    // Retrieval timestamp
	} `json:"data"`
}

// QhamRetrieveOptions contains options for QHAM retrieve operations
type QhamRetrieveOptions struct {
	Source      string `json:"source,omitempty"`      // Source to retrieve from
	Destination string `json:"destination,omitempty"` // Destination path
	Mode        string `json:"mode,omitempty"`        // Retrieval mode
	Refresh     bool   `json:"refresh,omitempty"`     // Force refresh
	Limit       int    `json:"limit,omitempty"`       // Item limit
}

// QhamRetrieve performs QHAM retrieve operations
func (fs *FileStationService) QhamRetrieve(ctx context.Context, options *QhamRetrieveOptions) (*QhamRetrieveResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "qham_retrieve",
		"sid":  sid,
	}

	if options != nil {
		if options.Source != "" {
			params["source"] = options.Source
		}
		if options.Destination != "" {
			params["destination"] = options.Destination
		}
		if options.Mode != "" {
			params["mode"] = options.Mode
		}
		if options.Refresh {
			params["refresh"] = "1"
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

	var result QhamRetrieveResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse qham retrieve response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// QrpacResponse represents the response from qrpac
type QrpacResponse struct {
	api.BaseResponse
	Data struct {
		Success    bool   `json:"success"`     // Operation success
		Message    string `json:"message"`     // Response message
		Action     string `json:"action"`      // Action performed
		StatusCode int    `json:"status_code"` // Status code
		RequestID  string `json:"request_id"`  // Request ID
	} `json:"data"`
}

// QrpacOptions contains options for QRPAC operations
type QrpacOptions struct {
	Action     string            `json:"action,omitempty"`     // Action to perform
	Parameters map[string]string `json:"parameters,omitempty"` // Additional parameters
	Async      bool              `json:"async,omitempty"`      // Async operation
}

// Qrpac performs QRPAC operations
func (fs *FileStationService) Qrpac(ctx context.Context, options *QrpacOptions) (*QrpacResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "qrpac",
		"sid":  sid,
	}

	if options != nil {
		if options.Action != "" {
			params["action"] = options.Action
		}
		if options.Async {
			params["async"] = "1"
		}
		// Add additional parameters
		for key, value := range options.Parameters {
			params[key] = value
		}
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result QrpacResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse qrpac response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// HwtsResponse represents the response from hwts
type HwtsResponse struct {
	api.BaseResponse
	Data struct {
		Success     bool     `json:"success"`     // Operation success
		Message     string   `json:"message"`     // Response message
		Status      string   `json:"status"`      // System status
		Temperature string   `json:"temperature"` // Temperature
		PowerState  string   `json:"power_state"` // Power state
		Health      string   `json:"health"`      // Health status
		LastCheck   string   `json:"last_check"`  // Last check time
		Alerts      []string `json:"alerts"`      // Active alerts
	} `json:"data"`
}

// HwtsOptions contains options for HWTS operations
type HwtsOptions struct {
	Action    string `json:"action,omitempty"`    // Action to perform
	Detail    bool   `json:"detail,omitempty"`    // Include detailed info
	Refresh   bool   `json:"refresh,omitempty"`   // Force refresh
	Component string `json:"component,omitempty"` // Specific component
}

// Hwts performs HWTS operations
func (fs *FileStationService) Hwts(ctx context.Context, options *HwtsOptions) (*HwtsResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "hwts",
		"sid":  sid,
	}

	if options != nil {
		if options.Action != "" {
			params["action"] = options.Action
		}
		if options.Detail {
			params["detail"] = "1"
		}
		if options.Refresh {
			params["refresh"] = "1"
		}
		if options.Component != "" {
			params["component"] = options.Component
		}
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result HwtsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse hwts response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// Stub functions for undocumented operations

// UnsupportedOperationResponse represents response for unsupported operations
type UnsupportedOperationResponse struct {
	api.BaseResponse
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
}

// G is a stub for the undocumented 'g' function
func (fs *FileStationService) G(ctx context.Context, params map[string]string) (*UnsupportedOperationResponse, error) {
	logger := fs.client.GetLogger()
	logger.Warn("unsupported operation called", "operation", "g")

	return &UnsupportedOperationResponse{
		BaseResponse: api.BaseResponse{
			Success: 0,
			Code:    int(api.ErrUnknown),
			Message: "operation 'g' is not supported",
		},
	}, nil
}

// L is a stub for the undocumented 'l' function
func (fs *FileStationService) L(ctx context.Context, params map[string]string) (*UnsupportedOperationResponse, error) {
	logger := fs.client.GetLogger()
	logger.Warn("unsupported operation called", "operation", "l")

	return &UnsupportedOperationResponse{
		BaseResponse: api.BaseResponse{
			Success: 0,
			Code:    int(api.ErrUnknown),
			Message: "operation 'l' is not supported",
		},
	}, nil
}

// SetUnderscore is a stub for the undocumented 'set_' function
func (fs *FileStationService) SetUnderscore(ctx context.Context, params map[string]string) (*UnsupportedOperationResponse, error) {
	logger := fs.client.GetLogger()
	logger.Warn("unsupported operation called", "operation", "set_")

	return &UnsupportedOperationResponse{
		BaseResponse: api.BaseResponse{
			Success: 0,
			Code:    int(api.ErrUnknown),
			Message: "operation 'set_' is not supported",
		},
	}, nil
}

// SetP is a stub for the undocumented 'set_p' function
func (fs *FileStationService) SetP(ctx context.Context, params map[string]string) (*UnsupportedOperationResponse, error) {
	logger := fs.client.GetLogger()
	logger.Warn("unsupported operation called", "operation", "set_p")

	return &UnsupportedOperationResponse{
		BaseResponse: api.BaseResponse{
			Success: 0,
			Code:    int(api.ErrUnknown),
			Message: "operation 'set_p' is not supported",
		},
	}, nil
}

// GetS is a stub for the undocumented 'get_s' function
func (fs *FileStationService) GetS(ctx context.Context, params map[string]string) (*UnsupportedOperationResponse, error) {
	logger := fs.client.GetLogger()
	logger.Warn("unsupported operation called", "operation", "get_s")

	return &UnsupportedOperationResponse{
		BaseResponse: api.BaseResponse{
			Success: 0,
			Code:    int(api.ErrUnknown),
			Message: "operation 'get_s' is not supported",
		},
	}, nil
}

// GetR is a stub for the undocumented 'get_r' function
func (fs *FileStationService) GetR(ctx context.Context, params map[string]string) (*UnsupportedOperationResponse, error) {
	logger := fs.client.GetLogger()
	logger.Warn("unsupported operation called", "operation", "get_r")

	return &UnsupportedOperationResponse{
		BaseResponse: api.BaseResponse{
			Success: 0,
			Code:    int(api.ErrUnknown),
			Message: "operation 'get_r' is not supported",
		},
	}, nil
}

// GetUnderscore is a stub for the undocumented 'get_' function
func (fs *FileStationService) GetUnderscore(ctx context.Context, params map[string]string) (*UnsupportedOperationResponse, error) {
	logger := fs.client.GetLogger()
	logger.Warn("unsupported operation called", "operation", "get_")

	return &UnsupportedOperationResponse{
		BaseResponse: api.BaseResponse{
			Success: 0,
			Code:    int(api.ErrUnknown),
			Message: "operation 'get_' is not supported",
		},
	}, nil
}

// Func is a stub for the undocumented 'func' function
func (fs *FileStationService) Func(ctx context.Context, params map[string]string) (*UnsupportedOperationResponse, error) {
	logger := fs.client.GetLogger()
	logger.Warn("unsupported operation called", "operation", "func")

	return &UnsupportedOperationResponse{
		BaseResponse: api.BaseResponse{
			Success: 0,
			Code:    int(api.ErrUnknown),
			Message: "operation 'func' is not supported",
		},
	}, nil
}

// Dryru is a stub for the undocumented 'dryru' function
func (fs *FileStationService) Dryru(ctx context.Context, params map[string]string) (*UnsupportedOperationResponse, error) {
	logger := fs.client.GetLogger()
	logger.Warn("unsupported operation called", "operation", "dryru")

	return &UnsupportedOperationResponse{
		BaseResponse: api.BaseResponse{
			Success: 0,
			Code:    int(api.ErrUnknown),
			Message: "operation 'dryru' is not supported",
		},
	}, nil
}

// Umo is a stub for the undocumented 'umo' function
func (fs *FileStationService) Umo(ctx context.Context, params map[string]string) (*UnsupportedOperationResponse, error) {
	logger := fs.client.GetLogger()
	logger.Warn("unsupported operation called", "operation", "umo")

	return &UnsupportedOperationResponse{
		BaseResponse: api.BaseResponse{
			Success: 0,
			Code:    int(api.ErrUnknown),
			Message: "operation 'umo' is not supported",
		},
	}, nil
}

// Mou is a stub for the undocumented 'mou' function
func (fs *FileStationService) Mou(ctx context.Context, params map[string]string) (*UnsupportedOperationResponse, error) {
	logger := fs.client.GetLogger()
	logger.Warn("unsupported operation called", "operation", "mou")

	return &UnsupportedOperationResponse{
		BaseResponse: api.BaseResponse{
			Success: 0,
			Code:    int(api.ErrUnknown),
			Message: "operation 'mou' is not supported",
		},
	}, nil
}

// ShareUnderscore is a stub for the undocumented 'share_' function
func (fs *FileStationService) ShareUnderscore(ctx context.Context, params map[string]string) (*UnsupportedOperationResponse, error) {
	logger := fs.client.GetLogger()
	logger.Warn("unsupported operation called", "operation", "share_")

	return &UnsupportedOperationResponse{
		BaseResponse: api.BaseResponse{
			Success: 0,
			Code:    int(api.ErrUnknown),
			Message: "operation 'share_' is not supported",
		},
	}, nil
}
