package filestation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// setupUploadTestClient creates a test client with HTTP test server for upload tests
func setupUploadTestClient(t *testing.T) (*api.Client, *httptest.Server) {
	t.Helper()

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		// Handle upload requests - check for upload function in query
		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"uploaded": []map[string]interface{}{
						{
							"path":     "/home/test_upload.txt",
							"name":     "test_upload.txt",
							"size":     int64(100),
							"checksum": "abc123",
						},
					},
				},
			})
			return
		}

		// Default response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		testServer.Close()
		t.Fatalf("NewClient() error = %v", err)
	}

	return client, testServer
}

// TestUploadFile_Success tests successful file upload
func TestUploadFile_Success(t *testing.T) {
	client, testServer := setupUploadTestClient(t)
	defer testServer.Close()

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	// Create a temporary file to upload
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_upload.txt")
	testContent := []byte("Hello, World!")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	resp, err := fs.UploadFile(ctx, testFile, "/home", nil)

	if err != nil {
		t.Errorf("UploadFile() error = %v", err)
	}

	if resp == nil {
		t.Fatal("UploadFile() returned nil response")
	}

	if !resp.IsSuccess() {
		t.Errorf("UploadFile() response not successful, got success = %d", resp.Success)
	}

	if len(resp.Data.Uploaded) != 1 {
		t.Errorf("UploadFile() uploaded count = %d, want 1", len(resp.Data.Uploaded))
	}

	if len(resp.Data.Uploaded) > 0 {
		uploaded := resp.Data.Uploaded[0]
		if uploaded.Name != "test_upload.txt" {
			t.Errorf("UploadFile() uploaded name = %s, want test_upload.txt", uploaded.Name)
		}
	}
}

// TestUploadFile_FileNotExists tests upload when local file doesn't exist
func TestUploadFile_FileNotExists(t *testing.T) {
	client, testServer := setupUploadTestClient(t)
	defer testServer.Close()

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	nonExistentFile := filepath.Join(t.TempDir(), "nonexistent.txt")

	_, err := fs.UploadFile(ctx, nonExistentFile, "/home", nil)

	if err == nil {
		t.Error("UploadFile() expected error for non-existent file, got nil")
	}

	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		if apiErr.Code != api.ErrUnknown {
			t.Errorf("UploadFile() error code = %d, want ErrUnknown", apiErr.Code)
		}
	} else {
		t.Errorf("UploadFile() error type = %T, want *api.APIError", err)
	}
}

