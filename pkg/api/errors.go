package api

import "fmt"

// ErrorCode represents QNAP API error codes
type ErrorCode int

const (
	// General errors
	ErrUnknown        ErrorCode = 0
	ErrSuccess        ErrorCode = 1
	ErrAuthFailed     ErrorCode = 1000
	ErrInvalidSID     ErrorCode = 1001
	ErrSessionExpired ErrorCode = 1002
	ErrInvalidParams  ErrorCode = 1003

	// File operation errors
	ErrNotFound       ErrorCode = 2001
	ErrAlreadyExists  ErrorCode = 2002
	ErrPermission     ErrorCode = 2003
	ErrInvalidPath    ErrorCode = 2004
	ErrNotEmpty       ErrorCode = 2005
	ErrQuotaExceeded  ErrorCode = 2006

	// Network errors
	ErrNetwork        ErrorCode = 3001
	ErrTimeout        ErrorCode = 3002
)

// APIError represents an error returned by the QNAP API
type APIError struct {
	Code      ErrorCode
	Message   string
	Detail    string
	Err       error
	RequestID string
}

func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("API error %d: %s (detail: %s)", e.Code, e.Message, e.Detail)
	}
	if e.Err != nil {
		return fmt.Sprintf("API error %d: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

// IsAuthError returns true if the error is an authentication error
func (e *APIError) IsAuthError() bool {
	return e.Code == ErrAuthFailed || e.Code == ErrInvalidSID || e.Code == ErrSessionExpired
}

// IsNotFound returns true if the error is a not found error
func (e *APIError) IsNotFound() bool {
	return e.Code == ErrNotFound
}

// IsPermissionError returns true if the error is a permission error
func (e *APIError) IsPermissionError() bool {
	return e.Code == ErrPermission
}

// NewAPIError creates a new APIError
func NewAPIError(code ErrorCode, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

// WrapAPIError wraps an error with additional API context
func WrapAPIError(code ErrorCode, message string, err error) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}
