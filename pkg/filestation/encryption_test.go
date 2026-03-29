package filestation

import (
	"context"
	"net/http"
	"testing"

	"github.com/fatelei/qnap-filestation/pkg/api"
	"github.com/fatelei/qnap-filestation/internal/testutil"
)


// TestEncryptFile tests the EncryptFile function
func TestEncryptFile(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *EncryptOptions
		wantPID        string
		wantErr        bool
		expectedErr    api.ErrorCode
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "encrypt single file successfully with aes256",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "encrypt-pid-12345",
				},
			},
			options: &EncryptOptions{
				SourceFiles: []string{"/home/file1.txt"},
				Password:    "securepassword",
				Algorithm:   "aes256",
			},
			wantPID: "encrypt-pid-12345",
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "encrypt" {
					t.Errorf("Expected func=encrypt, got %s", fn)
				}
				if pwd := r.URL.Query().Get("password"); pwd != "securepassword" {
					t.Errorf("Expected password=securepassword, got %s", pwd)
				}
				if alg := r.URL.Query().Get("algorithm"); alg != "aes256" {
					t.Errorf("Expected algorithm=aes256, got %s", alg)
				}
				if file := r.URL.Query().Get("source_file[0]"); file != "/home/file1.txt" {
					t.Errorf("Expected source_file[0]=/home/file1.txt, got %s", file)
				}
			},
		},
		{
			name: "encrypt multiple files successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "encrypt-pid-67890",
				},
			},
			options: &EncryptOptions{
				SourceFiles: []string{"/home/file1.txt", "/home/file2.txt", "/home/file3.txt"},
				SourcePath:  "/home",
				Password:    "mypassword",
				Algorithm:   "aes256",
			},
			wantPID: "encrypt-pid-67890",
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if path := r.URL.Query().Get("source_path"); path != "/home" {
					t.Errorf("Expected source_path=/home, got %s", path)
				}
				if r.URL.Query().Get("source_file[0]") != "/home/file1.txt" {
					t.Error("Expected source_file[0]=/home/file1.txt")
				}
				if r.URL.Query().Get("source_file[1]") != "/home/file2.txt" {
					t.Error("Expected source_file[1]=/home/file2.txt")
				}
				if r.URL.Query().Get("source_file[2]") != "/home/file3.txt" {
					t.Error("Expected source_file[2]=/home/file3.txt")
				}
			},
		},
		{
			name: "encrypt with different algorithm (aes128)",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "encrypt-pid-aes128",
				},
			},
			options: &EncryptOptions{
				SourceFiles: []string{"/home/doc.pdf"},
				Password:    "password123",
				Algorithm:   "aes128",
			},
			wantPID: "encrypt-pid-aes128",
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if alg := r.URL.Query().Get("algorithm"); alg != "aes128" {
					t.Errorf("Expected algorithm=aes128, got %s", alg)
				}
			},
		},
		{
			name: "encrypt without algorithm specified",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "encrypt-pid-default",
				},
			},
			options: &EncryptOptions{
				SourceFiles: []string{"/home/file.txt"},
				Password:    "password",
			},
			wantPID: "encrypt-pid-default",
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if alg := r.URL.Query().Get("algorithm"); alg != "" {
					t.Errorf("Expected empty algorithm, got %s", alg)
				}
			},
		},
		{
			name: "encrypt without source path",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "encrypt-pid-nopath",
				},
			},
			options: &EncryptOptions{
				SourceFiles: []string{"/home/file.txt"},
				Password:    "password",
				Algorithm:   "aes256",
			},
			wantPID: "encrypt-pid-nopath",
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if path := r.URL.Query().Get("source_path"); path != "" {
					t.Errorf("Expected empty source_path, got %s", path)
				}
			},
		},
		{
			name:        "error when options is nil",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "error when source files is empty",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "encrypt-pid",
				},
			},
			options: &EncryptOptions{
				SourceFiles: []string{},
				Password:    "password",
			},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "error when password is empty",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "encrypt-pid",
				},
			},
			options: &EncryptOptions{
				SourceFiles: []string{"/home/file.txt"},
				Password:    "",
			},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "encrypt fails with API error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  0,
					"success": "false",
					"message": "Encryption failed",
				},
			},
			options: &EncryptOptions{
				SourceFiles: []string{"/home/file.txt"},
				Password:    "password",
			},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
			},
			options: &EncryptOptions{
				SourceFiles: []string{"/home/file.txt"},
				Password:    "password",
			},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			options: &EncryptOptions{
				SourceFiles: []string{"/home/file.txt"},
				Password:    "password",
			},
			wantErr: true,
		},
		{
			name: "encrypt with special characters in filename",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "encrypt-pid-special",
				},
			},
			options: &EncryptOptions{
				SourceFiles: []string{"/home/file with spaces.txt", "/home/file(1).txt"},
				Password:    "password",
				Algorithm:   "aes256",
			},
			wantPID: "encrypt-pid-special",
			wantErr: false,
		},
		{
			name: "encrypt many files",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "encrypt-pid-batch",
				},
			},
			options: &EncryptOptions{
				SourceFiles: []string{
					"/home/file1.txt",
					"/home/file2.txt",
					"/home/file3.txt",
					"/home/file4.txt",
					"/home/file5.txt",
					"/home/file6.txt",
					"/home/file7.txt",
					"/home/file8.txt",
					"/home/file9.txt",
					"/home/file10.txt",
				},
				Password:  "batchpassword",
				Algorithm: "aes256",
			},
			wantPID: "encrypt-pid-batch",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			pid, err := fs.EncryptFile(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("EncryptFile() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("EncryptFile() unexpected error = %v", err)
				return
			}

			if pid != tt.wantPID {
				t.Errorf("EncryptFile() returned PID = %s, want %s", pid, tt.wantPID)
			}

			if tt.assertRequest != nil {
				lastReq := mockServer.GetLastRequest()
				if lastReq != nil {
					tt.assertRequest(t, lastReq)
				}
			}
		})
	}
}

