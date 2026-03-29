package filestation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// Transcode Types

// TranscodeJob represents a transcode job
type TranscodeJob struct {
	PID         string    `json:"pid"`          // Process ID
	SourceFile  string    `json:"source_file"`  // Source video file
	OutputPath  string    `json:"output_path"`  // Output transcoded file path
	Codec       string    `json:"codec"`        // Target codec (h264, h265, etc.)
	Resolution  string    `json:"resolution"`   // Target resolution (720p, 1080p, etc.)
	Bitrate     int       `json:"bitrate"`      // Target bitrate (kbps)
	Status      string    `json:"status"`       // Status: pending, running, finished, failed
	Progress    float64   `json:"progress"`     // Progress percentage (0-100)
	StartTime   time.Time `json:"start_time"`   // Job start time
	EndTime     time.Time `json:"end_time"`     // Job end time
	Error       string    `json:"error,omitempty"` // Error message if failed
	FileSize    int64     `json:"file_size"`    // Source file size
	Processed   int64     `json:"processed"`    // Processed bytes
}

// EstTranscodeOptions contains options for starting a transcode job
type EstTranscodeOptions struct {
	SourceFile string `json:"source_file"` // Source video file path
	OutputPath string `json:"output_path"` // Output transcoded file path
	Codec      string `json:"codec"`       // Target codec (h264, h265, vp9, etc.)
	Resolution string `json:"resolution"`  // Target resolution (720p, 1080p, 4k, etc.)
	Bitrate    int    `json:"bitrate"`     // Target bitrate in kbps
	Framerate  int    `json:"framerate"`   // Target framerate (optional)
	AudioCodec string `json:"audio_codec"` // Audio codec (aac, mp3, etc.)
}

// TranscodeStatusResponse represents the response from transcode status query
type TranscodeStatusResponse struct {
	api.BaseResponse
	Data TranscodeJob `json:"data"`
}

// VideoQueueStatus represents the video transcode queue status
type VideoQueueStatus struct {
	Jobs       []TranscodeJob `json:"jobs"`       // List of jobs in queue
	Total      int            `json:"total"`      // Total number of jobs
	Running    int            `json:"running"`    // Number of running jobs
	Pending    int            `json:"pending"`    // Number of pending jobs
	Finished   int            `json:"finished"`   // Number of finished jobs
	Failed     int            `json:"failed"`     // Number of failed jobs
	MaxConcurrent int         `json:"max_concurrent"` // Max concurrent jobs
}

// VideoQueueStatusResponse represents the response from get_video_qstatus
type VideoQueueStatusResponse struct {
	api.BaseResponse
	Data VideoQueueStatus `json:"data"`
}

// VideoFolderMonitorOptions contains options for video folder monitoring
type VideoFolderMonitorOptions struct {
	Path            string `json:"path"`             // Folder path to monitor
	Recursive       bool   `json:"recursive"`        // Monitor subdirectories
	AutoTranscode   bool   `json:"auto_transcode"`   // Auto-transcode new videos
	TranscodeOptions *EstTranscodeOptions `json:"transcode_options,omitempty"` // Transcode settings
}

// VideoFolderMonitorResponse represents the response from video_folder_monitor
type VideoFolderMonitorResponse struct {
	api.BaseResponse
	Data struct {
		MonitorID string `json:"monitor_id"` // Monitor ID
		Enabled   bool   `json:"enabled"`    // Monitor enabled
		Path      string `json:"path"`       // Monitored path
	} `json:"data"`
}

// VideoMlQueueOptions contains options for ML-based video processing
type VideoMlQueueOptions struct {
	SourceFile   string `json:"source_file"`   // Source video file
	Operation    string `json:"operation"`     // Operation: scene_detection, object_detection, etc.
	Model        string `json:"model"`         // ML model to use
	Confidence   float64 `json:"confidence"`   // Confidence threshold (0-1)
}

// VideoMlQueueResponse represents the response from video_ml_queue
type VideoMlQueueResponse struct {
	api.BaseResponse
	Data struct {
		JobID   string `json:"job_id"`   // ML job ID
		Status  string `json:"status"`   // Job status
		Message string `json:"message"`  // Status message
	} `json:"data"`
}

