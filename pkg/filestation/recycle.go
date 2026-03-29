package filestation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// TrashRecoveryResponse represents the response from trash_recovery
type TrashRecoveryResponse struct {
	api.BaseResponse
	Data struct {
		PID     string `json:"pid"` // Process ID for tracking
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
	} `json:"data"`
}

// TrashRecoveryOptions contains options for trash recovery
type TrashRecoveryOptions struct {
	TaskID     string `json:"task_id,omitempty"`     // Task ID for status tracking
	Overwrite  bool   `json:"overwrite,omitempty"`   // Overwrite existing files
	DestPath   string `json:"dest_path,omitempty"`   // Destination path for recovery
	SourcePath string `json:"source_path,omitempty"` // Source path in trash
}

// TrashRecovery recovers files from trash
func (fs *FileStationService) TrashRecovery(ctx context.Context, sourcePath string, files []string, options *TrashRecoveryOptions) (*TrashRecoveryResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if len(files) == 0 {
		return nil, api.NewAPIError(api.ErrInvalidPath, "at least one file is required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":        "trash_recovery",
		"sid":         sid,
		"source_path": sourcePath,
		"file_total":  fmt.Sprintf("%d", len(files)),
	}

	for i, file := range files {
		key := fmt.Sprintf("source_file[%d]", i)
		params[key] = file
	}

	if options != nil {
		if options.TaskID != "" {
			params["task_id"] = options.TaskID
		}
		if options.Overwrite {
			params["overwrite"] = "1"
		}
		if options.DestPath != "" {
			params["dest_path"] = options.DestPath
		}
		if options.SourcePath != "" {
			params["source_path"] = options.SourcePath
		}
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result TrashRecoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse trash recovery response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// CancelTrashRecoveryResponse represents the response from cancel_trash_recovery
type CancelTrashRecoveryResponse struct {
	api.BaseResponse
	Data struct {
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
	} `json:"data"`
}

// CancelTrashRecovery cancels an ongoing trash recovery operation
func (fs *FileStationService) CancelTrashRecovery(ctx context.Context, taskID string) (*CancelTrashRecoveryResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if taskID == "" {
		return nil, api.NewAPIError(api.ErrInvalidParams, "task ID is required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":    "cancel_trash_recovery",
		"sid":     sid,
		"task_id": taskID,
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result CancelTrashRecoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse cancel trash recovery response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// GetRecycleBinStatusResponse represents the response from get_recycle_bin_status
type GetRecycleBinStatusResponse struct {
	api.BaseResponse
	Data struct {
		Enabled     bool               `json:"enabled"`
		VolumeCount int                `json:"volume_count"`
		Volumes     []RecycleBinVolume `json:"volumes,omitempty"`
	} `json:"data"`
}

// RecycleBinVolume represents recycle bin information for a volume
type RecycleBinVolume struct {
	VolumeName string `json:"volume_name"`
	Enabled    bool   `json:"enabled"`
	Path       string `json:"path"`
	ItemCount  int    `json:"item_count,omitempty"`
	TotalSize  int64  `json:"total_size,omitempty"`
}

// GetRecycleBinStatusOptions contains options for getting recycle bin status
type GetRecycleBinStatusOptions struct {
	VolumeName string `json:"volume_name,omitempty"` // Specific volume name
}

// GetRecycleBinStatus gets the status of recycle bin
func (fs *FileStationService) GetRecycleBinStatus(ctx context.Context, options *GetRecycleBinStatusOptions) (*GetRecycleBinStatusResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_recycle_bin_status",
		"sid":  sid,
	}

	if options != nil && options.VolumeName != "" {
		params["volume_name"] = options.VolumeName
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetRecycleBinStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse get recycle bin status response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// EmptyTrashResponse represents the response from empty_trash
type EmptyTrashResponse struct {
	api.BaseResponse
	Data struct {
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
	} `json:"data"`
}

// EmptyTrashOptions contains options for emptying trash
type EmptyTrashOptions struct {
	VolumeName string `json:"volume_name,omitempty"` // Specific volume to empty
}

// EmptyTrash empties the recycle bin
func (fs *FileStationService) EmptyTrash(ctx context.Context, options *EmptyTrashOptions) (*EmptyTrashResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "empty_trash",
		"sid":  sid,
	}

	if options != nil && options.VolumeName != "" {
		params["volume_name"] = options.VolumeName
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result EmptyTrashResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse empty trash response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// SetDeletePermanentlyResponse represents the response from set_delete_permanently
type SetDeletePermanentlyResponse struct {
	api.BaseResponse
	Data struct {
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
	} `json:"data"`
}

// SetDeletePermanentlyOptions contains options for setting delete permanently
type SetDeletePermanentlyOptions struct {
	Enabled    bool   `json:"enabled"`               // Enable/disable permanent delete
	VolumeName string `json:"volume_name,omitempty"` // Specific volume name
}

// SetDeletePermanently sets whether files are deleted permanently or moved to trash
func (fs *FileStationService) SetDeletePermanently(ctx context.Context, options *SetDeletePermanentlyOptions) (*SetDeletePermanentlyResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "set_delete_permanently",
		"sid":  sid,
	}

	if options != nil {
		if options.Enabled {
			params["enabled"] = "1"
		} else {
			params["enabled"] = "0"
		}
		if options.VolumeName != "" {
			params["volume_name"] = options.VolumeName
		}
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result SetDeletePermanentlyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse set delete permanently response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// GetDeleteStatusResponse represents the response from get_delete_status
type GetDeleteStatusResponse struct {
	api.BaseResponse
	Data struct {
		PID         string  `json:"pid"`             // Process ID
		Status      string  `json:"status"`          // Status: running, finished, failed
		Progress    float64 `json:"progress"`        // Progress percentage (0-100)
		TotalCount  int     `json:"total_count"`     // Total files to delete
		Processed   int     `json:"processed"`       // Processed files
		FailedCount int     `json:"failed_count"`    // Failed files
		Error       string  `json:"error,omitempty"` // Error message if failed
	} `json:"data"`
}

// GetDeleteStatusOptions contains options for getting delete status
type GetDeleteStatusOptions struct {
	TaskID string `json:"task_id,omitempty"` // Task ID to check
}

// GetDeleteStatus gets the status of a delete operation
func (fs *FileStationService) GetDeleteStatus(ctx context.Context, options *GetDeleteStatusOptions) (*GetDeleteStatusResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_delete_status",
		"sid":  sid,
	}

	if options != nil && options.TaskID != "" {
		params["task_id"] = options.TaskID
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result GetDeleteStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse get delete status response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}