// TestEncryptFileAuthentication tests authentication scenarios for EncryptFile
func TestEncryptFileAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		options := &EncryptOptions{
			SourceFiles: []string{"/home/file.txt"},
			Password:    "password",
		}

		_, err := fs.EncryptFile(ctx, options)
		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestDecryptFile tests the DecryptFile function
func TestDecryptFile(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *DecryptOptions
		wantPID        string
		wantErr        bool
		expectedErr    api.ErrorCode
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "decrypt single file successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "decrypt-pid-12345",
				},
			},
			options: &DecryptOptions{
				SourceFiles: []string{"/home/file1.txt.enc"},
				Password:    "securepassword",
			},
			wantPID: "decrypt-pid-12345",
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "decrypt" {
					t.Errorf("Expected func=decrypt, got %s", fn)
				}
				if pwd := r.URL.Query().Get("password"); pwd != "securepassword" {
					t.Errorf("Expected password=securepassword, got %s", pwd)
				}
				if file := r.URL.Query().Get("source_file[0]"); file != "/home/file1.txt.enc" {
					t.Errorf("Expected source_file[0]=/home/file1.txt.enc, got %s", file)
				}
			},
		},
		{
			name: "decrypt multiple files successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "decrypt-pid-67890",
				},
			},
			options: &DecryptOptions{
				SourceFiles: []string{"/home/file1.txt.enc", "/home/file2.txt.enc"},
				SourcePath:  "/home",
				Password:    "mypassword",
			},
			wantPID: "decrypt-pid-67890",
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if path := r.URL.Query().Get("source_path"); path != "/home" {
					t.Errorf("Expected source_path=/home, got %s", path)
				}
				if r.URL.Query().Get("source_file[0]") != "/home/file1.txt.enc" {
					t.Error("Expected source_file[0]=/home/file1.txt.enc")
				}
				if r.URL.Query().Get("source_file[1]") != "/home/file2.txt.enc" {
					t.Error("Expected source_file[1]=/home/file2.txt.enc")
				}
			},
		},
		{
			name: "decrypt with source path",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "decrypt-pid-path",
				},
			},
			options: &DecryptOptions{
				SourceFiles: []string{"/archive/backup.enc"},
				SourcePath:  "/archive",
				Password:    "password123",
			},
			wantPID: "decrypt-pid-path",
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if path := r.URL.Query().Get("source_path"); path != "/archive" {
					t.Errorf("Expected source_path=/archive, got %s", path)
				}
			},
		},
		{
			name:        "error when options is nil",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "error when source files is empty",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "decrypt-pid",
				},
			},
			options: &DecryptOptions{
				SourceFiles: []string{},
				Password:    "password",
			},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "error when password is empty",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "decrypt-pid",
				},
			},
			options: &DecryptOptions{
				SourceFiles: []string{"/home/file.txt.enc"},
				Password:    "",
			},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "decrypt fails with API error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  0,
					"success": "false",
					"message": "Decryption failed - wrong password",
				},
			},
			options: &DecryptOptions{
				SourceFiles: []string{"/home/file.txt.enc"},
				Password:    "wrongpassword",
			},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
			},
			options: &DecryptOptions{
				SourceFiles: []string{"/home/file.txt.enc"},
				Password:    "password",
			},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			options: &DecryptOptions{
				SourceFiles: []string{"/home/file.txt.enc"},
				Password:    "password",
			},
			wantErr: true,
		},
		{
			name: "decrypt with special characters in filename",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "decrypt-pid-special",
				},
			},
			options: &DecryptOptions{
				SourceFiles: []string{"/home/encrypted file.txt.enc"},
				Password:    "password",
			},
			wantPID: "decrypt-pid-special",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			pid, err := fs.DecryptFile(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("DecryptFile() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("DecryptFile() unexpected error = %v", err)
				return
			}

			if pid != tt.wantPID {
				t.Errorf("DecryptFile() returned PID = %s, want %s", pid, tt.wantPID)
			}

			if tt.assertRequest != nil {
				lastReq := mockServer.GetLastRequest()
				if lastReq != nil {
					tt.assertRequest(t, lastReq)
				}
			}
		})
	}
}

