package filestation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// CompressResponse represents the response from compression operation
type CompressResponse struct {
	api.BaseResponse
	PID string `json:"pid"`
}

// ExtractResponse represents the response from extraction operation
type ExtractResponse struct {
	api.BaseResponse
	PID string `json:"pid"`
}

// CompressFiles compresses files/folders into an archive
func (fs *FileStationService) CompressFiles(ctx context.Context, options *CompressOptions) (string, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return "", api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if options == nil {
		return "", api.NewAPIError(api.ErrInvalidParams, "compress options required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":         "compress",
		"sid":          sid,
		"source_file":  strings.Join(options.SourceFiles, ","),
		"source_path":  options.SourcePath,
		"compress_name": options.CompressName,
	}

	if options.Level > 0 {
		params["level"] = fmt.Sprintf("%d", options.Level)
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var compressResp CompressResponse
	if err := json.NewDecoder(resp.Body).Decode(&compressResp); err != nil {
		return "", api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if !compressResp.IsSuccess() {
		return "", &api.APIError{
			Code:    compressResp.GetErrorCode(),
			Message: compressResp.Message,
		}
	}

	return compressResp.PID, nil
}

// CancelCompress cancels an ongoing compression operation
func (fs *FileStationService) CancelCompress(ctx context.Context, pid string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if pid == "" {
		return api.NewAPIError(api.ErrInvalidParams, "process ID required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "cancel_compress",
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

// GetCompressStatus gets the status of a compression operation
func (fs *FileStationService) GetCompressStatus(ctx context.Context, pid string) (*CompressStatus, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if pid == "" {
		return nil, api.NewAPIError(api.ErrInvalidParams, "process ID required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_compress_status",
		"sid":  sid,
		"pid":  pid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		api.BaseResponse
		Data CompressStatus `json:"data"`
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

	return &result.Data, nil
}

// ExtractArchive extracts an archive to a destination path
func (fs *FileStationService) ExtractArchive(ctx context.Context, options *ExtractOptions) (string, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return "", api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if options == nil {
		return "", api.NewAPIError(api.ErrInvalidParams, "extract options required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":         "extract",
		"sid":          sid,
		"extract_file": options.ExtractFile,
		"path":         options.DestPath,
	}

	if options.CodePage != "" {
		params["code_page"] = options.CodePage
	}

	if options.Overwrite {
		params["overwrite"] = "true"
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var extractResp ExtractResponse
	if err := json.NewDecoder(resp.Body).Decode(&extractResp); err != nil {
		return "", api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if !extractResp.IsSuccess() {
		return "", &api.APIError{
			Code:    extractResp.GetErrorCode(),
			Message: extractResp.Message,
		}
	}

	return extractResp.PID, nil
}

// CancelExtract cancels an ongoing extraction operation
func (fs *FileStationService) CancelExtract(ctx context.Context, pid string) error {
	sid := fs.client.GetSID()
	if sid == "" {
		return api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if pid == "" {
		return api.NewAPIError(api.ErrInvalidParams, "process ID required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "cancel_extract",
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

// GetExtractList lists the contents of an archive without extracting
func (fs *FileStationService) GetExtractList(ctx context.Context, archivePath string) ([]ExtractFile, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if archivePath == "" {
		return nil, api.NewAPIError(api.ErrInvalidParams, "archive path required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":         "get_extract_list",
		"sid":          sid,
		"extract_file": archivePath,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		api.BaseResponse
		Datas []ExtractFile `json:"datas"`
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

	return result.Datas, nil
}

// GetExtractStatus gets the status of an extraction operation
func (fs *FileStationService) GetExtractStatus(ctx context.Context, pid string) (*ExtractStatus, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return nil, api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if pid == "" {
		return nil, api.NewAPIError(api.ErrInvalidParams, "process ID required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func": "get_extract_status_ext",
		"sid":  sid,
		"pid":  pid,
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		api.BaseResponse
		Data ExtractStatus `json:"data"`
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

	return &result.Data, nil
}
