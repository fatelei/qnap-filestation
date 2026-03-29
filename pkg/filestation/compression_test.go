package filestation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// setupCompressionTestClient creates a test client with HTTP test server
func setupCompressionTestClient(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) (*api.Client, *httptest.Server) {
	t.Helper()

	defaultHandler := handler
	if defaultHandler == nil {
		defaultHandler = func(w http.ResponseWriter, r *http.Request) {
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
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"data":    map[string]interface{}{},
			})
		}
	}

	testServer := httptest.NewServer(http.HandlerFunc(defaultHandler))
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

// =============================================================================
// CompressFiles Tests (func=compress)
// =============================================================================

func TestCompressFiles_Success(t *testing.T) {
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
			r.Method == "POST" {
			// Verify request parameters
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			if r.FormValue("func") != "compress" {
				t.Errorf("Expected func=compress, got %s", r.FormValue("func"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"pid":     "compress-pid-12345",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": 1,
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

	options := &CompressOptions{
		SourceFiles:  []string{"file1.txt", "file2.txt"},
		SourcePath:   "/home/documents",
		CompressName: "archive.zip",
	}

	pid, err := fs.CompressFiles(ctx, options)

	if err != nil {
		t.Errorf("CompressFiles() error = %v", err)
	}

	if pid != "compress-pid-12345" {
		t.Errorf("CompressFiles() pid = %s, want compress-pid-12345", pid)
	}
}

func TestCompressFiles_MultipleFiles(t *testing.T) {
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
			r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			sourceFiles := r.FormValue("source_file")
			expectedFiles := "file1.txt,file2.txt,file3.txt,folder1"

			if sourceFiles != expectedFiles {
				t.Errorf("source_file = %s, want %s", sourceFiles, expectedFiles)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"pid":     "compress-multi-pid",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	options := &CompressOptions{
		SourceFiles:  []string{"file1.txt", "file2.txt", "file3.txt", "folder1"},
		SourcePath:   "/home",
		CompressName: "multi-archive.zip",
	}

	pid, err := fs.CompressFiles(ctx, options)

	if err != nil {
		t.Errorf("CompressFiles() error = %v", err)
	}

	if pid != "compress-multi-pid" {
		t.Errorf("CompressFiles() pid = %s, want compress-multi-pid", pid)
	}
}

func TestCompressFiles_WithCompressionLevel(t *testing.T) {
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
			r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			level := r.FormValue("level")
			if level != "9" {
				t.Errorf("level = %s, want 9", level)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"pid":     "compress-level-pid",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	options := &CompressOptions{
		SourceFiles:  []string{"large_file.dat"},
		SourcePath:   "/home/data",
		CompressName: "compressed.7z",
		Level:        9,
	}

	pid, err := fs.CompressFiles(ctx, options)

	if err != nil {
		t.Errorf("CompressFiles() error = %v", err)
	}

	if pid != "compress-level-pid" {
		t.Errorf("CompressFiles() pid = %s, want compress-level-pid", pid)
	}
}

func TestCompressFiles_NoCompressionLevel(t *testing.T) {
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
			r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			level := r.FormValue("level")
			if level != "" {
				t.Errorf("level should be empty when Level is 0, got %s", level)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"pid":     "compress-no-level-pid",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	options := &CompressOptions{
		SourceFiles:  []string{"file.txt"},
		SourcePath:   "/home",
		CompressName: "archive.zip",
		Level:        0,
	}

	pid, err := fs.CompressFiles(ctx, options)

	if err != nil {
		t.Errorf("CompressFiles() error = %v", err)
	}

	if pid != "compress-no-level-pid" {
		t.Errorf("CompressFiles() pid = %s, want compress-no-level-pid", pid)
	}
}

func TestCompressFiles_NotAuthenticated(t *testing.T) {
	client, testServer := setupCompressionTestClient(t, nil)
	defer testServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	options := &CompressOptions{
		SourceFiles:  []string{"file.txt"},
		SourcePath:   "/home",
		CompressName: "archive.zip",
	}

	_, err := fs.CompressFiles(ctx, options)

	if err == nil {
		t.Error("CompressFiles() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("CompressFiles() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("CompressFiles() error = %v, want auth error", apiErr)
	}
}

func TestCompressFiles_NilOptions(t *testing.T) {
	client, testServer := setupCompressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
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
	})
	defer testServer.Close()

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	_, err := fs.CompressFiles(ctx, nil)

	if err == nil {
		t.Error("CompressFiles() expected error for nil options, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("CompressFiles() error type = %T, want *api.APIError", err)
	} else if apiErr.Code != api.ErrInvalidParams {
		t.Errorf("CompressFiles() error code = %d, want ErrInvalidParams", apiErr.Code)
	}
}

func TestCompressFiles_APIError(t *testing.T) {
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
			r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    0,
				"error_code": 2004,
				"error_msg":  "Invalid source path",
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	options := &CompressOptions{
		SourceFiles:  []string{"nonexistent.txt"},
		SourcePath:   "/invalid/path",
		CompressName: "archive.zip",
	}

	_, err = fs.CompressFiles(ctx, options)

	if err == nil {
		t.Error("CompressFiles() expected error for invalid path, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("CompressFiles() error type = %T, want *api.APIError", err)
	} else if apiErr.Code != api.ErrInvalidPath {
		t.Errorf("CompressFiles() error code = %d, want ErrInvalidPath", apiErr.Code)
	}
}

// =============================================================================
// CancelCompress Tests (func=cancel_compress)
// =============================================================================

func TestCancelCompress_Success(t *testing.T) {
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
			r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			if r.FormValue("func") != "cancel_compress" {
				t.Errorf("Expected func=cancel_compress, got %s", r.FormValue("func"))
			}

			if r.FormValue("pid") != "compress-pid-12345" {
				t.Errorf("pid = %s, want compress-pid-12345", r.FormValue("pid"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	err = fs.CancelCompress(ctx, "compress-pid-12345")

	if err != nil {
		t.Errorf("CancelCompress() error = %v", err)
	}
}

func TestCancelCompress_InvalidPID(t *testing.T) {
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
			r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    0,
				"error_code": 2001,
				"error_msg":  "Process not found",
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	err = fs.CancelCompress(ctx, "invalid-pid")

	if err == nil {
		t.Error("CancelCompress() expected error for invalid PID, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("CancelCompress() error type = %T, want *api.APIError", err)
	}
}

func TestCancelCompress_EmptyPID(t *testing.T) {
	client, testServer := setupCompressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
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
	})
	defer testServer.Close()

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	err := fs.CancelCompress(ctx, "")

	if err == nil {
		t.Error("CancelCompress() expected error for empty PID, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("CancelCompress() error type = %T, want *api.APIError", err)
	} else if apiErr.Code != api.ErrInvalidParams {
		t.Errorf("CancelCompress() error code = %d, want ErrInvalidParams", apiErr.Code)
	}
}

func TestCancelCompress_NotAuthenticated(t *testing.T) {
	client, testServer := setupCompressionTestClient(t, nil)
	defer testServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	err := fs.CancelCompress(ctx, "some-pid")

	if err == nil {
		t.Error("CancelCompress() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("CancelCompress() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("CancelCompress() error = %v, want auth error", apiErr)
	}
}

// =============================================================================
// GetCompressStatus Tests (func=get_compress_status)
// =============================================================================

func TestGetCompressStatus_Running(t *testing.T) {
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
			r.Method == "GET" {
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			if r.FormValue("func") != "get_compress_status" {
				t.Errorf("Expected func=get_compress_status, got %s", r.FormValue("func"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"pid":         "compress-pid-12345",
					"status":      "running",
					"progress":    45.5,
					"source_path": "/home/documents",
					"dest_path":   "/home/archive.zip",
					"file_size":   1024000,
					"processed":   465920,
				},
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	status, err := fs.GetCompressStatus(ctx, "compress-pid-12345")

	if err != nil {
		t.Errorf("GetCompressStatus() error = %v", err)
	}

	if status == nil {
		t.Fatal("GetCompressStatus() returned nil status")
	}

	if status.PID != "compress-pid-12345" {
		t.Errorf("GetCompressStatus() PID = %s, want compress-pid-12345", status.PID)
	}

	if status.Status != "running" {
		t.Errorf("GetCompressStatus() status = %s, want running", status.Status)
	}

	if status.Progress != 45.5 {
		t.Errorf("GetCompressStatus() progress = %f, want 45.5", status.Progress)
	}

	if status.SourcePath != "/home/documents" {
		t.Errorf("GetCompressStatus() source_path = %s, want /home/documents", status.SourcePath)
	}

	if status.DestPath != "/home/archive.zip" {
		t.Errorf("GetCompressStatus() dest_path = %s, want /home/archive.zip", status.DestPath)
	}

	if status.FileSize != 1024000 {
		t.Errorf("GetCompressStatus() file_size = %d, want 1024000", status.FileSize)
	}

	if status.Processed != 465920 {
		t.Errorf("GetCompressStatus() processed = %d, want 465920", status.Processed)
	}
}

func TestGetCompressStatus_Finished(t *testing.T) {
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
			r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"pid":         "compress-pid-finished",
					"status":      "finished",
					"progress":    100.0,
					"source_path": "/home/photos",
					"dest_path":   "/home/photos.zip",
					"file_size":   5120000,
					"processed":   5120000,
				},
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	status, err := fs.GetCompressStatus(ctx, "compress-pid-finished")

	if err != nil {
		t.Errorf("GetCompressStatus() error = %v", err)
	}

	if status.Status != "finished" {
		t.Errorf("GetCompressStatus() status = %s, want finished", status.Status)
	}

	if status.Progress != 100.0 {
		t.Errorf("GetCompressStatus() progress = %f, want 100.0", status.Progress)
	}

	if status.Processed != status.FileSize {
		t.Errorf("GetCompressStatus() processed %d != file_size %d", status.Processed, status.FileSize)
	}
}

func TestGetCompressStatus_Failed(t *testing.T) {
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
			r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"pid":         "compress-pid-failed",
					"status":      "failed",
					"progress":    25.0,
					"source_path": "/home/data",
					"dest_path":   "/home/data.zip",
					"file_size":   2048000,
					"processed":   512000,
					"error":       "Disk full",
				},
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	status, err := fs.GetCompressStatus(ctx, "compress-pid-failed")

	if err != nil {
		t.Errorf("GetCompressStatus() error = %v", err)
	}

	if status.Status != "failed" {
		t.Errorf("GetCompressStatus() status = %s, want failed", status.Status)
	}

	if status.Error != "Disk full" {
		t.Errorf("GetCompressStatus() error = %s, want 'Disk full'", status.Error)
	}
}

func TestGetCompressStatus_EmptyPID(t *testing.T) {
	client, testServer := setupCompressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
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
	})
	defer testServer.Close()

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	_, err := fs.GetCompressStatus(ctx, "")

	if err == nil {
		t.Error("GetCompressStatus() expected error for empty PID, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetCompressStatus() error type = %T, want *api.APIError", err)
	} else if apiErr.Code != api.ErrInvalidParams {
		t.Errorf("GetCompressStatus() error code = %d, want ErrInvalidParams", apiErr.Code)
	}
}

func TestGetCompressStatus_NotAuthenticated(t *testing.T) {
	client, testServer := setupCompressionTestClient(t, nil)
	defer testServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	_, err := fs.GetCompressStatus(ctx, "some-pid")

	if err == nil {
		t.Error("GetCompressStatus() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetCompressStatus() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("GetCompressStatus() error = %v, want auth error", apiErr)
	}
}

// =============================================================================
// ExtractArchive Tests (func=extract)
// =============================================================================

func TestExtractArchive_Success(t *testing.T) {
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
			r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			if r.FormValue("func") != "extract" {
				t.Errorf("Expected func=extract, got %s", r.FormValue("func"))
			}

			if r.FormValue("extract_file") != "/home/archive.zip" {
				t.Errorf("extract_file = %s, want /home/archive.zip", r.FormValue("extract_file"))
			}

			if r.FormValue("path") != "/home/extracted" {
				t.Errorf("path = %s, want /home/extracted", r.FormValue("path"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"pid":     "extract-pid-12345",
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	options := &ExtractOptions{
		ExtractFile: "/home/archive.zip",
		DestPath:    "/home/extracted",
	}

	pid, err := fs.ExtractArchive(ctx, options)

	if err != nil {
		t.Errorf("ExtractArchive() error = %v", err)
	}

	if pid != "extract-pid-12345" {
		t.Errorf("ExtractArchive() pid = %s, want extract-pid-12345", pid)
	}
}

func TestExtractArchive_WithOverwrite(t *testing.T) {
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
			r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			overwrite := r.FormValue("overwrite")
			if overwrite != "true" {
				t.Errorf("overwrite = %s, want true", overwrite)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"pid":     "extract-overwrite-pid",
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	options := &ExtractOptions{
		ExtractFile: "/home/archive.zip",
		DestPath:    "/home/extracted",
		Overwrite:   true,
	}

	pid, err := fs.ExtractArchive(ctx, options)

	if err != nil {
		t.Errorf("ExtractArchive() error = %v", err)
	}

	if pid != "extract-overwrite-pid" {
		t.Errorf("ExtractArchive() pid = %s, want extract-overwrite-pid", pid)
	}
}

func TestExtractArchive_WithCodePage(t *testing.T) {
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
			r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			codePage := r.FormValue("code_page")
			if codePage != "cp936" {
				t.Errorf("code_page = %s, want cp936", codePage)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"pid":     "extract-codepage-pid",
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	options := &ExtractOptions{
		ExtractFile: "/home/chinese_archive.zip",
		DestPath:    "/home/extracted",
		CodePage:    "cp936",
	}

	pid, err := fs.ExtractArchive(ctx, options)

	if err != nil {
		t.Errorf("ExtractArchive() error = %v", err)
	}

	if pid != "extract-codepage-pid" {
		t.Errorf("ExtractArchive() pid = %s, want extract-codepage-pid", pid)
	}
}

func TestExtractArchive_WithAllOptions(t *testing.T) {
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
			r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			overwrite := r.FormValue("overwrite")
			if overwrite != "true" {
				t.Errorf("overwrite = %s, want true", overwrite)
			}

			codePage := r.FormValue("code_page")
			if codePage != "utf8" {
				t.Errorf("code_page = %s, want utf8", codePage)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"pid":     "extract-all-options-pid",
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	options := &ExtractOptions{
		ExtractFile: "/home/archive.zip",
		DestPath:    "/home/extracted",
		CodePage:    "utf8",
		Overwrite:   true,
	}

	pid, err := fs.ExtractArchive(ctx, options)

	if err != nil {
		t.Errorf("ExtractArchive() error = %v", err)
	}

	if pid != "extract-all-options-pid" {
		t.Errorf("ExtractArchive() pid = %s, want extract-all-options-pid", pid)
	}
}

func TestExtractArchive_WithoutOverwrite(t *testing.T) {
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
			r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			overwrite := r.FormValue("overwrite")
			if overwrite != "" {
				t.Errorf("overwrite should be empty when Overwrite is false, got %s", overwrite)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"pid":     "extract-no-overwrite-pid",
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	options := &ExtractOptions{
		ExtractFile: "/home/archive.zip",
		DestPath:    "/home/extracted",
		Overwrite:   false,
	}

	pid, err := fs.ExtractArchive(ctx, options)

	if err != nil {
		t.Errorf("ExtractArchive() error = %v", err)
	}

	if pid != "extract-no-overwrite-pid" {
		t.Errorf("ExtractArchive() pid = %s, want extract-no-overwrite-pid", pid)
	}
}

func TestExtractArchive_NilOptions(t *testing.T) {
	client, testServer := setupCompressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
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
	})
	defer testServer.Close()

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	_, err := fs.ExtractArchive(ctx, nil)

	if err == nil {
		t.Error("ExtractArchive() expected error for nil options, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("ExtractArchive() error type = %T, want *api.APIError", err)
	} else if apiErr.Code != api.ErrInvalidParams {
		t.Errorf("ExtractArchive() error code = %d, want ErrInvalidParams", apiErr.Code)
	}
}

func TestExtractArchive_NotAuthenticated(t *testing.T) {
	client, testServer := setupCompressionTestClient(t, nil)
	defer testServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	options := &ExtractOptions{
		ExtractFile: "/home/archive.zip",
		DestPath:    "/home/extracted",
	}

	_, err := fs.ExtractArchive(ctx, options)

	if err == nil {
		t.Error("ExtractArchive() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("ExtractArchive() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("ExtractArchive() error = %v, want auth error", apiErr)
	}
}

func TestExtractArchive_APIError(t *testing.T) {
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
			r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    0,
				"error_code": 2005,
				"error_msg":  "Archive file not found",
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	options := &ExtractOptions{
		ExtractFile: "/home/nonexistent.zip",
		DestPath:    "/home/extracted",
	}

	_, err = fs.ExtractArchive(ctx, options)

	if err == nil {
		t.Error("ExtractArchive() expected error for non-existent archive, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("ExtractArchive() error type = %T, want *api.APIError", err)
	}
}

// =============================================================================
// CancelExtract Tests (func=cancel_extract)
// =============================================================================

func TestCancelExtract_Success(t *testing.T) {
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
			r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			if r.FormValue("func") != "cancel_extract" {
				t.Errorf("Expected func=cancel_extract, got %s", r.FormValue("func"))
			}

			if r.FormValue("pid") != "extract-pid-12345" {
				t.Errorf("pid = %s, want extract-pid-12345", r.FormValue("pid"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	err = fs.CancelExtract(ctx, "extract-pid-12345")

	if err != nil {
		t.Errorf("CancelExtract() error = %v", err)
	}
}

func TestCancelExtract_InvalidPID(t *testing.T) {
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
			r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    0,
				"error_code": 2001,
				"error_msg":  "Extract process not found",
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	err = fs.CancelExtract(ctx, "invalid-extract-pid")

	if err == nil {
		t.Error("CancelExtract() expected error for invalid PID, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("CancelExtract() error type = %T, want *api.APIError", err)
	}
}

func TestCancelExtract_EmptyPID(t *testing.T) {
	client, testServer := setupCompressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
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
	})
	defer testServer.Close()

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	err := fs.CancelExtract(ctx, "")

	if err == nil {
		t.Error("CancelExtract() expected error for empty PID, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("CancelExtract() error type = %T, want *api.APIError", err)
	} else if apiErr.Code != api.ErrInvalidParams {
		t.Errorf("CancelExtract() error code = %d, want ErrInvalidParams", apiErr.Code)
	}
}

func TestCancelExtract_NotAuthenticated(t *testing.T) {
	client, testServer := setupCompressionTestClient(t, nil)
	defer testServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	err := fs.CancelExtract(ctx, "some-pid")

	if err == nil {
		t.Error("CancelExtract() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("CancelExtract() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("CancelExtract() error = %v, want auth error", apiErr)
	}
}

// =============================================================================
// GetExtractList Tests (func=get_extract_list)
// =============================================================================

func TestGetExtractList_Success(t *testing.T) {
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
			r.Method == "GET" {
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			if r.FormValue("func") != "get_extract_list" {
				t.Errorf("Expected func=get_extract_list, got %s", r.FormValue("func"))
			}

			if r.FormValue("extract_file") != "/home/archive.zip" {
				t.Errorf("extract_file = %s, want /home/archive.zip", r.FormValue("extract_file"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"datas": []map[string]interface{}{
					{
						"filename":  "file1.txt",
						"filesize":  int64(1024),
						"isfolder":  0,
					},
					{
						"filename":  "folder1",
						"filesize":  int64(0),
						"isfolder":  1,
					},
					{
						"filename":  "file2.dat",
						"filesize":  int64(2048),
						"isfolder":  0,
					},
				},
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	files, err := fs.GetExtractList(ctx, "/home/archive.zip")

	if err != nil {
		t.Errorf("GetExtractList() error = %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("GetExtractList() returned %d files, want 3", len(files))
	}

	if files[0].FileName != "file1.txt" {
		t.Errorf("GetExtractList() files[0].FileName = %s, want file1.txt", files[0].FileName)
	}

	if files[0].FileSize != 1024 {
		t.Errorf("GetExtractList() files[0].FileSize = %d, want 1024", files[0].FileSize)
	}

	if files[0].IsFolder != 0 {
		t.Errorf("GetExtractList() files[0].IsFolder = %d, want 0", files[0].IsFolder)
	}

	if files[1].FileName != "folder1" {
		t.Errorf("GetExtractList() files[1].FileName = %s, want folder1", files[1].FileName)
	}

	if files[1].IsFolder != 1 {
		t.Errorf("GetExtractList() files[1].IsFolder = %d, want 1", files[1].IsFolder)
	}

	if files[2].FileName != "file2.dat" {
		t.Errorf("GetExtractList() files[2].FileName = %s, want file2.dat", files[2].FileName)
	}

	if files[2].FileSize != 2048 {
		t.Errorf("GetExtractList() files[2].FileSize = %d, want 2048", files[2].FileSize)
	}
}

func TestGetExtractList_EmptyArchive(t *testing.T) {
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
			r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"datas":   []map[string]interface{}{},
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	files, err := fs.GetExtractList(ctx, "/home/empty.zip")

	if err != nil {
		t.Errorf("GetExtractList() error = %v", err)
	}

	if len(files) != 0 {
		t.Errorf("GetExtractList() returned %d files, want 0", len(files))
	}
}

func TestGetExtractList_LargeArchive(t *testing.T) {
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
			r.Method == "GET" {
			datas := make([]map[string]interface{}, 100)
			for i := 0; i < 100; i++ {
				datas[i] = map[string]interface{}{
					"filename":  fmt.Sprintf("file%d.txt", i),
					"filesize":  int64((i + 1) * 1024),
					"isfolder":  0,
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"datas":   datas,
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	files, err := fs.GetExtractList(ctx, "/home/large.zip")

	if err != nil {
		t.Errorf("GetExtractList() error = %v", err)
	}

	if len(files) != 100 {
		t.Errorf("GetExtractList() returned %d files, want 100", len(files))
	}

	if files[0].FileName != "file0.txt" {
		t.Errorf("GetExtractList() files[0].FileName = %s, want file0.txt", files[0].FileName)
	}

	if files[99].FileName != "file99.txt" {
		t.Errorf("GetExtractList() files[99].FileName = %s, want file99.txt", files[99].FileName)
	}
}

func TestGetExtractList_EmptyPath(t *testing.T) {
	client, testServer := setupCompressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
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
	})
	defer testServer.Close()

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	_, err := fs.GetExtractList(ctx, "")

	if err == nil {
		t.Error("GetExtractList() expected error for empty path, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetExtractList() error type = %T, want *api.APIError", err)
	} else if apiErr.Code != api.ErrInvalidParams {
		t.Errorf("GetExtractList() error code = %d, want ErrInvalidParams", apiErr.Code)
	}
}

func TestGetExtractList_NotAuthenticated(t *testing.T) {
	client, testServer := setupCompressionTestClient(t, nil)
	defer testServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	_, err := fs.GetExtractList(ctx, "/home/archive.zip")

	if err == nil {
		t.Error("GetExtractList() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetExtractList() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("GetExtractList() error = %v, want auth error", apiErr)
	}
}

func TestGetExtractList_APIError(t *testing.T) {
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
			r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    0,
				"error_code": 2005,
				"error_msg":  "Cannot open archive file",
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	_, err = fs.GetExtractList(ctx, "/home/corrupted.zip")

	if err == nil {
		t.Error("GetExtractList() expected error for corrupted archive, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetExtractList() error type = %T, want *api.APIError", err)
	}
}

// =============================================================================
// GetExtractStatus Tests (func=get_extract_status_ext)
// =============================================================================

func TestGetExtractStatus_Running(t *testing.T) {
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
			r.Method == "GET" {
			if err := r.ParseForm(); err != nil {
				t.Errorf("Failed to parse form: %v", err)
			}

			if r.FormValue("func") != "get_extract_status_ext" {
				t.Errorf("Expected func=get_extract_status_ext, got %s", r.FormValue("func"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"pid":          "extract-pid-12345",
					"status":       "running",
					"progress":     30.0,
					"extract_file": "/home/archive.zip",
					"dest_path":    "/home/extracted",
					"file_count":   10,
					"processed":    3,
				},
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	status, err := fs.GetExtractStatus(ctx, "extract-pid-12345")

	if err != nil {
		t.Errorf("GetExtractStatus() error = %v", err)
	}

	if status == nil {
		t.Fatal("GetExtractStatus() returned nil status")
	}

	if status.PID != "extract-pid-12345" {
		t.Errorf("GetExtractStatus() PID = %s, want extract-pid-12345", status.PID)
	}

	if status.Status != "running" {
		t.Errorf("GetExtractStatus() status = %s, want running", status.Status)
	}

	if status.Progress != 30.0 {
		t.Errorf("GetExtractStatus() progress = %f, want 30.0", status.Progress)
	}

	if status.ExtractFile != "/home/archive.zip" {
		t.Errorf("GetExtractStatus() extract_file = %s, want /home/archive.zip", status.ExtractFile)
	}

	if status.DestPath != "/home/extracted" {
		t.Errorf("GetExtractStatus() dest_path = %s, want /home/extracted", status.DestPath)
	}

	if status.FileCount != 10 {
		t.Errorf("GetExtractStatus() file_count = %d, want 10", status.FileCount)
	}

	if status.Processed != 3 {
		t.Errorf("GetExtractStatus() processed = %d, want 3", status.Processed)
	}
}

func TestGetExtractStatus_Finished(t *testing.T) {
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
			r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"pid":          "extract-pid-finished",
					"status":       "finished",
					"progress":     100.0,
					"extract_file": "/home/archive.zip",
					"dest_path":    "/home/extracted",
					"file_count":   50,
					"processed":    50,
				},
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	status, err := fs.GetExtractStatus(ctx, "extract-pid-finished")

	if err != nil {
		t.Errorf("GetExtractStatus() error = %v", err)
	}

	if status.Status != "finished" {
		t.Errorf("GetExtractStatus() status = %s, want finished", status.Status)
	}

	if status.Progress != 100.0 {
		t.Errorf("GetExtractStatus() progress = %f, want 100.0", status.Progress)
	}

	if status.Processed != status.FileCount {
		t.Errorf("GetExtractStatus() processed %d != file_count %d", status.Processed, status.FileCount)
	}
}

func TestGetExtractStatus_Failed(t *testing.T) {
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
			r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"pid":          "extract-pid-failed",
					"status":       "failed",
					"progress":     15.0,
					"extract_file": "/home/corrupted.zip",
					"dest_path":    "/home/extracted",
					"file_count":   100,
					"processed":    15,
					"error":        "Checksum mismatch",
				},
			})
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
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	status, err := fs.GetExtractStatus(ctx, "extract-pid-failed")

	if err != nil {
		t.Errorf("GetExtractStatus() error = %v", err)
	}

	if status.Status != "failed" {
		t.Errorf("GetExtractStatus() status = %s, want failed", status.Status)
	}

	if status.Error != "Checksum mismatch" {
		t.Errorf("GetExtractStatus() error = %s, want 'Checksum mismatch'", status.Error)
	}
}

func TestGetExtractStatus_ProgressTracking(t *testing.T) {
	progressValues := []float64{0.0, 25.0, 50.0, 75.0, 99.5, 100.0}
	processedValues := []int{0, 12, 25, 37, 49, 50}
	totalFiles := 50

	for i, progress := range progressValues {
		i := i
		progress := progress

		t.Run(fmt.Sprintf("Progress_%d", i), func(t *testing.T) {
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
					r.Method == "GET" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": 1,
						"data": map[string]interface{}{
							"pid":          fmt.Sprintf("extract-pid-progress-%d", i),
							"status":       "running",
							"progress":     progress,
							"extract_file": "/home/archive.zip",
							"dest_path":    "/home/extracted",
							"file_count":   totalFiles,
							"processed":    processedValues[i],
						},
					})
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
			if err := client.Login(ctx); err != nil {
				t.Fatalf("Login() error = %v", err)
			}

			fs := NewFileStationService(client)

			status, err := fs.GetExtractStatus(ctx, fmt.Sprintf("extract-pid-progress-%d", i))

			if err != nil {
				t.Errorf("GetExtractStatus() error = %v", err)
			}

			if status.Progress != progress {
				t.Errorf("GetExtractStatus() progress = %f, want %f", status.Progress, progress)
			}

			if status.Processed != processedValues[i] {
				t.Errorf("GetExtractStatus() processed = %d, want %d", status.Processed, processedValues[i])
			}
		})
	}
}

func TestGetExtractStatus_EmptyPID(t *testing.T) {
	client, testServer := setupCompressionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
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
	})
	defer testServer.Close()

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	fs := NewFileStationService(client)

	_, err := fs.GetExtractStatus(ctx, "")

	if err == nil {
		t.Error("GetExtractStatus() expected error for empty PID, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetExtractStatus() error type = %T, want *api.APIError", err)
	} else if apiErr.Code != api.ErrInvalidParams {
		t.Errorf("GetExtractStatus() error code = %d, want ErrInvalidParams", apiErr.Code)
	}
}

func TestGetExtractStatus_NotAuthenticated(t *testing.T) {
	client, testServer := setupCompressionTestClient(t, nil)
	defer testServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	_, err := fs.GetExtractStatus(ctx, "some-pid")

	if err == nil {
		t.Error("GetExtractStatus() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetExtractStatus() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("GetExtractStatus() error = %v, want auth error", apiErr)
	}
}

// =============================================================================
// Table-Driven Tests for Edge Cases
// =============================================================================

func TestCompressOptions_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		options     *CompressOptions
		expectError bool
		errorCode   api.ErrorCode
	}{
		{
			name: "nil options",
			options: &CompressOptions{
				SourceFiles:  nil,
				SourcePath:   "/home",
				CompressName: "archive.zip",
			},
			expectError: false,
		},
		{
			name: "empty source files",
			options: &CompressOptions{
				SourceFiles:  []string{},
				SourcePath:   "/home",
				CompressName: "archive.zip",
			},
			expectError: false,
		},
		{
			name: "single file",
			options: &CompressOptions{
				SourceFiles:  []string{"single.txt"},
				SourcePath:   "/home",
				CompressName: "single.zip",
			},
			expectError: false,
		},
		{
			name: "special characters in filename",
			options: &CompressOptions{
				SourceFiles:  []string{"file with spaces.txt", "file-with-dashes.txt"},
				SourcePath:   "/home",
				CompressName: "archive with spaces.zip",
			},
			expectError: false,
		},
		{
			name: "maximum compression level",
			options: &CompressOptions{
				SourceFiles:  []string{"large.dat"},
				SourcePath:   "/home",
				CompressName: "max.7z",
				Level:        9,
			},
			expectError: false,
		},
		{
			name: "minimum compression level",
			options: &CompressOptions{
				SourceFiles:  []string{"fast.txt"},
				SourcePath:   "/home",
				CompressName: "fast.zip",
				Level:        1,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
					r.Method == "POST" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": 1,
						"pid":     "test-pid",
					})
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
			if err := client.Login(ctx); err != nil {
				t.Fatalf("Login() error = %v", err)
			}

			fs := NewFileStationService(client)

			_, err = fs.CompressFiles(ctx, tc.options)

			if tc.expectError {
				if err == nil {
					t.Errorf("CompressFiles() expected error, got nil")
				}
				var apiErr *api.APIError
				if errors.As(err, &apiErr) && apiErr.Code != tc.errorCode {
					t.Errorf("CompressFiles() error code = %d, want %d", apiErr.Code, tc.errorCode)
				}
			} else if err != nil {
				t.Errorf("CompressFiles() unexpected error = %v", err)
			}
		})
	}
}

func TestExtractOptions_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		options     *ExtractOptions
		expectError bool
	}{
		{
			name: "with code page utf8",
			options: &ExtractOptions{
				ExtractFile: "/home/archive.zip",
				DestPath:    "/home",
				CodePage:    "utf8",
			},
			expectError: false,
		},
		{
			name: "with code page cp936",
			options: &ExtractOptions{
				ExtractFile: "/home/archive.zip",
				DestPath:    "/home",
				CodePage:    "cp936",
			},
			expectError: false,
		},
		{
			name: "with overwrite true",
			options: &ExtractOptions{
				ExtractFile: "/home/archive.zip",
				DestPath:    "/home",
				Overwrite:   true,
			},
			expectError: false,
		},
		{
			name: "with all options",
			options: &ExtractOptions{
				ExtractFile: "/home/archive.zip",
				DestPath:    "/home",
				CodePage:    "utf8",
				Overwrite:   true,
			},
			expectError: false,
		},
		{
			name: "with no special options",
			options: &ExtractOptions{
				ExtractFile: "/home/archive.zip",
				DestPath:    "/home",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
					r.Method == "POST" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": 1,
						"pid":     "test-pid",
					})
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
			if err := client.Login(ctx); err != nil {
				t.Fatalf("Login() error = %v", err)
			}

			fs := NewFileStationService(client)

			_, err = fs.ExtractArchive(ctx, tc.options)

			if tc.expectError && err == nil {
				t.Errorf("ExtractArchive() expected error, got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("ExtractArchive() unexpected error = %v", err)
			}
		})
	}
}

// =============================================================================
// Response Structure Tests
// =============================================================================

func TestCompressStatus_Structure(t *testing.T) {
	status := &CompressStatus{
		PID:        "test-pid",
		Status:     "running",
		Progress:   50.0,
		SourcePath: "/source",
		DestPath:   "/dest",
		FileSize:   1000,
		Processed:  500,
		Error:      "",
	}

	if status.PID != "test-pid" {
		t.Errorf("CompressStatus.PID = %s, want test-pid", status.PID)
	}

	if status.Status != "running" {
		t.Errorf("CompressStatus.Status = %s, want running", status.Status)
	}

	if status.Progress != 50.0 {
		t.Errorf("CompressStatus.Progress = %f, want 50.0", status.Progress)
	}
}

func TestExtractStatus_Structure(t *testing.T) {
	status := &ExtractStatus{
		PID:         "test-pid",
		Status:      "finished",
		Progress:    100.0,
		ExtractFile: "/archive.zip",
		DestPath:    "/dest",
		FileCount:   10,
		Processed:   10,
		Error:       "",
	}

	if status.PID != "test-pid" {
		t.Errorf("ExtractStatus.PID = %s, want test-pid", status.PID)
	}

	if status.Status != "finished" {
		t.Errorf("ExtractStatus.Status = %s, want finished", status.Status)
	}

	if status.Progress != 100.0 {
		t.Errorf("ExtractStatus.Progress = %f, want 100.0", status.Progress)
	}
}

func TestExtractFile_Structure(t *testing.T) {
	file := ExtractFile{
		FileName: "test.txt",
		FileSize: 1024,
		IsFolder: 0,
	}

	if file.FileName != "test.txt" {
		t.Errorf("ExtractFile.FileName = %s, want test.txt", file.FileName)
	}

	if file.FileSize != 1024 {
		t.Errorf("ExtractFile.FileSize = %d, want 1024", file.FileSize)
	}

	if file.IsFolder != 0 {
		t.Errorf("ExtractFile.IsFolder = %d, want 0", file.IsFolder)
	}

	folder := ExtractFile{
		FileName: "folder",
		FileSize: 0,
		IsFolder: 1,
	}

	if folder.IsFolder != 1 {
		t.Errorf("ExtractFile.IsFolder = %d, want 1", folder.IsFolder)
	}
}