// TestDecryptFileAuthentication tests authentication scenarios for DecryptFile
func TestDecryptFileAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		options := &DecryptOptions{
			SourceFiles: []string{"/home/file.txt.enc"},
			Password:    "password",
		}

		_, err := fs.DecryptFile(ctx, options)
		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestCipherFile tests the CipherFile function
func TestCipherFile(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *CipherOptions
		wantPID        string
		wantErr        bool
		expectedErr    api.ErrorCode
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "cipher encrypt action successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "cipher-encrypt-pid-12345",
				},
			},
			options: &CipherOptions{
				SourceFiles: []string{"/home/file1.txt"},
				Action:      "encrypt",
				Password:    "securepassword",
				Algorithm:   "aes256",
			},
			wantPID: "cipher-encrypt-pid-12345",
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "cipher" {
					t.Errorf("Expected func=cipher, got %s", fn)
				}
				if action := r.URL.Query().Get("action"); action != "encrypt" {
					t.Errorf("Expected action=encrypt, got %s", action)
				}
				if pwd := r.URL.Query().Get("password"); pwd != "securepassword" {
					t.Errorf("Expected password=securepassword, got %s", pwd)
				}
				if alg := r.URL.Query().Get("algorithm"); alg != "aes256" {
					t.Errorf("Expected algorithm=aes256, got %s", alg)
				}
			},
		},
		{
			name: "cipher decrypt action successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "cipher-decrypt-pid-12345",
				},
			},
			options: &CipherOptions{
				SourceFiles: []string{"/home/file1.txt.enc"},
				Action:      "decrypt",
				Password:    "securepassword",
				Algorithm:   "aes256",
			},
			wantPID: "cipher-decrypt-pid-12345",
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if action := r.URL.Query().Get("action"); action != "decrypt" {
					t.Errorf("Expected action=decrypt, got %s", action)
				}
			},
		},
		{
			name: "cipher with uppercase ENCRYPT action",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "cipher-upper-pid",
				},
			},
			options: &CipherOptions{
				SourceFiles: []string{"/home/file.txt"},
				Action:      "ENCRYPT",
				Password:    "password",
			},
			wantPID: "cipher-upper-pid",
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				// Should be lowercase
				if action := r.URL.Query().Get("action"); action != "encrypt" {
					t.Errorf("Expected action=encrypt (lowercase), got %s", action)
				}
			},
		},
		{
			name: "cipher with MixedCase action",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "cipher-mixed-pid",
				},
			},
			options: &CipherOptions{
				SourceFiles: []string{"/home/file.txt"},
				Action:      "DeCrYpT",
				Password:    "password",
			},
			wantPID: "cipher-mixed-pid",
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				// Should be lowercase
				if action := r.URL.Query().Get("action"); action != "decrypt" {
					t.Errorf("Expected action=decrypt (lowercase), got %s", action)
				}
			},
		},
		{
			name: "cipher multiple files",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "cipher-multi-pid",
				},
			},
			options: &CipherOptions{
				SourceFiles: []string{"/home/file1.txt", "/home/file2.txt"},
				SourcePath:  "/home",
				Action:      "encrypt",
				Password:    "password",
				Algorithm:   "aes256",
			},
			wantPID: "cipher-multi-pid",
			wantErr: false,
		},
		{
			name: "cipher without algorithm",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "cipher-noalg-pid",
				},
			},
			options: &CipherOptions{
				SourceFiles: []string{"/home/file.txt"},
				Action:      "encrypt",
				Password:    "password",
			},
			wantPID: "cipher-noalg-pid",
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if alg := r.URL.Query().Get("algorithm"); alg != "" {
					t.Errorf("Expected empty algorithm, got %s", alg)
				}
			},
		},
		{
			name:        "error when options is nil",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "error when source files is empty",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "cipher-pid",
				},
			},
			options: &CipherOptions{
				SourceFiles: []string{},
				Action:      "encrypt",
				Password:    "password",
			},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "error when password is empty",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "cipher-pid",
				},
			},
			options: &CipherOptions{
				SourceFiles: []string{"/home/file.txt"},
				Action:      "encrypt",
				Password:    "",
			},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "error when action is empty",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "cipher-pid",
				},
			},
			options: &CipherOptions{
				SourceFiles: []string{"/home/file.txt"},
				Action:      "",
				Password:    "password",
			},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "error when action is invalid",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
					"pid":     "cipher-pid",
				},
			},
			options: &CipherOptions{
				SourceFiles: []string{"/home/file.txt"},
				Action:      "invalid_action",
				Password:    "password",
			},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "cipher fails with API error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  0,
					"success": "false",
					"message": "Cipher operation failed",
				},
			},
			options: &CipherOptions{
				SourceFiles: []string{"/home/file.txt"},
				Action:      "encrypt",
				Password:    "password",
			},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
			},
			options: &CipherOptions{
				SourceFiles: []string{"/home/file.txt"},
				Action:      "encrypt",
				Password:    "password",
			},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			options: &CipherOptions{
				SourceFiles: []string{"/home/file.txt"},
				Action:      "encrypt",
				Password:    "password",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			pid, err := fs.CipherFile(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("CipherFile() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("CipherFile() unexpected error = %v", err)
				return
			}

			if pid != tt.wantPID {
				t.Errorf("CipherFile() returned PID = %s, want %s", pid, tt.wantPID)
			}

			if tt.assertRequest != nil {
				lastReq := mockServer.GetLastRequest()
				if lastReq != nil {
					tt.assertRequest(t, lastReq)
				}
			}
		})
	}
}

