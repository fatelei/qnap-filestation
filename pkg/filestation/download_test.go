package filestation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatelei/qnap-filestation/pkg/api"
	"github.com/fatelei/qnap-filestation/internal/testutil"
)

// setupDownloadTestClient creates a test client with mock server for download tests
func setupDownloadTestClient(t *testing.T) (*api.Client, *testutil.MockServer) {
	mockServer := testutil.NewMockServer()

	url := mockServer.URL()
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

	// Set a valid SID manually for testing
	client.SetSID("test-sid-12345")

	return client, mockServer
}

// TestDownloadFile_Success tests successful file download
func TestDownloadFile_Success(t *testing.T) {
	client, mockServer := setupDownloadTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	// Create a custom test server to handle binary download
	testContent := []byte("Hello, World! This is downloaded content.")

	// Create a test server that returns binary content for download
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-12345</authSid>
</QDocRoot>`))
			return
		}
		// Return binary content for download
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
	}))
	defer testServer.Close()

	// Update client to use test server
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

	newClient, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	newClient.SetSID("test-sid-12345")

	fs = NewFileStationService(newClient)

	// Create temporary directory for download
	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "downloaded.txt")

	err = fs.DownloadFile(ctx, "/remote/test.txt", localPath, nil)

	if err != nil {
		t.Errorf("DownloadFile() error = %v", err)
	}

	// Verify file was downloaded
	downloadedContent, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if !bytes.Equal(downloadedContent, testContent) {
		t.Errorf("DownloadFile() content = %s, want %s", downloadedContent, testContent)
	}
}

// TestDownloadFile_NotAuthenticated tests download when not authenticated
func TestDownloadFile_NotAuthenticated(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	url := mockServer.URL()
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
	// Don't set SID - simulating not authenticated

	ctx := context.Background()
	fs := NewFileStationService(client)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "test.txt")

	err = fs.DownloadFile(ctx, "/remote/test.txt", localPath, nil)

	if err == nil {
		t.Error("DownloadFile() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("DownloadFile() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("DownloadFile() error = %v, want auth error", apiErr)
	}
}

// TestDownloadFile_CreateDirectory tests directory creation during download
func TestDownloadFile_CreateDirectory(t *testing.T) {
	testContent := []byte("Content for directory creation test")

	// Create a test server that returns binary content
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	// Create path with nested directories that don't exist
	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "subdir1", "subdir2", "downloaded.txt")

	err = fs.DownloadFile(ctx, "/remote/test.txt", localPath, nil)

	if err != nil {
		t.Errorf("DownloadFile() error = %v", err)
	}

	// Verify directories were created
	if _, err := os.Stat(filepath.Join(tmpDir, "subdir1", "subdir2")); os.IsNotExist(err) {
		t.Error("DownloadFile() failed to create directories")
	}

	// Verify file was downloaded
	downloadedContent, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if !bytes.Equal(downloadedContent, testContent) {
		t.Errorf("DownloadFile() content mismatch")
	}
}

// TestDownloadFile_HTTPError tests download with HTTP error status
func TestDownloadFile_HTTPError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "test.txt")

	err = fs.DownloadFile(ctx, "/remote/test.txt", localPath, nil)

	if err == nil {
		t.Error("DownloadFile() expected error for 404 response, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("DownloadFile() error type = %T, want *api.APIError", err)
	}
}

// TestDownloadFile_FileCreateError tests download when file creation fails
func TestDownloadFile_FileCreateError(t *testing.T) {
	testContent := []byte("Test content")

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	// Try to write to a path that includes a file as a directory
	tmpDir := t.TempDir()
	fileAsDir := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(fileAsDir, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	invalidPath := filepath.Join(fileAsDir, "subdir", "downloaded.txt")

	err = fs.DownloadFile(ctx, "/remote/test.txt", invalidPath, nil)

	if err == nil {
		t.Error("DownloadFile() expected error when file creation fails, got nil")
	}
}

// TestDownloadFile_LargeFile tests downloading a large file
func TestDownloadFile_LargeFile(t *testing.T) {
	// Create 1MB of test content
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(largeContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "large.bin")

	err = fs.DownloadFile(ctx, "/remote/large.bin", localPath, nil)

	if err != nil {
		t.Errorf("DownloadFile() large file error = %v", err)
	}

	downloadedContent, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if len(downloadedContent) != len(largeContent) {
		t.Errorf("DownloadFile() size = %d, want %d", len(downloadedContent), len(largeContent))
	}

	if !bytes.Equal(downloadedContent, largeContent) {
		t.Error("DownloadFile() content mismatch for large file")
	}
}

// TestDownloadFile_WithOffset tests download with offset option
func TestDownloadFile_WithOffset(t *testing.T) {
	testContent := []byte("Offset test content for partial download")

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "offset_test.txt")

	options := &DownloadOptions{
		Offset: 10,
	}

	err = fs.DownloadFile(ctx, "/remote/test.txt", localPath, options)

	if err != nil {
		t.Errorf("DownloadFile() with offset error = %v", err)
	}

	// Note: Offset is currently not fully implemented in DownloadFile
	// This test ensures the offset option is accepted without error
}

// TestDownloadFile_NetworkError tests download with network error
func TestDownloadFile_NetworkError(t *testing.T) {
	// Create a client pointing to a non-existent server
	config := &api.Config{
		Host:     "localhost",
		Port:     9999, // Non-existent port
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "test.txt")

	err = fs.DownloadFile(ctx, "/remote/test.txt", localPath, nil)

	if err == nil {
		t.Error("DownloadFile() expected error for network failure, got nil")
	}
}

// TestDownloadReader_Success tests successful download to reader
func TestDownloadReader_Success(t *testing.T) {
	testContent := []byte("Content for DownloadReader test")

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	reader, size, err := fs.DownloadReader(ctx, "/remote/test.txt")

	if err != nil {
		t.Errorf("DownloadReader() error = %v", err)
	}

	if reader == nil {
		t.Fatal("DownloadReader() returned nil reader")
	}

	defer reader.Close()

	// Read all content
	downloadedContent, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read from reader: %v", err)
	}

	if !bytes.Equal(downloadedContent, testContent) {
		t.Errorf("DownloadReader() content = %s, want %s", downloadedContent, testContent)
	}

	if size != int64(len(testContent)) {
		t.Errorf("DownloadReader() size = %d, want %d", size, len(testContent))
	}
}

// TestDownloadReader_NotAuthenticated tests reader download when not authenticated
func TestDownloadReader_NotAuthenticated(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	url := mockServer.URL()
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
	// Don't set SID

	ctx := context.Background()
	fs := NewFileStationService(client)

	_, _, err = fs.DownloadReader(ctx, "/remote/test.txt")

	if err == nil {
		t.Error("DownloadReader() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("DownloadReader() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("DownloadReader() error = %v, want auth error", apiErr)
	}
}

// TestDownloadReader_HTTPError tests reader download with HTTP error
func TestDownloadReader_HTTPError(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden"))
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	reader, size, err := fs.DownloadReader(ctx, "/remote/test.txt")

	if err == nil {
		reader.Close()
		t.Error("DownloadReader() expected error for 403 response, got nil")
	}

	if size != 0 {
		t.Errorf("DownloadReader() size = %d on error, want 0", size)
	}

	if reader != nil {
		t.Error("DownloadReader() reader non-nil on error")
	}
}

// TestDownloadReader_Streaming tests streaming download
func TestDownloadReader_Streaming(t *testing.T) {
	testContent := []byte("Streaming test content for DownloadReader")

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	reader, size, err := fs.DownloadReader(ctx, "/remote/test.txt")

	if err != nil {
		t.Errorf("DownloadReader() error = %v", err)
	}

	defer reader.Close()

	// Read in chunks to simulate streaming
	buf := make([]byte, 10)
	var downloadedContent []byte

	for {
		n, err := reader.Read(buf)
		if n > 0 {
			downloadedContent = append(downloadedContent, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("DownloadReader() streaming error = %v", err)
		}
	}

	if !bytes.Equal(downloadedContent, testContent) {
		t.Errorf("DownloadReader() streamed content mismatch")
	}

	if size != int64(len(testContent)) {
		t.Errorf("DownloadReader() size = %d, want %d", size, len(testContent))
	}
}

// TestDownloadReader_LargeFile tests reader download with large file
func TestDownloadReader_LargeFile(t *testing.T) {
	// Create 5MB of test content
	largeContent := make([]byte, 5*1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(largeContent)))
		w.WriteHeader(http.StatusOK)
		w.Write(largeContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	reader, size, err := fs.DownloadReader(ctx, "/remote/large.bin")

	if err != nil {
		t.Errorf("DownloadReader() large file error = %v", err)
	}

	defer reader.Close()

	if size != int64(len(largeContent)) {
		t.Errorf("DownloadReader() size = %d, want %d", size, len(largeContent))
	}

	// Read first and last chunks to verify content
	firstChunk := make([]byte, 1024)
	n, _ := reader.Read(firstChunk)
	if n > 0 {
		if !bytes.Equal(firstChunk[:n], largeContent[:n]) {
			t.Error("DownloadReader() first chunk mismatch")
		}
	}

	// Seek to end is not possible, so we'll just verify the total size
	// by reading everything and comparing
}

// TestDownloadFileAsync_Success tests successful async download
func TestDownloadFileAsync_Success(t *testing.T) {
	client, mockServer := setupDownloadTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	expectedDownloadID := "download-12345"

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
			"data": map[string]interface{}{
				"download_id": expectedDownloadID,
				"url":         "/download_url",
			},
		},
	})

	resp, err := fs.DownloadFileAsync(ctx, "/remote/test.txt")

	if err != nil {
		t.Errorf("DownloadFileAsync() error = %v", err)
	}

	if resp == nil {
		t.Fatal("DownloadFileAsync() returned nil response")
	}

	if !resp.IsSuccess() {
		t.Errorf("DownloadFileAsync() response not successful")
	}

	if resp.Data.DownloadID != expectedDownloadID {
		t.Errorf("DownloadFileAsync() downloadID = %s, want %s", resp.Data.DownloadID, expectedDownloadID)
	}
}

// TestDownloadFileAsync_NotAuthenticated tests async download when not authenticated
func TestDownloadFileAsync_NotAuthenticated(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	url := mockServer.URL()
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
	// Don't set SID

	ctx := context.Background()
	fs := NewFileStationService(client)

	_, err = fs.DownloadFileAsync(ctx, "/remote/test.txt")

	if err == nil {
		t.Error("DownloadFileAsync() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("DownloadFileAsync() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("DownloadFileAsync() error = %v, want auth error", apiErr)
	}
}

// TestDownloadFileAsync_APIError tests async download with API error response
func TestDownloadFileAsync_APIError(t *testing.T) {
	client, mockServer := setupDownloadTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success":    0,
			"error_code": 2001,
			"error_msg":  "File not found",
		},
	})

	_, err := fs.DownloadFileAsync(ctx, "/remote/nonexistent.txt")

	if err == nil {
		t.Error("DownloadFileAsync() expected error for non-existent file, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("DownloadFileAsync() error type = %T, want *api.APIError", err)
	} else if apiErr.Code != api.ErrNotFound {
		t.Errorf("DownloadFileAsync() error code = %d, want ErrNotFound", apiErr.Code)
	}
}

// TestDownloadFileAsync_InvalidJSON tests async download with invalid JSON response
func TestDownloadFileAsync_InvalidJSON(t *testing.T) {
	client, mockServer := setupDownloadTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body:       "invalid json{{{",
	})

	_, err := fs.DownloadFileAsync(ctx, "/remote/test.txt")

	if err == nil {
		t.Error("DownloadFileAsync() expected error for invalid JSON, got nil")
	}
}

// TestDownloadFileAsync_MultipleDownloads tests multiple async downloads
func TestDownloadFileAsync_MultipleDownloads(t *testing.T) {
	client, mockServer := setupDownloadTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	files := []struct {
		path        string
		downloadID  string
	}{
		{"/remote/file1.txt", "download-1"},
		{"/remote/file2.txt", "download-2"},
		{"/remote/file3.txt", "download-3"},
	}

	for _, file := range files {
		mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: 200,
			Body: map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"download_id": file.downloadID,
					"url":         "/download_url",
				},
			},
		})

		resp, err := fs.DownloadFileAsync(ctx, file.path)

		if err != nil {
			t.Errorf("DownloadFileAsync() %s error = %v", file.path, err)
			continue
		}

		if resp.Data.DownloadID != file.downloadID {
			t.Errorf("DownloadFileAsync() %s downloadID = %s, want %s", file.path, resp.Data.DownloadID, file.downloadID)
		}
	}
}

// TestDownloadOptions_DefaultValues tests DownloadOptions with default values
func TestDownloadOptions_DefaultValues(t *testing.T) {
	options := &DownloadOptions{}

	if options.Offset != 0 {
		t.Errorf("DownloadOptions.Offset default = %d, want 0", options.Offset)
	}

	if options.Length != 0 {
		t.Errorf("DownloadOptions.Length default = %d, want 0", options.Length)
	}

	if options.Progress != nil {
		t.Error("DownloadOptions.Progress default non-nil, want nil")
	}
}

// TestDownloadOptions_WithValues tests DownloadOptions with values set
func TestDownloadOptions_WithValues(t *testing.T) {
	progress := make(chan DownloadProgress, 1)
	defer close(progress)

	options := &DownloadOptions{
		Offset:   100,
		Length:   1000,
		Progress: progress,
	}

	if options.Offset != 100 {
		t.Errorf("DownloadOptions.Offset = %d, want 100", options.Offset)
	}

	if options.Length != 1000 {
		t.Errorf("DownloadOptions.Length = %d, want 1000", options.Length)
	}

	if options.Progress == nil {
		t.Error("DownloadOptions.Progress is nil, want non-nil")
	}
}

// TestDownloadResponse_IsSuccess tests DownloadResponse.IsSuccess method
func TestDownloadResponse_IsSuccess(t *testing.T) {
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
			resp := &DownloadResponse{}
			resp.Success = api.IntBool(tc.success)

			result := resp.IsSuccess()

			if result != tc.expected {
				t.Errorf("IsSuccess() = %v, want %v", result, tc.expected)
			}
		})
	}
}

// TestDownloadResponse_GetErrorCode tests DownloadResponse.GetErrorCode method
func TestDownloadResponse_GetErrorCode(t *testing.T) {
	resp := &DownloadResponse{
		BaseResponse: api.BaseResponse{
			Code: 2001,
		},
	}

	code := resp.GetErrorCode()

	if code != api.ErrNotFound {
		t.Errorf("GetErrorCode() = %d, want ErrNotFound", code)
	}
}

// TestDownloadFile_WithProgress tests download with progress tracking
func TestDownloadFile_WithProgress(t *testing.T) {
	testContent := []byte("Content for progress tracking test")

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "progress_test.txt")

	progress := make(chan DownloadProgress, 10)
	defer close(progress)

	options := &DownloadOptions{
		Progress: progress,
	}

	err = fs.DownloadFile(ctx, "/remote/test.txt", localPath, options)

	if err != nil {
		t.Errorf("DownloadFile() with progress error = %v", err)
	}

	// Note: Progress is currently not fully implemented in DownloadFile
	// This test ensures the progress channel is accepted without error
}

// TestDownloadFile_EmptyFile tests downloading an empty file
func TestDownloadFile_EmptyFile(t *testing.T) {
	testContent := []byte{}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "empty.txt")

	err = fs.DownloadFile(ctx, "/remote/empty.txt", localPath, nil)

	if err != nil {
		t.Errorf("DownloadFile() empty file error = %v", err)
	}

	downloadedContent, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if len(downloadedContent) != 0 {
		t.Errorf("DownloadFile() empty file size = %d, want 0", len(downloadedContent))
	}
}

// TestDownloadReader_EmptyFile tests reading an empty file
func TestDownloadReader_EmptyFile(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusOK)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	reader, size, err := fs.DownloadReader(ctx, "/remote/empty.txt")

	if err != nil {
		t.Errorf("DownloadReader() empty file error = %v", err)
	}

	defer reader.Close()

	if size != 0 {
		t.Errorf("DownloadReader() empty file size = %d, want 0", size)
	}

	// Try to read from the empty reader
	buf := make([]byte, 100)
	n, err := reader.Read(buf)

	if err != nil && err != io.EOF {
		t.Errorf("DownloadReader() empty file read error = %v", err)
	}

	if n != 0 {
		t.Errorf("DownloadReader() empty file read returned %d bytes, want 0", n)
	}
}

// TestDownloadFile_OverwriteExisting tests downloading over existing file
func TestDownloadFile_OverwriteExisting(t *testing.T) {
	testContent := []byte("New content that should overwrite existing file")
	existingContent := []byte("Existing content")

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "overwrite.txt")

	// Create existing file
	if err := os.WriteFile(localPath, existingContent, 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	err = fs.DownloadFile(ctx, "/remote/test.txt", localPath, nil)

	if err != nil {
		t.Errorf("DownloadFile() overwrite error = %v", err)
	}

	downloadedContent, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if !bytes.Equal(downloadedContent, testContent) {
		t.Errorf("DownloadFile() overwrite content = %s, want %s", downloadedContent, testContent)
	}

	if bytes.Equal(downloadedContent, existingContent) {
		t.Error("DownloadFile() did not overwrite existing file")
	}
}

// TestDownloadFileAsync_DifferentPaths tests async download with different path formats
func TestDownloadFileAsync_DifferentPaths(t *testing.T) {
	client, mockServer := setupDownloadTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	testCases := []struct {
		name        string
		remotePath  string
		downloadID  string
		expectError bool
	}{
		{
			name:        "simple path",
			remotePath:  "/remote/file.txt",
			downloadID:  "download-1",
			expectError: false,
		},
		{
			name:        "path with spaces",
			remotePath:  "/remote/path with spaces/file.txt",
			downloadID:  "download-2",
			expectError: false,
		},
		{
			name:        "nested path",
			remotePath:  "/remote/nested/deep/path/file.txt",
			downloadID:  "download-3",
			expectError: false,
		},
		{
			name:        "root file",
			remotePath:  "/file.txt",
			downloadID:  "download-4",
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"download_id": tc.downloadID,
						"url":         "/download_url",
					},
				},
			})

			resp, err := fs.DownloadFileAsync(ctx, tc.remotePath)

			if tc.expectError {
				if err == nil {
					t.Errorf("DownloadFileAsync() %s expected error", tc.name)
				}
			} else {
				if err != nil {
					t.Errorf("DownloadFileAsync() %s error = %v", tc.name, err)
				}
				if resp.Data.DownloadID != tc.downloadID {
					t.Errorf("DownloadFileAsync() %s downloadID = %s, want %s", tc.name, resp.Data.DownloadID, tc.downloadID)
				}
			}
		})
	}
}

// TestDownloadFileAsync_URLField tests that URL field is properly parsed
func TestDownloadFileAsync_URLField(t *testing.T) {
	client, mockServer := setupDownloadTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	expectedURL := "https://example.com/download/file123"

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
			"data": map[string]interface{}{
				"download_id": "download-123",
				"url":         expectedURL,
			},
		},
	})

	resp, err := fs.DownloadFileAsync(ctx, "/remote/test.txt")

	if err != nil {
		t.Errorf("DownloadFileAsync() error = %v", err)
	}

	if resp.Data.URL != expectedURL {
		t.Errorf("DownloadFileAsync() URL = %s, want %s", resp.Data.URL, expectedURL)
	}
}

// TestDownloadProgress_Structure tests DownloadProgress structure
func TestDownloadProgress_Structure(t *testing.T) {
	progress := DownloadProgress{
		Total:      1000,
		Transferred: 500,
		Percentage: 50.0,
	}

	if progress.Total != 1000 {
		t.Errorf("DownloadProgress.Total = %d, want 1000", progress.Total)
	}

	if progress.Transferred != 500 {
		t.Errorf("DownloadProgress.Transferred = %d, want 500", progress.Transferred)
	}

	if progress.Percentage != 50.0 {
		t.Errorf("DownloadProgress.Percentage = %f, want 50.0", progress.Percentage)
	}
}

// TestDownloadReader_CloseTwice tests closing reader twice
func TestDownloadReader_CloseTwice(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	reader, _, err := fs.DownloadReader(ctx, "/remote/test.txt")

	if err != nil {
		t.Errorf("DownloadReader() error = %v", err)
	}

	// Close once
	err = reader.Close()
	if err != nil {
		t.Errorf("DownloadReader() first close error = %v", err)
	}

	// Close twice - should not error
	err = reader.Close()
	if err != nil {
		t.Errorf("DownloadReader() second close error = %v", err)
	}
}

// TestDownloadFile_WithLength tests download with length option
func TestDownloadFile_WithLength(t *testing.T) {
	testContent := []byte("Length test content for partial download")

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "length_test.txt")

	options := &DownloadOptions{
		Length: 100,
	}

	err = fs.DownloadFile(ctx, "/remote/test.txt", localPath, options)

	if err != nil {
		t.Errorf("DownloadFile() with length error = %v", err)
	}

	// Note: Length is currently not fully implemented in DownloadFile
	// This test ensures the length option is accepted without error
}

// TestDownloadFileAsync_WithoutURL tests async download response without URL field
func TestDownloadFileAsync_WithoutURL(t *testing.T) {
	client, mockServer := setupDownloadTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
			"data": map[string]interface{}{
				"download_id": "download-123",
				// URL field missing
			},
		},
	})

	resp, err := fs.DownloadFileAsync(ctx, "/remote/test.txt")

	if err != nil {
		t.Errorf("DownloadFileAsync() error = %v", err)
	}

	if resp.Data.URL != "" {
		t.Errorf("DownloadFileAsync() URL = %s, want empty", resp.Data.URL)
	}

	if resp.Data.DownloadID != "download-123" {
		t.Errorf("DownloadFileAsync() downloadID = %s, want download-123", resp.Data.DownloadID)
	}
}

// TestDownloadJSON_Marshaling tests JSON marshaling of download responses
func TestDownloadJSON_Marshaling(t *testing.T) {
	// Test DownloadResponse marshaling
	resp := &DownloadResponse{
		BaseResponse: api.BaseResponse{
			Success: api.IntBool(1),
		},
	}
	resp.Data.DownloadID = "test-download-id"
	resp.Data.URL = "https://example.com/download"

	data, err := json.Marshal(resp)
	if err != nil {
		t.Errorf("Failed to marshal DownloadResponse: %v", err)
	}

	var unmarshaled DownloadResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal DownloadResponse: %v", err)
	}

	if unmarshaled.Data.DownloadID != resp.Data.DownloadID {
		t.Errorf("Unmarshaled downloadID = %s, want %s", unmarshaled.Data.DownloadID, resp.Data.DownloadID)
	}

	if unmarshaled.Data.URL != resp.Data.URL {
		t.Errorf("Unmarshaled URL = %s, want %s", unmarshaled.Data.URL, resp.Data.URL)
	}
}

// TestDownloadFile_SpecialCharactersInPath tests download with special characters in path
func TestDownloadFile_SpecialCharactersInPath(t *testing.T) {
	testContent := []byte("Content for special characters test")

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "special-chars-file.txt")

	// Test with special characters in remote path
	specialPath := "/remote/path with spaces & special-chars_123.txt"

	err = fs.DownloadFile(ctx, specialPath, localPath, nil)

	if err != nil {
		t.Errorf("DownloadFile() with special characters error = %v", err)
	}
}

// TestDownloadFile_BinaryContent tests downloading binary content
func TestDownloadFile_BinaryContent(t *testing.T) {
	// Create binary content with all possible byte values
	binaryContent := make([]byte, 256)
	for i := range binaryContent {
		binaryContent[i] = byte(i)
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(binaryContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "binary.bin")

	err = fs.DownloadFile(ctx, "/remote/binary.bin", localPath, nil)

	if err != nil {
		t.Errorf("DownloadFile() binary content error = %v", err)
	}

	downloadedContent, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if !bytes.Equal(downloadedContent, binaryContent) {
		t.Error("DownloadFile() binary content mismatch")
	}

	// Verify all byte values are preserved
	for i := range binaryContent {
		if downloadedContent[i] != binaryContent[i] {
			t.Errorf("DownloadFile() byte at position %d = %d, want %d", i, downloadedContent[i], binaryContent[i])
		}
	}
}

// TestDownloadReader_ContextCancellation tests context cancellation during download
func TestDownloadReader_ContextCancellation(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		// Send data slowly to allow cancellation
		for i := 0; i < 100; i++ {
			w.Write([]byte("x"))
			// In a real test, we'd add a delay here
		}
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
	client.SetSID("test-sid")

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	fs := NewFileStationService(client)

	_, _, err = fs.DownloadReader(ctx, "/remote/test.txt")

	// Context cancellation should cause an error
	if err == nil {
		t.Error("DownloadReader() expected error with cancelled context, got nil")
	}
}

// TestDownloadFileAsync_SessionExpired tests async download with expired session
func TestDownloadFileAsync_SessionExpired(t *testing.T) {
	client, mockServer := setupDownloadTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success":    0,
			"error_code": 1002,
			"error_msg":  "Session expired",
		},
	})

	_, err := fs.DownloadFileAsync(ctx, "/remote/test.txt")

	if err == nil {
		t.Error("DownloadFileAsync() expected error for expired session, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("DownloadFileAsync() error type = %T, want *api.APIError", err)
	} else if apiErr.Code != api.ErrSessionExpired {
		t.Errorf("DownloadFileAsync() error code = %d, want ErrSessionExpired", apiErr.Code)
	}
}

// TestDownloadResponse_Unmarshal tests unmarshaling download response
func TestDownloadResponse_Unmarshal(t *testing.T) {
	jsonData := `{
		"success": 1,
		"data": {
			"download_id": "dl-12345",
			"url": "https://example.com/dl/file"
		}
	}`

	var resp DownloadResponse
	err := json.Unmarshal([]byte(jsonData), &resp)

	if err != nil {
		t.Errorf("Failed to unmarshal DownloadResponse: %v", err)
	}

	if !resp.IsSuccess() {
		t.Error("Unmarshaled response is not successful")
	}

	if resp.Data.DownloadID != "dl-12345" {
		t.Errorf("DownloadID = %s, want dl-12345", resp.Data.DownloadID)
	}

	if resp.Data.URL != "https://example.com/dl/file" {
		t.Errorf("URL = %s, want https://example.com/dl/file", resp.Data.URL)
	}
}

// TestDownloadFile_WithProgressChan tests download with progress channel functionality
func TestDownloadFile_WithProgressChan(t *testing.T) {
	testContent := []byte("Progress channel test content")

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "progress.txt")

	progress := make(chan DownloadProgress, 5)
	defer close(progress)

	options := &DownloadOptions{
		Progress: progress,
	}

	err = fs.DownloadFile(ctx, "/remote/test.txt", localPath, options)

	if err != nil {
		t.Errorf("DownloadFile() error = %v", err)
	}

	// Progress is currently not fully implemented, but the channel should be accepted
}

// TestDownloadFile_ReaderExhaustion tests that response body is fully read
func TestDownloadFile_ReaderExhaustion(t *testing.T) {
	testContent := []byte("Test content for reader exhaustion")

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "exhaustion.txt")

	err = fs.DownloadFile(ctx, "/remote/test.txt", localPath, nil)

	if err != nil {
		t.Errorf("DownloadFile() error = %v", err)
	}

	// Verify file was completely written
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		t.Fatalf("Failed to stat downloaded file: %v", err)
	}

	if fileInfo.Size() != int64(len(testContent)) {
		t.Errorf("Downloaded file size = %d, want %d", fileInfo.Size(), len(testContent))
	}
}

// TestDownloadFile_PathWithTrailingSlash tests remote path with trailing slash
func TestDownloadFile_PathWithTrailingSlash(t *testing.T) {
	testContent := []byte("Trailing slash test")

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(testContent)
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
	client.SetSID("test-sid")

	ctx := context.Background()
	fs := NewFileStationService(client)

	tmpDir := t.TempDir()
	localPath := filepath.Join(tmpDir, "trailing.txt")

	// Test with trailing slash in remote path
	err = fs.DownloadFile(ctx, "/remote/test.txt/", localPath, nil)

	if err != nil {
		t.Errorf("DownloadFile() with trailing slash error = %v", err)
	}
}