// TestUploadFile_NotAuthenticated tests upload when not authenticated
func TestUploadFile_NotAuthenticated(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	// Don't login - simulating not authenticated

	ctx := context.Background()
	fs := NewFileStationService(client)

	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = fs.UploadFile(ctx, testFile, "/home", nil)

	if err == nil {
		t.Error("UploadFile() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("UploadFile() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("UploadFile() error = %v, want auth error", apiErr)
	}
}

// TestUploadFile_WithOverwrite tests upload with overwrite option
func TestUploadFile_WithOverwrite(t *testing.T) {
	client, testServer := setupUploadTestClient(t)
	defer testServer.Close()

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("Overwritten content")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	options := &UploadOptions{
		Overwrite: true,
	}

	resp, err := fs.UploadFile(ctx, testFile, "/home", options)

	if err != nil {
		t.Errorf("UploadFile() with overwrite error = %v", err)
	}

	if resp == nil || !resp.IsSuccess() {
		t.Error("UploadFile() with overwrite failed")
	}
}

// TestUploadReader_Success tests successful reader upload
func TestUploadReader_Success(t *testing.T) {
	client, testServer := setupUploadTestClient(t)
	defer testServer.Close()

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	testContent := []byte("Reader content test")
	reader := bytes.NewReader(testContent)

	resp, err := fs.UploadReader(ctx, reader, "/home", "reader_test.txt", int64(len(testContent)), nil)

	if err != nil {
		t.Errorf("UploadReader() error = %v", err)
	}

	if resp == nil {
		t.Fatal("UploadReader() returned nil response")
	}

	if !resp.IsSuccess() {
		t.Errorf("UploadReader() response not successful")
	}

	if len(resp.Data.Uploaded) != 1 {
		t.Errorf("UploadReader() uploaded count = %d, want 1", len(resp.Data.Uploaded))
	}
}

// TestUploadReader_LargeFile tests upload of larger file content
func TestUploadReader_LargeFile(t *testing.T) {
	client, testServer := setupUploadTestClient(t)
	defer testServer.Close()

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	// Create a large content (1MB)
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	reader := bytes.NewReader(largeContent)

	resp, err := fs.UploadReader(ctx, reader, "/home", "large_file.bin", int64(len(largeContent)), nil)

	if err != nil {
		t.Errorf("UploadReader() large file error = %v", err)
	}

	if resp == nil || !resp.IsSuccess() {
		t.Error("UploadReader() large file failed")
	}

	if len(resp.Data.Uploaded) != 1 {
		t.Errorf("UploadReader() large file uploaded count = %d, want 1", len(resp.Data.Uploaded))
	}
}

// TestUploadReader_NotAuthenticated tests reader upload when not authenticated
func TestUploadReader_NotAuthenticated(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	fs := NewFileStationService(client)

	reader := bytes.NewReader([]byte("test"))

	_, err = fs.UploadReader(ctx, reader, "/home", "test.txt", 4, nil)

	if err == nil {
		t.Error("UploadReader() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("UploadReader() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("UploadReader() error = %v, want auth error", apiErr)
	}
}

// TestStartChunkedUpload_Success tests starting a chunked upload session
func TestStartChunkedUpload_Success(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		// Handle chunked upload start
		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") &&
			r.URL.Query().Get("func") == "start_chunked_upload" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"upload_id": "upload-session-12345",
				},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	expectedUploadID := "upload-session-12345"
	uploadID, err := fs.StartChunkedUpload(ctx, "/home/uploads")

	if err != nil {
		t.Errorf("StartChunkedUpload() error = %v", err)
	}

	if uploadID != expectedUploadID {
		t.Errorf("StartChunkedUpload() uploadID = %s, want %s", uploadID, expectedUploadID)
	}
}

// TestStartChunkedUpload_NotAuthenticated tests chunked upload start when not authenticated
func TestStartChunkedUpload_NotAuthenticated(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	fs := NewFileStationService(client)

	_, err = fs.StartChunkedUpload(ctx, "/home/uploads")

	if err == nil {
		t.Error("StartChunkedUpload() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("StartChunkedUpload() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("StartChunkedUpload() error = %v, want auth error", apiErr)
	}
}

// TestStartChunkedUpload_NoUploadID tests handling when server doesn't return upload_id
func TestStartChunkedUpload_NoUploadID(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") &&
			r.URL.Query().Get("func") == "start_chunked_upload" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"data":    map[string]interface{}{}, // Missing upload_id
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	_, err = fs.StartChunkedUpload(ctx, "/home/uploads")

	if err == nil {
		t.Error("StartChunkedUpload() expected error when no upload_id returned, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("StartChunkedUpload() error type = %T, want *api.APIError", err)
	}
}

// TestStartChunkedUpload_APIError tests handling of API error response
func TestStartChunkedUpload_APIError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") &&
			r.URL.Query().Get("func") == "start_chunked_upload" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    0,
				"error_code": 2004,
				"error_msg":  "Invalid path",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	_, err = fs.StartChunkedUpload(ctx, "/invalid/path")

	if err == nil {
		t.Error("StartChunkedUpload() expected error for invalid path, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("StartChunkedUpload() error type = %T, want *api.APIError", err)
	} else if apiErr.Code != api.ErrInvalidPath {
		t.Errorf("StartChunkedUpload() error code = %d, want ErrInvalidPath", apiErr.Code)
	}
}

// TestChunkedUpload_Success tests successful chunk upload
func TestChunkedUpload_Success(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") &&
			r.URL.Query().Get("func") == "chunked_upload" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	uploadID := "upload-session-12345"
	chunkData := []byte("This is chunk data")
	offset := int64(0)

	err = fs.ChunkedUpload(ctx, uploadID, offset, chunkData)

	if err != nil {
		t.Errorf("ChunkedUpload() error = %v", err)
	}
}

// TestChunkedUpload_WithOffset tests chunk upload with offset
func TestChunkedUpload_WithOffset(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") &&
			r.URL.Query().Get("func") == "chunked_upload" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	uploadID := "upload-session-12345"
	chunkData := []byte("Second chunk")
	offset := int64(1024 * 1024) // 1MB offset

	err = fs.ChunkedUpload(ctx, uploadID, offset, chunkData)

	if err != nil {
		t.Errorf("ChunkedUpload() with offset error = %v", err)
	}
}

// TestChunkedUpload_NotAuthenticated tests chunk upload when not authenticated
func TestChunkedUpload_NotAuthenticated(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	fs := NewFileStationService(client)

	err = fs.ChunkedUpload(ctx, "upload-id", 0, []byte("data"))

	if err == nil {
		t.Error("ChunkedUpload() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("ChunkedUpload() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("ChunkedUpload() error = %v, want auth error", apiErr)
	}
}

// TestChunkedUpload_APIError tests handling of API error during chunk upload
func TestChunkedUpload_APIError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") &&
			r.URL.Query().Get("func") == "chunked_upload" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    0,
				"error_code": 2001,
				"error_msg":  "Upload session not found",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	uploadID := "invalid-upload-id"

	err = fs.ChunkedUpload(ctx, uploadID, 0, []byte("data"))

	if err == nil {
		t.Error("ChunkedUpload() expected error for invalid upload ID, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("ChunkedUpload() error type = %T, want *api.APIError", err)
	}
}

// TestChunkedUpload_EmptyData tests chunk upload with empty data
func TestChunkedUpload_EmptyData(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") &&
			r.URL.Query().Get("func") == "chunked_upload" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	err = fs.ChunkedUpload(ctx, "upload-id", 0, []byte{})

	if err != nil {
		t.Errorf("ChunkedUpload() with empty data error = %v", err)
	}
}

// TestGetChunkedUpload_Success tests getting chunked upload status
func TestGetChunkedUpload_Success(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") &&
			r.URL.Query().Get("func") == "get_chunked_upload" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"upload_id":  "upload-session-12345",
					"status":     "uploading",
					"offset":     1024,
					"size":       1024000,
					"file_path":  "/home/uploads",
					"file_name":  "large_file.bin",
				},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	uploadID := "upload-session-12345"

	status, err := fs.GetChunkedUpload(ctx, uploadID)

	if err != nil {
		t.Errorf("GetChunkedUpload() error = %v", err)
	}

	if status == nil {
		t.Fatal("GetChunkedUpload() returned nil status")
	}

	if !status.IsSuccess() {
		t.Errorf("GetChunkedUpload() response not successful")
	}

	if status.Data.UploadID != uploadID {
		t.Errorf("GetChunkedUpload() uploadID = %s, want %s", status.Data.UploadID, uploadID)
	}

	if status.Data.Status != "uploading" {
		t.Errorf("GetChunkedUpload() status = %s, want uploading", status.Data.Status)
	}

	if status.Data.Offset != 1024 {
		t.Errorf("GetChunkedUpload() offset = %d, want 1024", status.Data.Offset)
	}

	if status.Data.Size != 1024000 {
		t.Errorf("GetChunkedUpload() size = %d, want 1024000", status.Data.Size)
	}
}

// TestGetChunkedUpload_NotAuthenticated tests getting status when not authenticated
func TestGetChunkedUpload_NotAuthenticated(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	fs := NewFileStationService(client)

	_, err = fs.GetChunkedUpload(ctx, "upload-id")

	if err == nil {
		t.Error("GetChunkedUpload() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetChunkedUpload() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("GetChunkedUpload() error = %v, want auth error", apiErr)
	}
}

// TestGetChunkedUpload_NotFound tests getting status of non-existent upload
func TestGetChunkedUpload_NotFound(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") &&
			r.URL.Query().Get("func") == "get_chunked_upload" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    0,
				"error_code": 2001,
				"error_msg":  "Upload session not found",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	_, err = fs.GetChunkedUpload(ctx, "non-existent-upload")

	if err == nil {
		t.Error("GetChunkedUpload() expected error for non-existent upload, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetChunkedUpload() error type = %T, want *api.APIError", err)
	}
}

// TestDeleteChunkedUploadFile_Success tests deleting a chunked upload
func TestDeleteChunkedUploadFile_Success(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") &&
			r.URL.Query().Get("func") == "delete_chunked_upload_file" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	uploadID := "upload-session-12345"

	err = fs.DeleteChunkedUploadFile(ctx, uploadID)

	if err != nil {
		t.Errorf("DeleteChunkedUploadFile() error = %v", err)
	}
}

// TestDeleteChunkedUploadFile_NotAuthenticated tests deletion when not authenticated
func TestDeleteChunkedUploadFile_NotAuthenticated(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	fs := NewFileStationService(client)

	err = fs.DeleteChunkedUploadFile(ctx, "upload-id")

	if err == nil {
		t.Error("DeleteChunkedUploadFile() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("DeleteChunkedUploadFile() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("DeleteChunkedUploadFile() error = %v, want auth error", apiErr)
	}
}

// TestDeleteChunkedUploadFile_NotFound tests deleting non-existent upload
func TestDeleteChunkedUploadFile_NotFound(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") &&
			r.URL.Query().Get("func") == "delete_chunked_upload_file" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    0,
				"error_code": 2001,
				"error_msg":  "Upload session not found",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	err = fs.DeleteChunkedUploadFile(ctx, "non-existent-upload")

	if err == nil {
		t.Error("DeleteChunkedUploadFile() expected error for non-existent upload, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("DeleteChunkedUploadFile() error type = %T, want *api.APIError", err)
	}
}

// TestUploadFile_APIErrorResponse tests upload with API error response
func TestUploadFile_APIErrorResponse(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    0,
				"error_code": 2006,
				"error_msg":  "Quota exceeded",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = fs.UploadFile(ctx, testFile, "/home", nil)

	if err == nil {
		t.Error("UploadFile() expected error for quota exceeded, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("UploadFile() error type = %T, want *api.APIError", err)
	} else if apiErr.Code != api.ErrQuotaExceeded {
		t.Errorf("UploadFile() error code = %d, want ErrQuotaExceeded", apiErr.Code)
	}
}

// TestUploadResponse_IsSuccess tests UploadResponse.IsSuccess method
func TestUploadResponse_IsSuccess(t *testing.T) {
	testCases := []struct {
		name     string
		success  int
		expected bool
	}{
		{"success", 1, true},
		{"failure", 0, false},
		{"invalid", 2, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &UploadResponse{}
			resp.Success = api.IntBool(tc.success)

			result := resp.IsSuccess()

			if result != tc.expected {
				t.Errorf("IsSuccess() = %v, want %v", result, tc.expected)
			}
		})
	}
}

// TestUploadResponse_GetErrorCode tests UploadResponse.GetErrorCode method
func TestUploadResponse_GetErrorCode(t *testing.T) {
	resp := &UploadResponse{
		BaseResponse: api.BaseResponse{
			Code: 2004,
		},
	}

	code := resp.GetErrorCode()

	if code != api.ErrInvalidPath {
		t.Errorf("GetErrorCode() = %d, want ErrInvalidPath", code)
	}
}

// TestUploadOptions_DefaultValues tests UploadOptions with default values
func TestUploadOptions_DefaultValues(t *testing.T) {
	options := &UploadOptions{}

	if options.Overwrite {
		t.Error("UploadOptions.Overwrite default = true, want false")
	}

	if options.Checksum != "" {
		t.Errorf("UploadOptions.Checksum default = %s, want empty", options.Checksum)
	}

	if options.Progress != nil {
		t.Error("UploadOptions.Progress default non-nil, want nil")
	}
}

// TestUploadOptions_WithValues tests UploadOptions with values set
func TestUploadOptions_WithValues(t *testing.T) {
	progress := make(chan UploadProgress, 1)
	defer close(progress)

	options := &UploadOptions{
		Overwrite: true,
		Checksum:  "abc123",
		Progress:  progress,
	}

	if !options.Overwrite {
		t.Error("UploadOptions.Overwrite not set to true")
	}

	if options.Checksum != "abc123" {
		t.Errorf("UploadOptions.Checksum = %s, want abc123", options.Checksum)
	}

	if options.Progress == nil {
		t.Error("UploadOptions.Progress is nil, want non-nil")
	}
}

// TestChunkedUpload_MultipleChunks tests uploading multiple chunks sequentially
func TestChunkedUpload_MultipleChunks(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") &&
			r.URL.Query().Get("func") == "chunked_upload" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	chunkSize := 1024
	numChunks := 5

	for i := 0; i < numChunks; i++ {
		chunkData := make([]byte, chunkSize)
		for j := range chunkData {
			chunkData[j] = byte(i + j)
		}
		offset := int64(i * chunkSize)

		err := fs.ChunkedUpload(ctx, "test-upload-id", offset, chunkData)
		if err != nil {
			t.Errorf("ChunkedUpload() chunk %d error = %v", i, err)
		}
	}
}

// TestChunkedUpload_LargeChunk tests uploading a large chunk
func TestChunkedUpload_LargeChunk(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") &&
			r.URL.Query().Get("func") == "chunked_upload" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	// Create a 5MB chunk
	largeChunk := make([]byte, 5*1024*1024)
	for i := range largeChunk {
		largeChunk[i] = byte(i % 256)
	}

	err = fs.ChunkedUpload(ctx, "test-upload-id", 0, largeChunk)

	if err != nil {
		t.Errorf("ChunkedUpload() large chunk error = %v", err)
	}
}

// TestGetChunkedUpload_CompletedStatus tests getting status of completed upload
func TestGetChunkedUpload_CompletedStatus(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}

		if strings.Contains(r.URL.Path, "/cgi-bin/filemanager/utilRequest.cgi") &&
			r.URL.Query().Get("func") == "get_chunked_upload" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"upload_id":  "completed-upload-123",
					"status":     "completed",
					"offset":     1024000,
					"size":       1024000,
					"file_path":  "/home/uploads",
					"file_name":  "completed_file.bin",
				},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	uploadID := "completed-upload-123"

	status, err := fs.GetChunkedUpload(ctx, uploadID)

	if err != nil {
		t.Errorf("GetChunkedUpload() error = %v", err)
	}

	if status.Data.Status != "completed" {
		t.Errorf("GetChunkedUpload() status = %s, want completed", status.Data.Status)
	}

	if status.Data.Offset != status.Data.Size {
		t.Errorf("GetChunkedUpload() completed upload offset %d != size %d", status.Data.Offset, status.Data.Size)
	}
}

// errorReadCloser is a reader that always returns an error
type errorReadCloser struct {
	err error
}

func (e *errorReadCloser) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func (e *errorReadCloser) Close() error {
	return e.err
}

// failingReader is a reader that fails after reading a certain number of bytes
type failingReader struct {
	bytesRead int
	failAt    int
}

func (f *failingReader) Read(p []byte) (n int, err error) {
	if f.bytesRead >= f.failAt {
		return 0, errors.New("read failure simulated")
	}

	bytesToRead := len(p)
	if f.bytesRead+bytesToRead > f.failAt {
		bytesToRead = f.failAt - f.bytesRead
	}

	for i := 0; i < bytesToRead; i++ {
		p[i] = byte(f.bytesRead + i)
	}
	f.bytesRead += bytesToRead
	return bytesToRead, nil
}

// TestUploadReader_ReadError tests upload when reader fails
func TestUploadReader_ReadError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
			"data":    map[string]interface{}{},
		})
	}))
	defer testServer.Close()

	url := testServer.URL
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	// Create a reader that will fail on read
	errorReader := &errorReadCloser{err: errors.New("read error")}

	_, err = fs.UploadReader(ctx, errorReader, "/home", "test.txt", 10, nil)

	if err == nil {
		t.Error("UploadReader() expected error from failing reader, got nil")
	}
}