// TestCipherFileAuthentication tests authentication scenarios for CipherFile
func TestCipherFileAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		options := &CipherOptions{
			SourceFiles: []string{"/home/file.txt"},
			Action:      "encrypt",
			Password:    "password",
		}

		_, err := fs.CipherFile(ctx, options)
		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestChecksumFile tests the ChecksumFile function
func TestChecksumFile(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *ChecksumOptions
		wantChecksum   string
		wantErr        bool
		expectedErr    api.ErrorCode
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "calculate md5 checksum successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":   1,
					"success":  "true",
					"checksum": "5d41402abc4b2a76b9719d911017c592",
				},
			},
			options: &ChecksumOptions{
				SourceFile: "/home/file1.txt",
				Algorithm:  "md5",
			},
			wantChecksum: "5d41402abc4b2a76b9719d911017c592",
			wantErr:      false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "checksum" {
					t.Errorf("Expected func=checksum, got %s", fn)
				}
				if alg := r.URL.Query().Get("algorithm"); alg != "md5" {
					t.Errorf("Expected algorithm=md5, got %s", alg)
				}
				if file := r.URL.Query().Get("source_file"); file != "/home/file1.txt" {
					t.Errorf("Expected source_file=/home/file1.txt, got %s", file)
				}
			},
		},
		{
			name: "calculate sha1 checksum successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":   1,
					"success":  "true",
					"checksum": "356a192b7913b04c54574d18c28d46e6395428ab",
				},
			},
			options: &ChecksumOptions{
				SourceFile: "/home/file1.txt",
				Algorithm:  "sha1",
			},
			wantChecksum: "356a192b7913b04c54574d18c28d46e6395428ab",
			wantErr:      false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if alg := r.URL.Query().Get("algorithm"); alg != "sha1" {
					t.Errorf("Expected algorithm=sha1, got %s", alg)
				}
			},
		},
		{
			name: "calculate sha256 checksum successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":   1,
					"success":  "true",
					"checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				},
			},
			options: &ChecksumOptions{
				SourceFile: "/home/file1.txt",
				Algorithm:  "sha256",
			},
			wantChecksum: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			wantErr:      false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if alg := r.URL.Query().Get("algorithm"); alg != "sha256" {
					t.Errorf("Expected algorithm=sha256, got %s", alg)
				}
			},
		},
		{
			name: "calculate sha512 checksum successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":   1,
					"success":  "true",
					"checksum": "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
				},
			},
			options: &ChecksumOptions{
				SourceFile: "/home/file1.txt",
				Algorithm:  "sha512",
			},
			wantChecksum: "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
			wantErr:      false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if alg := r.URL.Query().Get("algorithm"); alg != "sha512" {
					t.Errorf("Expected algorithm=sha512, got %s", alg)
				}
			},
		},
		{
			name: "default to md5 when algorithm is empty",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":   1,
					"success":  "true",
					"checksum": "5d41402abc4b2a76b9719d911017c592",
				},
			},
			options: &ChecksumOptions{
				SourceFile: "/home/file1.txt",
				Algorithm:  "",
			},
			wantChecksum: "5d41402abc4b2a76b9719d911017c592",
			wantErr:      false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				// Should default to md5
				if alg := r.URL.Query().Get("algorithm"); alg != "md5" {
					t.Errorf("Expected algorithm=md5 (default), got %s", alg)
				}
			},
		},
		{
			name: "checksum with source path",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":   1,
					"success":  "true",
					"checksum": "5d41402abc4b2a76b9719d911017c592",
				},
			},
			options: &ChecksumOptions{
				SourceFile: "file1.txt",
				SourcePath: "/home",
				Algorithm:  "md5",
			},
			wantChecksum: "5d41402abc4b2a76b9719d911017c592",
			wantErr:      false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if path := r.URL.Query().Get("source_path"); path != "/home" {
					t.Errorf("Expected source_path=/home, got %s", path)
				}
			},
		},
		{
			name:        "error when options is nil",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "error when source file is empty",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":   1,
					"success":  "true",
					"checksum": "5d41402abc4b2a76b9719d911017c592",
				},
			},
			options: &ChecksumOptions{
				SourceFile: "",
				Algorithm:  "md5",
			},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "error when algorithm is invalid",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":   1,
					"success":  "true",
					"checksum": "5d41402abc4b2a76b9719d911017c592",
				},
			},
			options: &ChecksumOptions{
				SourceFile: "/home/file1.txt",
				Algorithm:  "invalid_algo",
			},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "error when algorithm is crc32 (not supported)",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":   1,
					"success":  "true",
					"checksum": "12345678",
				},
			},
			options: &ChecksumOptions{
				SourceFile: "/home/file1.txt",
				Algorithm:  "crc32",
			},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "checksum fails with API error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  0,
					"success": "false",
					"message": "File not found",
				},
			},
			options: &ChecksumOptions{
				SourceFile: "/home/nonexistent.txt",
				Algorithm:  "md5",
			},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
			},
			options: &ChecksumOptions{
				SourceFile: "/home/file1.txt",
				Algorithm:  "md5",
			},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			options: &ChecksumOptions{
				SourceFile: "/home/file1.txt",
				Algorithm:  "md5",
			},
			wantErr: true,
		},
		{
			name: "checksum with special characters in filename",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":   1,
					"success":  "true",
					"checksum": "5d41402abc4b2a76b9719d911017c592",
				},
			},
			options: &ChecksumOptions{
				SourceFile: "/home/file with spaces.txt",
				Algorithm:  "sha256",
			},
			wantChecksum: "5d41402abc4b2a76b9719d911017c592",
			wantErr:      false,
		},
		{
			name: "checksum returns empty checksum",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":   1,
					"success":  "true",
					"checksum": "",
				},
			},
			options: &ChecksumOptions{
				SourceFile: "/home/empty.txt",
				Algorithm:  "md5",
			},
			wantChecksum: "",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			checksum, err := fs.ChecksumFile(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("ChecksumFile() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ChecksumFile() unexpected error = %v", err)
				return
			}

			if checksum != tt.wantChecksum {
				t.Errorf("ChecksumFile() returned checksum = %s, want %s", checksum, tt.wantChecksum)
			}

			if tt.assertRequest != nil {
				lastReq := mockServer.GetLastRequest()
				if lastReq != nil {
					tt.assertRequest(t, lastReq)
				}
			}
		})
	}
}

