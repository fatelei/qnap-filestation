package api

import (
	"encoding/json"
	"io"
	"net/http"
)

// BaseResponse is the standard response structure from QNAP API
type BaseResponse struct {
	Success int    `json:"success"`
	Code    int    `json:"error_code,omitempty"`
	Message string `json:"error_msg,omitempty"`
}

// IsSuccess returns true if the response indicates success
func (r *BaseResponse) IsSuccess() bool {
	return r.Success == 1
}

// GetErrorCode returns the error code
func (r *BaseResponse) GetErrorCode() ErrorCode {
	return ErrorCode(r.Code)
}

// APIResponse wraps a response with data
type APIResponse[T any] struct {
	BaseResponse
	Data T `json:"data,omitempty"`
}

// ResponseParser handles parsing of API responses
type ResponseParser struct{}

// NewResponseParser creates a new ResponseParser
func NewResponseParser() *ResponseParser {
	return &ResponseParser{}
}

// ParseJSON parses a JSON response into the provided interface
func (rp *ResponseParser) ParseJSON(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, v); err != nil {
		return err
	}

	return nil
}

// ParseResponse parses a response and checks for API errors
func (rp *ResponseParser) ParseResponse(resp *http.Response, v interface{}) error {
	// First parse as base response to check for errors
	var baseResp BaseResponse
	if err := rp.ParseJSON(resp, &baseResp); err != nil {
		return err
	}

	// Check for API errors
	if !baseResp.IsSuccess() {
		return &APIError{
			Code:    baseResp.GetErrorCode(),
			Message: baseResp.Message,
		}
	}

	// If there's data to parse, parse the full response
	if v != nil {
		// Re-read body for full parsing
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(body, v); err != nil {
			return err
		}
	}

	return nil
}

// ParseListResponse parses a list response
func (rp *ResponseParser) ParseListResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, v); err != nil {
		return err
	}

	return nil
}