// SubtitleOptions contains options for subtitle operations
type SubtitleOptions struct {
	SourceFile  string `json:"source_file"`  // Video file path
	SubtitleFile string `json:"subtitle_file"` // Subtitle file path (srt, vtt, etc.)
	Language    string `json:"language"`     // Subtitle language
	Encoding    string `json:"encoding"`     // Subtitle encoding
	Offset      int    `json:"offset"`       // Subtitle offset in milliseconds
}

// SubtitleResponse represents the response from subtitle operations
type SubtitleResponse struct {
	api.BaseResponse
	Data struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Path    string `json:"path"`
	} `json:"data"`
}

// DiscoOptions contains options for video discovery operations
type DiscoOptions struct {
	Path      string `json:"path"`       // Path to scan
	Recursive bool   `json:"recursive"`  // Scan recursively
	FileType  string `json:"file_type"`  // File type filter (optional)
	MetaOnly  bool   `json:"meta_only"`  // Return metadata only
}

// DiscoResponse represents the response from disco operations
type DiscoResponse struct {
	api.BaseResponse
	Data struct {
		Files    []DiscoFile `json:"files"`    // Discovered files
		Total    int         `json:"total"`    // Total files found
		Duration int         `json:"duration"` // Scan duration in ms
	} `json:"data"`
}

// DiscoFile represents a discovered video file
type DiscoFile struct {
	Path        string         `json:"path"`         // File path
	Size        int64          `json:"size"`         // File size
	Duration    int            `json:"duration"`     // Video duration in seconds
	Resolution  string         `json:"resolution"`   // Video resolution
	Bitrate     int            `json:"bitrate"`      // Video bitrate
	Codec       string         `json:"codec"`        // Video codec
	AudioCodec  string         `json:"audio_codec"`  // Audio codec
	Thumbnail   string         `json:"thumbnail"`    // Thumbnail URL
	Metadata    map[string]string `json:"metadata"`  // Additional metadata
}

// DryrunOptions contains options for transcode dry run
type DryrunOptions struct {
	SourceFile string `json:"source_file"` // Source video file
	Codec      string `json:"codec"`       // Target codec
	Resolution string `json:"resolution"`  // Target resolution
	Bitrate    int    `json:"bitrate"`     // Target bitrate
}

// DryrunResponse represents the response from dryrun operation
type DryrunResponse struct {
	api.BaseResponse
	Data struct {
		Feasible      bool   `json:"feasible"`       // Transcode is feasible
		EstimatedTime int    `json:"estimated_time"` // Estimated time in seconds
		EstimatedSize int64  `json:"estimated_size"` // Estimated output size
		Warnings      []string `json:"warnings"`     // Warning messages
		Recommendations []string `json:"recommendations"` // Recommendations
	} `json:"data"`
}