// TestChecksumFileAuthentication tests authentication scenarios for ChecksumFile
func TestChecksumFileAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		options := &ChecksumOptions{
			SourceFile: "/home/file.txt",
			Algorithm:  "md5",
		}

		_, err := fs.ChecksumFile(ctx, options)
		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestChecksumFileAllAlgorithms tests all supported checksum algorithms
func TestChecksumFileAllAlgorithms(t *testing.T) {
	algorithms := []string{"md5", "sha1", "sha256", "sha512"}

	for _, alg := range algorithms {
		t.Run("algorithm_"+alg, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":   1,
					"success":  "true",
					"checksum": "test-checksum-" + alg,
				},
			})

			ctx := context.Background()
			fs := NewFileStationService(client)

			options := &ChecksumOptions{
				SourceFile: "/home/test.txt",
				Algorithm:  alg,
			}

			checksum, err := fs.ChecksumFile(ctx, options)
			if err != nil {
				t.Errorf("ChecksumFile() with algorithm %s unexpected error = %v", alg, err)
				return
			}

			if checksum != "test-checksum-"+alg {
				t.Errorf("ChecksumFile() with algorithm %s returned checksum = %s, want test-checksum-%s", alg, checksum, alg)
			}

			lastReq := mockServer.GetLastRequest()
			if lastReq != nil {
				if reqAlg := lastReq.URL.Query().Get("algorithm"); reqAlg != alg {
					t.Errorf("Expected algorithm=%s, got %s", alg, reqAlg)
				}
			}
		})
	}
}

