package filestation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// EncryptResponse represents the response from encryption operation
type EncryptResponse struct {
	api.BaseResponse
	PID string `json:"pid"`
}

// DecryptResponse represents the response from decryption operation
type DecryptResponse struct {
	api.BaseResponse
	PID string `json:"pid"`
}

// CipherResponse represents the response from cipher operation
type CipherResponse struct {
	api.BaseResponse
	PID string `json:"pid"`
}

// ChecksumResponse represents the response from checksum operation
type ChecksumResponse struct {
	api.BaseResponse
	Checksum string `json:"checksum"`
}

// EncryptOptions contains options for encryption operations
type EncryptOptions struct {
	SourceFiles []string // List of source files to encrypt
	SourcePath  string   // Source directory path
	Password    string   // Encryption password
	Algorithm   string   // Encryption algorithm (e.g., "aes256")
}

// DecryptOptions contains options for decryption operations
type DecryptOptions struct {
	SourceFiles []string // List of source files to decrypt
	SourcePath  string   // Source directory path
	Password    string   // Decryption password
}

// CipherOptions contains options for cipher operations
type CipherOptions struct {
	SourceFiles []string // List of source files to cipher
	SourcePath  string   // Source directory path
	Action      string   // Cipher action: "encrypt" or "decrypt"
	Password    string   // Cipher password
	Algorithm   string   // Cipher algorithm (e.g., "aes256")
}

// ChecksumOptions contains options for checksum operations
type ChecksumOptions struct {
	SourceFile string // File to calculate checksum for
	SourcePath string // Source directory path
	Algorithm  string // Checksum algorithm: "md5", "sha1", "sha256", "sha512"
}