// EstTranscode starts a video transcoding job
func (fs *FileStationService) EstTranscode(ctx context.Context, options *EstTranscodeOptions) (string, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return "", api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if options == nil {
		return "", api.NewAPIError(api.ErrInvalidParams, "transcode options required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":        "est_transcode",
		"sid":         sid,
		"source_file": options.SourceFile,
		"output_path": options.OutputPath,
		"codec":       options.Codec,
	}

	if options.Resolution != "" {
		params["resolution"] = options.Resolution
	}
	if options.Bitrate > 0 {
		params["bitrate"] = fmt.Sprintf("%d", options.Bitrate)
	}
	if options.Framerate > 0 {
		params["framerate"] = fmt.Sprintf("%d", options.Framerate)
	}
	if options.AudioCodec != "" {
		params["audio_codec"] = options.AudioCodec
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		api.BaseResponse
		PID string `json:"pid"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", api.WrapAPIError(api.ErrUnknown, "failed to parse est transcode response", err)
	}

	if !result.IsSuccess() {
		return "", &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return result.PID, nil
}

// KillTranscode kills a running transcode job
func (fs *FileStationService) KillTranscode(ctx context.Context, pid string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if pid == "" {
		return api.NewAPIError(api.ErrInvalidParams, "process ID required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "kill_transcode",
		"sid":  sid,
		"pid":  pid,
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result api.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse kill transcode response", err)
	}

	if !result.IsSuccess() {
		return &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return nil
}

// DeleteTranscode deletes a transcode job
func (fs *FileStationService) DeleteTranscode(ctx context.Context, pid string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if pid == "" {
		return api.NewAPIError(api.ErrInvalidParams, "process ID required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "delete_transcode",
		"sid":  sid,
		"pid":  pid,
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result api.BaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return api.WrapAPIError(api.ErrUnknown, "failed to parse delete transcode response", err)
	}

	if !result.IsSuccess() {
		return &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return nil
}

// GetVideoQStatus gets the transcode queue status
func (fs *FileStationService) GetVideoQStatus(ctx context.Context) (*VideoQueueStatus, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_video_qstatus",
		"sid":  sid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result VideoQueueStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse get video qstatus response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result.Data, nil
}

// VideoFolderMonitor sets up video folder monitoring
func (fs *FileStationService) VideoFolderMonitor(ctx context.Context, options *VideoFolderMonitorOptions) (*VideoFolderMonitorResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if options == nil {
		return nil, api.NewAPIError(api.ErrInvalidParams, "monitor options required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "video_folder_monitor",
		"sid":  sid,
		"path": options.Path,
	}

	if options.Recursive {
		params["recursive"] = "1"
	}
	if options.AutoTranscode {
		params["auto_transcode"] = "1"
		if options.TranscodeOptions != nil {
			if options.TranscodeOptions.Codec != "" {
				params["codec"] = options.TranscodeOptions.Codec
			}
			if options.TranscodeOptions.Resolution != "" {
				params["resolution"] = options.TranscodeOptions.Resolution
			}
			if options.TranscodeOptions.Bitrate > 0 {
				params["bitrate"] = fmt.Sprintf("%d", options.TranscodeOptions.Bitrate)
			}
		}
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result VideoFolderMonitorResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse video folder monitor response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// VideoMlQueue adds a video to ML-based processing queue
func (fs *FileStationService) VideoMlQueue(ctx context.Context, options *VideoMlQueueOptions) (*VideoMlQueueResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if options == nil {
		return nil, api.NewAPIError(api.ErrInvalidParams, "ML queue options required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":        "video_ml_queue",
		"sid":         sid,
		"source_file": options.SourceFile,
		"operation":   options.Operation,
		"model":       options.Model,
	}

	if options.Confidence > 0 {
		params["confidence"] = fmt.Sprintf("%.2f", options.Confidence)
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result VideoMlQueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse video ml queue response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// Subtitle performs subtitle operations
func (fs *FileStationService) Subtitle(ctx context.Context, operation string, options *SubtitleOptions) (*SubtitleResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if options == nil {
		return nil, api.NewAPIError(api.ErrInvalidParams, "subtitle options required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":        "subtitle",
		"sid":         sid,
		"operation":   operation,
		"source_file": options.SourceFile,
	}

	if options.SubtitleFile != "" {
		params["subtitle_file"] = options.SubtitleFile
	}
	if options.Language != "" {
		params["language"] = options.Language
	}
	if options.Encoding != "" {
		params["encoding"] = options.Encoding
	}
	if options.Offset != 0 {
		params["offset"] = fmt.Sprintf("%d", options.Offset)
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SubtitleResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse subtitle response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// Disco performs video discovery operations
func (fs *FileStationService) Disco(ctx context.Context, operation string, options *DiscoOptions) (*DiscoResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if options == nil {
		return nil, api.NewAPIError(api.ErrInvalidParams, "disco options required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":      "disco",
		"sid":       sid,
		"operation": operation,
		"path":      options.Path,
	}

	if options.Recursive {
		params["recursive"] = "1"
	}
	if options.FileType != "" {
		params["file_type"] = options.FileType
	}
	if options.MetaOnly {
		params["meta_only"] = "1"
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result DiscoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse disco response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}

// Dryrun performs a transcode test without actual processing
func (fs *FileStationService) Dryrun(ctx context.Context, options *DryrunOptions) (*DryrunResponse, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if options == nil {
		return nil, api.NewAPIError(api.ErrInvalidParams, "dryrun options required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":        "dryrun",
		"sid":         sid,
		"source_file": options.SourceFile,
		"codec":       options.Codec,
	}

	if options.Resolution != "" {
		params["resolution"] = options.Resolution
	}
	if options.Bitrate > 0 {
		params["bitrate"] = fmt.Sprintf("%d", options.Bitrate)
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result DryrunResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, api.WrapAPIError(api.ErrUnknown, "failed to parse dryrun response", err)
	}

	if !result.IsSuccess() {
		return nil, &api.APIError{
			Code:    result.GetErrorCode(),
			Message: result.Message,
		}
	}

	return &result, nil
}