// TestEncryptionIntegration tests integration between encryption methods
func TestEncryptionIntegration(t *testing.T) {
	t.Run("encrypt then decrypt workflow", func(t *testing.T) {
		client, mockServer := setupTestClient(t)
		defer mockServer.Close()

		ctx := context.Background()
		fs := NewFileStationService(client)

		// Step 1: Encrypt file
		mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"status":  1,
				"success": "true",
				"pid":     "encrypt-pid-workflow",
			},
		})

		encryptPID, err := fs.EncryptFile(ctx, &EncryptOptions{
			SourceFiles: []string{"/home/file.txt"},
			Password:    "password",
			Algorithm:   "aes256",
		})
		if err != nil {
			t.Fatalf("EncryptFile() error = %v", err)
		}
		if encryptPID != "encrypt-pid-workflow" {
			t.Fatalf("EncryptFile() PID = %s, want encrypt-pid-workflow", encryptPID)
		}

		// Step 2: Decrypt file
		mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"status":  1,
				"success": "true",
				"pid":     "decrypt-pid-workflow",
			},
		})

		decryptPID, err := fs.DecryptFile(ctx, &DecryptOptions{
			SourceFiles: []string{"/home/file.txt.enc"},
			Password:    "password",
		})
		if err != nil {
			t.Fatalf("DecryptFile() error = %v", err)
		}
		if decryptPID != "decrypt-pid-workflow" {
			t.Fatalf("DecryptFile() PID = %s, want decrypt-pid-workflow", decryptPID)
		}

		// Step 3: Verify checksum
		mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"status":   1,
				"success":  "true",
				"checksum": "5d41402abc4b2a76b9719d911017c592",
			},
		})

		checksum, err := fs.ChecksumFile(ctx, &ChecksumOptions{
			SourceFile: "/home/file.txt",
			Algorithm:  "md5",
		})
		if err != nil {
			t.Fatalf("ChecksumFile() error = %v", err)
		}
		if checksum != "5d41402abc4b2a76b9719d911017c592" {
			t.Fatalf("ChecksumFile() checksum = %s, want 5d41402abc4b2a76b9719d911017c592", checksum)
		}
	})
}