// EncryptFile encrypts one or more files
func (fs *FileStationService) EncryptFile(ctx context.Context, options *EncryptOptions) (string, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return "", api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if options == nil {
		return "", api.NewAPIError(api.ErrInvalidParams, "encrypt options required")
	}

	if len(options.SourceFiles) == 0 {
		return "", api.NewAPIError(api.ErrInvalidParams, "at least one source file is required")
	}

	if options.Password == "" {
		return "", api.NewAPIError(api.ErrInvalidParams, "password is required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":     "encrypt",
		"sid":      sid,
		"password": options.Password,
	}

	if options.SourcePath != "" {
		params["source_path"] = options.SourcePath
	}

	if options.Algorithm != "" {
		params["algorithm"] = options.Algorithm
	}

	// Add each source file (will be sent as repeated parameters)
	for i, file := range options.SourceFiles {
		key := fmt.Sprintf("source_file[%d]", i)
		params[key] = file
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var encryptResp EncryptResponse
	if err := json.NewDecoder(resp.Body).Decode(&encryptResp); err != nil {
		return "", api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if !encryptResp.IsSuccess() {
		return "", &api.APIError{
			Code:    encryptResp.GetErrorCode(),
			Message: encryptResp.Message,
		}
	}

	return encryptResp.PID, nil
}

// DecryptFile decrypts one or more files
func (fs *FileStationService) DecryptFile(ctx context.Context, options *DecryptOptions) (string, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return "", api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if options == nil {
		return "", api.NewAPIError(api.ErrInvalidParams, "decrypt options required")
	}

	if len(options.SourceFiles) == 0 {
		return "", api.NewAPIError(api.ErrInvalidParams, "at least one source file is required")
	}

	if options.Password == "" {
		return "", api.NewAPIError(api.ErrInvalidParams, "password is required")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":     "decrypt",
		"sid":      sid,
		"password": options.Password,
	}

	if options.SourcePath != "" {
		params["source_path"] = options.SourcePath
	}

	// Add each source file (will be sent as repeated parameters)
	for i, file := range options.SourceFiles {
		key := fmt.Sprintf("source_file[%d]", i)
		params[key] = file
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var decryptResp DecryptResponse
	if err := json.NewDecoder(resp.Body).Decode(&decryptResp); err != nil {
		return "", api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if !decryptResp.IsSuccess() {
		return "", &api.APIError{
			Code:    decryptResp.GetErrorCode(),
			Message: decryptResp.Message,
		}
	}

	return decryptResp.PID, nil
}

// CipherFile performs cipher operations (encrypt/decrypt) on files
func (fs *FileStationService) CipherFile(ctx context.Context, options *CipherOptions) (string, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return "", api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if options == nil {
		return "", api.NewAPIError(api.ErrInvalidParams, "cipher options required")
	}

	if len(options.SourceFiles) == 0 {
		return "", api.NewAPIError(api.ErrInvalidParams, "at least one source file is required")
	}

	if options.Password == "" {
		return "", api.NewAPIError(api.ErrInvalidParams, "password is required")
	}

	if options.Action == "" {
		return "", api.NewAPIError(api.ErrInvalidParams, "action is required (encrypt/decrypt)")
	}

	action := strings.ToLower(options.Action)
	if action != "encrypt" && action != "decrypt" {
		return "", api.NewAPIError(api.ErrInvalidParams, "action must be 'encrypt' or 'decrypt'")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":     "cipher",
		"sid":      sid,
		"action":   action,
		"password": options.Password,
	}

	if options.SourcePath != "" {
		params["source_path"] = options.SourcePath
	}

	if options.Algorithm != "" {
		params["algorithm"] = options.Algorithm
	}

	// Add each source file (will be sent as repeated parameters)
	for i, file := range options.SourceFiles {
		key := fmt.Sprintf("source_file[%d]", i)
		params[key] = file
	}

	resp, err := fs.client.DoRequest(ctx, "POST", endpoint, params, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var cipherResp CipherResponse
	if err := json.NewDecoder(resp.Body).Decode(&cipherResp); err != nil {
		return "", api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if !cipherResp.IsSuccess() {
		return "", &api.APIError{
			Code:    cipherResp.GetErrorCode(),
			Message: cipherResp.Message,
		}
	}

	return cipherResp.PID, nil
}

// ChecksumFile calculates the checksum of a file
func (fs *FileStationService) ChecksumFile(ctx context.Context, options *ChecksumOptions) (string, error) {
	sid := fs.client.GetSID()
	if sid == "" {
		return "", api.WrapAPIError(api.ErrAuthFailed, "not authenticated", nil)
	}

	if options == nil {
		return "", api.NewAPIError(api.ErrInvalidParams, "checksum options required")
	}

	if options.SourceFile == "" {
		return "", api.NewAPIError(api.ErrInvalidParams, "source file is required")
	}

	if options.Algorithm == "" {
		options.Algorithm = "md5" // Default algorithm
	}

	// Validate algorithm
	validAlgorithms := map[string]bool{
		"md5":    true,
		"sha1":   true,
		"sha256": true,
		"sha512": true,
	}
	if !validAlgorithms[options.Algorithm] {
		return "", api.NewAPIError(api.ErrInvalidParams, "invalid algorithm: must be md5, sha1, sha256, or sha512")
	}

	endpoint := "/cgi-bin/filemanager/utilRequest.cgi"
	params := map[string]string{
		"func":        "checksum",
		"sid":         sid,
		"source_file": options.SourceFile,
		"algorithm":   options.Algorithm,
	}

	if options.SourcePath != "" {
		params["source_path"] = options.SourcePath
	}

	resp, err := fs.client.DoRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var checksumResp ChecksumResponse
	if err := json.NewDecoder(resp.Body).Decode(&checksumResp); err != nil {
		return "", api.WrapAPIError(api.ErrUnknown, "failed to parse response", err)
	}

	if !checksumResp.IsSuccess() {
		return "", &api.APIError{
			Code:    checksumResp.GetErrorCode(),
			Message: checksumResp.Message,
		}
	}

	return checksumResp.Checksum, nil
}
