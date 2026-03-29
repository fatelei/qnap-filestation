package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// IntBool is a helper type that unmarshals JSON values that may be
// encoded as int/bool/string into an integer 0/1 representation.
type IntBool int

// UnmarshalJSON implements json.Unmarshaler to accept 1/0, true/false, or
// "true"/"false"/"1"/"0" for compatibility with inconsistent APIs.
func (b *IntBool) UnmarshalJSON(data []byte) error {
	// Try boolean
	var vb bool
	if err := json.Unmarshal(data, &vb); err == nil {
		if vb {
			*b = 1
		} else {
			*b = 0
		}
		return nil
	}
	// Try integer
	var vi int
	if err := json.Unmarshal(data, &vi); err == nil {
		if vi != 0 {
			*b = 1
		} else {
			*b = 0
		}
		return nil
	}
	// Try string forms
	var vs string
	if err := json.Unmarshal(data, &vs); err == nil {
		s := strings.ToLower(strings.TrimSpace(vs))
		switch s {
		case "true", "1", "yes", "y":
			*b = 1
			return nil
		case "false", "0", "no", "n":
			*b = 0
			return nil
		}
		return fmt.Errorf("invalid IntBool string %q", vs)
	}
	return fmt.Errorf("invalid IntBool: %s", string(data))
}

// BaseResponse is the standard response structure from QNAP API
type BaseResponse struct {
	Success IntBool `json:"success"`
	Code    int     `json:"error_code,omitempty"`
	Message string  `json:"error_msg,omitempty"`
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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, v); err != nil {
		return err
	}

	return nil
}