// BenchmarkEncryptFile benchmarks the EncryptFile function
func BenchmarkEncryptFile(b *testing.B) {
	client, mockServer := setupTestClient(&testing.T{})
	defer mockServer.Close()

	mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: http.StatusOK,
		Body: map[string]interface{}{
			"status":  1,
			"success": "true",
			"pid":     "benchmark-pid",
		},
	})

	ctx := context.Background()
	fs := NewFileStationService(client)
	options := &EncryptOptions{
		SourceFiles: []string{"/home/file.txt"},
		Password:    "password",
		Algorithm:   "aes256",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = fs.EncryptFile(ctx, options)
	}
}

// BenchmarkChecksumFile benchmarks the ChecksumFile function
func BenchmarkChecksumFile(b *testing.B) {
	client, mockServer := setupTestClient(&testing.T{})
	defer mockServer.Close()

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: http.StatusOK,
		Body: map[string]interface{}{
			"status":   1,
			"success":  "true",
			"checksum": "5d41402abc4b2a76b9719d911017c592",
		},
	})

	ctx := context.Background()
	fs := NewFileStationService(client)
	options := &ChecksumOptions{
		SourceFile: "/home/file.txt",
		Algorithm:  "md5",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = fs.ChecksumFile(ctx, options)
	}
}
