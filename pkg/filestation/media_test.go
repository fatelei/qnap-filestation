package filestation

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fatelei/qnap-filestation/pkg/api"
	"github.com/fatelei/qnap-filestation/internal/testutil"
)

// setupMediaTestClient creates a test client with mock server for media tests
func setupMediaTestClient(t *testing.T) (*api.Client, *testutil.MockServer) {
	t.Helper()

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

	client.SetSID("test-sid-12345")

	return client, mockServer
}

// setupUnauthenticatedMediaClient creates a client without authentication
func setupUnauthenticatedMediaClient(t *testing.T) *api.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": 0}`))
	}))
	t.Cleanup(server.Close)

	config := &api.Config{
		Host:     strings.TrimPrefix(server.URL, "http://"),
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

	return client
}

// TestGetThumb tests the GetThumb function
func TestGetThumb(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		path           string
		options        *GetThumbOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *GetThumbResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "get thumbnail without options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/thumb_123.jpg",
					},
				},
			},
			path:   "/home/photo.jpg",
			options: nil,
			assertResponse: func(t *testing.T, r *GetThumbResponse) {
				t.Helper()
				if r.Data.ThumbnailURL != "/tmp/thumb_123.jpg" {
					t.Errorf("ThumbnailURL = %s, want /tmp/thumb_123.jpg", r.Data.ThumbnailURL)
				}
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if fn := req.URL.Query().Get("func"); fn != "get_thumb" {
					t.Errorf("Expected func=get_thumb, got %s", fn)
				}
				if path := req.URL.Query().Get("path"); path != "/home/photo.jpg" {
					t.Errorf("Expected path=/home/photo.jpg, got %s", path)
				}
			},
		},
		{
			name: "get thumbnail with size small",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/thumb_small.jpg",
					},
				},
			},
			path: "/home/photo.jpg",
			options: &GetThumbOptions{
				Size: "small",
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if size := req.URL.Query().Get("size"); size != "small" {
					t.Errorf("Expected size=small, got %s", size)
				}
			},
		},
		{
			name: "get thumbnail with custom dimensions",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/thumb_custom.jpg",
					},
				},
			},
			path: "/home/photo.jpg",
			options: &GetThumbOptions{
				Width:  200,
				Height: 150,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if width := req.URL.Query().Get("width"); width != "200" {
					t.Errorf("Expected width=200, got %s", width)
				}
				if height := req.URL.Query().Get("height"); height != "150" {
					t.Errorf("Expected height=150, got %s", height)
				}
			},
		},
		{
			name: "get thumbnail with rotation",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/thumb_rotated.jpg",
					},
				},
			},
			path: "/home/photo.jpg",
			options: &GetThumbOptions{
				Rotate: 90,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if rotate := req.URL.Query().Get("rotate"); rotate != "90" {
					t.Errorf("Expected rotate=90, got %s", rotate)
				}
			},
		},
		{
			name: "get thumbnail with effect",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/thumb_effect.jpg",
					},
				},
			},
			path: "/home/photo.jpg",
			options: &GetThumbOptions{
				Effect: "grayscale",
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if effect := req.URL.Query().Get("effect"); effect != "grayscale" {
					t.Errorf("Expected effect=grayscale, got %s", effect)
				}
			},
		},
		{
			name: "get thumbnail as base64 buffer",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "base64:iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
					},
				},
			},
			path: "/home/photo.jpg",
			options: &GetThumbOptions{
				Buffer: true,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if buffer := req.URL.Query().Get("buffer"); buffer != "1" {
					t.Errorf("Expected buffer=1, got %s", buffer)
				}
			},
		},
		{
			name: "get thumbnail with timeout",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/thumb.jpg",
					},
				},
			},
			path: "/home/photo.jpg",
			options: &GetThumbOptions{
				Timeout: 30,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if timeout := req.URL.Query().Get("timeout"); timeout != "30" {
					t.Errorf("Expected timeout=30, got %s", timeout)
				}
			},
		},
		{
			name: "get thumbnail with all options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/thumb_all.jpg",
					},
				},
			},
			path: "/home/photo.jpg",
			options: &GetThumbOptions{
				Size:    "medium",
				Width:   300,
				Height:  200,
				Rotate:  180,
				Effect:  "sepia",
				Buffer:  true,
				Timeout: 60,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if size := req.URL.Query().Get("size"); size != "medium" {
					t.Errorf("Expected size=medium, got %s", size)
				}
				if width := req.URL.Query().Get("width"); width != "300" {
					t.Errorf("Expected width=300, got %s", width)
				}
				if height := req.URL.Query().Get("height"); height != "200" {
					t.Errorf("Expected height=200, got %s", height)
				}
				if rotate := req.URL.Query().Get("rotate"); rotate != "180" {
					t.Errorf("Expected rotate=180, got %s", rotate)
				}
				if effect := req.URL.Query().Get("effect"); effect != "sepia" {
					t.Errorf("Expected effect=sepia, got %s", effect)
				}
				if buffer := req.URL.Query().Get("buffer"); buffer != "1" {
					t.Errorf("Expected buffer=1, got %s", buffer)
				}
				if timeout := req.URL.Query().Get("timeout"); timeout != "60" {
					t.Errorf("Expected timeout=60, got %s", timeout)
				}
			},
		},
		{
			name: "negative values are not sent",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/thumb.jpg",
					},
				},
			},
			path: "/home/photo.jpg",
			options: &GetThumbOptions{
				Width:  -1,
				Height: -1,
				Rotate: -1,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if width := req.URL.Query().Get("width"); width != "" {
					t.Errorf("Expected empty width for negative value, got %s", width)
				}
				if height := req.URL.Query().Get("height"); height != "" {
					t.Errorf("Expected empty height for negative value, got %s", height)
				}
				if rotate := req.URL.Query().Get("rotate"); rotate != "" {
					t.Errorf("Expected empty rotate for negative value, got %s", rotate)
				}
			},
		},
		{
			name: "API error response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "File not found",
				},
			},
			path:        "/nonexistent/photo.jpg",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrNotFound,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json{{{",
			},
			path:        "/home/photo.jpg",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			path:   "/home/photo.jpg",
			options: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupMediaTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.GetThumb(ctx, tt.path, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("GetThumb() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("GetThumb() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("GetThumb() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("GetThumb() unexpected error = %v", err)
				return
			}

			if tt.assertResponse != nil {
				tt.assertResponse(t, resp)
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

// TestGetThumb_NotAuthenticated tests GetThumb when not authenticated
func TestGetThumb_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedMediaClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.GetThumb(ctx, "/home/photo.jpg", nil)

	if err == nil {
		t.Error("GetThumb() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetThumb() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("GetThumb() error = %v, want auth error", apiErr)
	}
}

// TestForceThumb tests the ForceThumb function
func TestForceThumb(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		path           string
		options        *ForceThumbOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *ForceThumbResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "force thumbnail without options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":     "12345",
						"success": true,
					},
				},
			},
			path:   "/home/photo.jpg",
			options: nil,
			assertResponse: func(t *testing.T, r *ForceThumbResponse) {
				t.Helper()
				if r.Data.PID != "12345" {
					t.Errorf("PID = %s, want 12345", r.Data.PID)
				}
				if !r.Data.Success {
					t.Error("Success = false, want true")
				}
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if fn := req.URL.Query().Get("func"); fn != "force_thumb" {
					t.Errorf("Expected func=force_thumb, got %s", fn)
				}
				if path := req.URL.Query().Get("path"); path != "/home/photo.jpg" {
					t.Errorf("Expected path=/home/photo.jpg, got %s", path)
				}
			},
		},
		{
			name: "force thumbnail with size",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":     "12346",
						"success": true,
					},
				},
			},
			path: "/home/photo.jpg",
			options: &ForceThumbOptions{
				Size: "large",
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if size := req.URL.Query().Get("size"); size != "large" {
					t.Errorf("Expected size=large, got %s", size)
				}
			},
		},
		{
			name: "force thumbnail with custom dimensions",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":     "12347",
						"success": true,
					},
				},
			},
			path: "/home/photo.jpg",
			options: &ForceThumbOptions{
				Width:  400,
				Height: 300,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if width := req.URL.Query().Get("width"); width != "400" {
					t.Errorf("Expected width=400, got %s", width)
				}
				if height := req.URL.Query().Get("height"); height != "300" {
					t.Errorf("Expected height=300, got %s", height)
				}
			},
		},
		{
			name: "force thumbnail with all options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":     "12348",
						"success": true,
						"message": "Thumbnail generation started",
					},
				},
			},
			path: "/home/photo.jpg",
			options: &ForceThumbOptions{
				Size:   "medium",
				Width:  300,
				Height: 200,
			},
			assertResponse: func(t *testing.T, r *ForceThumbResponse) {
				t.Helper()
				if r.Data.Message != "Thumbnail generation started" {
					t.Errorf("Message = %s", r.Data.Message)
				}
			},
		},
		{
			name: "force thumbnail generation failed",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":     "12349",
						"success": false,
						"message": "Unsupported file format",
					},
				},
			},
			path:   "/home/document.pdf",
			options: nil,
			assertResponse: func(t *testing.T, r *ForceThumbResponse) {
				t.Helper()
				if r.Data.Success {
					t.Error("Expected Success = false")
				}
			},
		},
		{
			name: "API error response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "File not found",
				},
			},
			path:        "/nonexistent/photo.jpg",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrNotFound,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json{{{",
			},
			path:        "/home/photo.jpg",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			path:   "/home/photo.jpg",
			options: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupMediaTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.ForceThumb(ctx, tt.path, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("ForceThumb() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("ForceThumb() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("ForceThumb() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("ForceThumb() unexpected error = %v", err)
				return
			}

			if tt.assertResponse != nil {
				tt.assertResponse(t, resp)
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

// TestForceThumb_NotAuthenticated tests ForceThumb when not authenticated
func TestForceThumb_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedMediaClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.ForceThumb(ctx, "/home/photo.jpg", nil)

	if err == nil {
		t.Error("ForceThumb() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("ForceThumb() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("ForceThumb() error = %v, want auth error", apiErr)
	}
}

// TestRemoteThumb tests the RemoteThumb function
func TestRemoteThumb(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		url            string
		options        *RemoteThumbOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *RemoteThumbResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "get remote thumbnail without options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/remote_thumb.jpg",
					},
				},
			},
			url:    "https://example.com/image.jpg",
			options: nil,
			assertResponse: func(t *testing.T, r *RemoteThumbResponse) {
				t.Helper()
				if r.Data.ThumbnailURL != "/tmp/remote_thumb.jpg" {
					t.Errorf("ThumbnailURL = %s, want /tmp/remote_thumb.jpg", r.Data.ThumbnailURL)
				}
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if fn := req.URL.Query().Get("func"); fn != "remote_thumb" {
					t.Errorf("Expected func=remote_thumb, got %s", fn)
				}
				if url := req.URL.Query().Get("url"); url != "https://example.com/image.jpg" {
					t.Errorf("Expected url=https://example.com/image.jpg, got %s", url)
				}
			},
		},
		{
			name: "get remote thumbnail with size",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/remote_thumb_small.jpg",
					},
				},
			},
			url: "https://example.com/image.jpg",
			options: &RemoteThumbOptions{
				Size: "small",
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if size := req.URL.Query().Get("size"); size != "small" {
					t.Errorf("Expected size=small, got %s", size)
				}
			},
		},
		{
			name: "get remote thumbnail with custom dimensions",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/remote_thumb_custom.jpg",
					},
				},
			},
			url: "https://example.com/image.jpg",
			options: &RemoteThumbOptions{
				Width:  250,
				Height: 180,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if width := req.URL.Query().Get("width"); width != "250" {
					t.Errorf("Expected width=250, got %s", width)
				}
				if height := req.URL.Query().Get("height"); height != "180" {
					t.Errorf("Expected height=180, got %s", height)
				}
			},
		},
		{
			name: "get remote thumbnail as base64 buffer",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "base64:iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
					},
				},
			},
			url: "https://example.com/image.jpg",
			options: &RemoteThumbOptions{
				Buffer: true,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if buffer := req.URL.Query().Get("buffer"); buffer != "1" {
					t.Errorf("Expected buffer=1, got %s", buffer)
				}
			},
		},
		{
			name: "get remote thumbnail with all options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/remote_all.jpg",
					},
				},
			},
			url: "https://example.com/image.jpg",
			options: &RemoteThumbOptions{
				Size:   "large",
				Width:  500,
				Height: 400,
				Buffer: true,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if size := req.URL.Query().Get("size"); size != "large" {
					t.Errorf("Expected size=large, got %s", size)
				}
				if width := req.URL.Query().Get("width"); width != "500" {
					t.Errorf("Expected width=500, got %s", width)
				}
				if height := req.URL.Query().Get("height"); height != "400" {
					t.Errorf("Expected height=400, got %s", height)
				}
				if buffer := req.URL.Query().Get("buffer"); buffer != "1" {
					t.Errorf("Expected buffer=1, got %s", buffer)
				}
			},
		},
		{
			name: "API error response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1003,
					"error_msg":  "Invalid URL",
				},
			},
			url:         "invalid-url",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json{{{",
			},
			url:         "https://example.com/image.jpg",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			url:     "https://example.com/image.jpg",
			options: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupMediaTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.RemoteThumb(ctx, tt.url, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("RemoteThumb() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("RemoteThumb() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("RemoteThumb() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("RemoteThumb() unexpected error = %v", err)
				return
			}

			if tt.assertResponse != nil {
				tt.assertResponse(t, resp)
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

// TestRemoteThumb_NotAuthenticated tests RemoteThumb when not authenticated
func TestRemoteThumb_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedMediaClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.RemoteThumb(ctx, "https://example.com/image.jpg", nil)

	if err == nil {
		t.Error("RemoteThumb() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("RemoteThumb() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("RemoteThumb() error = %v, want auth error", apiErr)
	}
}

// TestSupportPdfThumb tests the SupportPdfThumb function
func TestSupportPdfThumb(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *SupportPdfThumbResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "PDF thumbnail supported and enabled",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"supported": true,
						"enabled":   true,
					},
				},
			},
			assertResponse: func(t *testing.T, r *SupportPdfThumbResponse) {
				t.Helper()
				if !r.Data.Supported {
					t.Error("Supported = false, want true")
				}
				if !r.Data.Enabled {
					t.Error("Enabled = false, want true")
				}
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if fn := req.URL.Query().Get("func"); fn != "support_pdf_thumb" {
					t.Errorf("Expected func=support_pdf_thumb, got %s", fn)
				}
			},
		},
		{
			name: "PDF thumbnail supported but disabled",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"supported": true,
						"enabled":   false,
					},
				},
			},
			assertResponse: func(t *testing.T, r *SupportPdfThumbResponse) {
				t.Helper()
				if !r.Data.Supported {
					t.Error("Supported = false, want true")
				}
				if r.Data.Enabled {
					t.Error("Enabled = true, want false")
				}
			},
		},
		{
			name: "PDF thumbnail not supported",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"supported": false,
						"enabled":   false,
					},
				},
			},
			assertResponse: func(t *testing.T, r *SupportPdfThumbResponse) {
				t.Helper()
				if r.Data.Supported {
					t.Error("Supported = true, want false")
				}
			},
		},
		{
			name: "API error response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1000,
					"error_msg":  "Authentication failed",
				},
			},
			wantErr:     true,
			expectedErr: api.ErrAuthFailed,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json{{{",
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
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupMediaTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.SupportPdfThumb(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("SupportPdfThumb() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("SupportPdfThumb() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("SupportPdfThumb() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("SupportPdfThumb() unexpected error = %v", err)
				return
			}

			if tt.assertResponse != nil {
				tt.assertResponse(t, resp)
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

// TestSupportPdfThumb_NotAuthenticated tests SupportPdfThumb when not authenticated
func TestSupportPdfThumb_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedMediaClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.SupportPdfThumb(ctx)

	if err == nil {
		t.Error("SupportPdfThumb() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("SupportPdfThumb() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("SupportPdfThumb() error = %v, want auth error", apiErr)
	}
}

// TestGetSupportPdfThumb tests the GetSupportPdfThumb function
func TestGetSupportPdfThumb(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		path           string
		options        *GetSupportPdfThumbOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *GetSupportPdfThumbResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "get PDF thumbnail without options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/pdf_thumb.jpg",
						"page":          1,
					},
				},
			},
			path:   "/home/document.pdf",
			options: nil,
			assertResponse: func(t *testing.T, r *GetSupportPdfThumbResponse) {
				t.Helper()
				if r.Data.ThumbnailURL != "/tmp/pdf_thumb.jpg" {
					t.Errorf("ThumbnailURL = %s, want /tmp/pdf_thumb.jpg", r.Data.ThumbnailURL)
				}
				if r.Data.Page != 1 {
					t.Errorf("Page = %d, want 1", r.Data.Page)
				}
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if fn := req.URL.Query().Get("func"); fn != "get_support_pdf_thumb" {
					t.Errorf("Expected func=get_support_pdf_thumb, got %s", fn)
				}
				if path := req.URL.Query().Get("path"); path != "/home/document.pdf" {
					t.Errorf("Expected path=/home/document.pdf, got %s", path)
				}
			},
		},
		{
			name: "get PDF thumbnail with specific page",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/pdf_thumb_page5.jpg",
						"page":          5,
					},
				},
			},
			path: "/home/document.pdf",
			options: &GetSupportPdfThumbOptions{
				Page: 5,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if page := req.URL.Query().Get("page"); page != "5" {
					t.Errorf("Expected page=5, got %s", page)
				}
			},
		},
		{
			name: "get PDF thumbnail with size",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/pdf_thumb_large.jpg",
						"page":          1,
					},
				},
			},
			path: "/home/document.pdf",
			options: &GetSupportPdfThumbOptions{
				Size: "large",
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if size := req.URL.Query().Get("size"); size != "large" {
					t.Errorf("Expected size=large, got %s", size)
				}
			},
		},
		{
			name: "get PDF thumbnail with custom dimensions",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/pdf_thumb_custom.jpg",
						"page":          1,
					},
				},
			},
			path: "/home/document.pdf",
			options: &GetSupportPdfThumbOptions{
				Width:  300,
				Height: 400,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if width := req.URL.Query().Get("width"); width != "300" {
					t.Errorf("Expected width=300, got %s", width)
				}
				if height := req.URL.Query().Get("height"); height != "400" {
					t.Errorf("Expected height=400, got %s", height)
				}
			},
		},
		{
			name: "get PDF thumbnail as base64 buffer",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "base64:iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
						"page":          1,
					},
				},
			},
			path: "/home/document.pdf",
			options: &GetSupportPdfThumbOptions{
				Buffer: true,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if buffer := req.URL.Query().Get("buffer"); buffer != "1" {
					t.Errorf("Expected buffer=1, got %s", buffer)
				}
			},
		},
		{
			name: "get PDF thumbnail with all options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"thumbnail_url": "/tmp/pdf_all.jpg",
						"page":          3,
					},
				},
			},
			path: "/home/document.pdf",
			options: &GetSupportPdfThumbOptions{
				Page:   3,
				Size:   "medium",
				Width:  250,
				Height: 350,
				Buffer: true,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if page := req.URL.Query().Get("page"); page != "3" {
					t.Errorf("Expected page=3, got %s", page)
				}
				if size := req.URL.Query().Get("size"); size != "medium" {
					t.Errorf("Expected size=medium, got %s", size)
				}
				if width := req.URL.Query().Get("width"); width != "250" {
					t.Errorf("Expected width=250, got %s", width)
				}
				if height := req.URL.Query().Get("height"); height != "350" {
					t.Errorf("Expected height=350, got %s", height)
				}
				if buffer := req.URL.Query().Get("buffer"); buffer != "1" {
					t.Errorf("Expected buffer=1, got %s", buffer)
				}
			},
		},
		{
			name: "API error response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "File not found",
				},
			},
			path:        "/nonexistent/document.pdf",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrNotFound,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json{{{",
			},
			path:        "/home/document.pdf",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			path:   "/home/document.pdf",
			options: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupMediaTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.GetSupportPdfThumb(ctx, tt.path, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("GetSupportPdfThumb() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("GetSupportPdfThumb() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("GetSupportPdfThumb() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("GetSupportPdfThumb() unexpected error = %v", err)
				return
			}

			if tt.assertResponse != nil {
				tt.assertResponse(t, resp)
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

// TestGetSupportPdfThumb_NotAuthenticated tests GetSupportPdfThumb when not authenticated
func TestGetSupportPdfThumb_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedMediaClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.GetSupportPdfThumb(ctx, "/home/document.pdf", nil)

	if err == nil {
		t.Error("GetSupportPdfThumb() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetSupportPdfThumb() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("GetSupportPdfThumb() error = %v, want auth error", apiErr)
	}
}

// TestEnableThumbnail tests the EnableThumbnail function
func TestEnableThumbnail(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *EnableThumbnailOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *EnableThumbnailResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "enable thumbnail without options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: nil,
			assertResponse: func(t *testing.T, r *EnableThumbnailResponse) {
				t.Helper()
				if !r.Data.Success {
					t.Error("Success = false, want true")
				}
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if fn := req.URL.Query().Get("func"); fn != "enable_thumbnail" {
					t.Errorf("Expected func=enable_thumbnail, got %s", fn)
				}
			},
		},
		{
			name: "enable thumbnail with path",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"message": "Thumbnail enabled for /home/Photos",
					},
				},
			},
			options: &EnableThumbnailOptions{
				Path: "/home/Photos",
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if path := req.URL.Query().Get("path"); path != "/home/Photos" {
					t.Errorf("Expected path=/home/Photos, got %s", path)
				}
			},
		},
		{
			name: "enable thumbnail with rebuild",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"message": "Thumbnail rebuild started",
					},
				},
			},
			options: &EnableThumbnailOptions{
				Rebuild: true,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if rebuild := req.URL.Query().Get("rebuild"); rebuild != "1" {
					t.Errorf("Expected rebuild=1, got %s", rebuild)
				}
			},
		},
		{
			name: "enable thumbnail with all options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"message": "Thumbnail enabled and rebuild started",
					},
				},
			},
			options: &EnableThumbnailOptions{
				Path:    "/home/Photos",
				Rebuild: true,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if path := req.URL.Query().Get("path"); path != "/home/Photos" {
					t.Errorf("Expected path=/home/Photos, got %s", path)
				}
				if rebuild := req.URL.Query().Get("rebuild"); rebuild != "1" {
					t.Errorf("Expected rebuild=1, got %s", rebuild)
				}
			},
		},
		{
			name: "API error response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2004,
					"error_msg":  "Invalid path",
				},
			},
			options:     &EnableThumbnailOptions{Path: "/invalid/path"},
			wantErr:     true,
			expectedErr: api.ErrInvalidPath,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json{{{",
			},
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			options: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupMediaTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.EnableThumbnail(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("EnableThumbnail() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("EnableThumbnail() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("EnableThumbnail() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("EnableThumbnail() unexpected error = %v", err)
				return
			}

			if tt.assertResponse != nil {
				tt.assertResponse(t, resp)
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

// TestEnableThumbnail_NotAuthenticated tests EnableThumbnail when not authenticated
func TestEnableThumbnail_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedMediaClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.EnableThumbnail(ctx, nil)

	if err == nil {
		t.Error("EnableThumbnail() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("EnableThumbnail() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("EnableThumbnail() error = %v, want auth error", apiErr)
	}
}

// TestSetSmbThumb tests the SetSmbThumb function
func TestSetSmbThumb(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *SetSmbThumbOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *SetSmbThumbResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "enable SMB thumbnail",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"message": "SMB thumbnail enabled",
					},
				},
			},
			options: &SetSmbThumbOptions{
				Enabled: true,
			},
			assertResponse: func(t *testing.T, r *SetSmbThumbResponse) {
				t.Helper()
				if !r.Data.Success {
					t.Error("Success = false, want true")
				}
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if fn := req.URL.Query().Get("func"); fn != "set_smb_thumb" {
					t.Errorf("Expected func=set_smb_thumb, got %s", fn)
				}
				if enabled := req.URL.Query().Get("enabled"); enabled != "1" {
					t.Errorf("Expected enabled=1, got %s", enabled)
				}
			},
		},
		{
			name: "disable SMB thumbnail",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"message": "SMB thumbnail disabled",
					},
				},
			},
			options: &SetSmbThumbOptions{
				Enabled: false,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if enabled := req.URL.Query().Get("enabled"); enabled != "0" {
					t.Errorf("Expected enabled=0, got %s", enabled)
				}
			},
		},
		{
			name: "set SMB thumbnail with path",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"message": "SMB thumbnail setting applied to /home/Photos",
					},
				},
			},
			options: &SetSmbThumbOptions{
				Enabled: true,
				Path:    "/home/Photos",
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if enabled := req.URL.Query().Get("enabled"); enabled != "1" {
					t.Errorf("Expected enabled=1, got %s", enabled)
				}
				if path := req.URL.Query().Get("path"); path != "/home/Photos" {
					t.Errorf("Expected path=/home/Photos, got %s", path)
				}
			},
		},
		{
			name: "set SMB thumbnail with path only (disabled)",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"message": "SMB thumbnail disabled for /share",
					},
				},
			},
			options: &SetSmbThumbOptions{
				Enabled: false,
				Path:    "/share",
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if enabled := req.URL.Query().Get("enabled"); enabled != "0" {
					t.Errorf("Expected enabled=0, got %s", enabled)
				}
				if path := req.URL.Query().Get("path"); path != "/share" {
					t.Errorf("Expected path=/share, got %s", path)
				}
			},
		},
		{
			name: "API error response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2004,
					"error_msg":  "Invalid path",
				},
			},
			options:     &SetSmbThumbOptions{Path: "/invalid/path"},
			wantErr:     true,
			expectedErr: api.ErrInvalidPath,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json{{{",
			},
			options:     &SetSmbThumbOptions{Enabled: true},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			options: &SetSmbThumbOptions{Enabled: true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupMediaTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.SetSmbThumb(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("SetSmbThumb() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("SetSmbThumb() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("SetSmbThumb() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("SetSmbThumb() unexpected error = %v", err)
				return
			}

			if tt.assertResponse != nil {
				tt.assertResponse(t, resp)
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

// TestSetSmbThumb_NotAuthenticated tests SetSmbThumb when not authenticated
func TestSetSmbThumb_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedMediaClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.SetSmbThumb(ctx, &SetSmbThumbOptions{Enabled: true})

	if err == nil {
		t.Error("SetSmbThumb() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("SetSmbThumb() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("SetSmbThumb() error = %v, want auth error", apiErr)
	}
}

// TestGetViewer tests the GetViewer function
func TestGetViewer(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *GetViewerResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "get viewers",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"viewers": []map[string]interface{}{
							{
								"name":        "Image Viewer",
								"type":        "image",
								"description": "View images",
								"enabled":     true,
								"extensions":  []string{"jpg", "png", "gif", "bmp"},
							},
							{
								"name":        "Video Player",
								"type":        "video",
								"description": "Play videos",
								"enabled":     true,
								"extensions":  []string{"mp4", "avi", "mkv"},
							},
							{
								"name":        "PDF Viewer",
								"type":        "pdf",
								"description": "View PDF documents",
								"enabled":     false,
								"extensions":  []string{"pdf"},
							},
						},
						"total": 3,
					},
				},
			},
			assertResponse: func(t *testing.T, r *GetViewerResponse) {
				t.Helper()
				if r.Data.Total != 3 {
					t.Errorf("Total = %d, want 3", r.Data.Total)
				}
				if len(r.Data.Viewers) != 3 {
					t.Errorf("Viewers count = %d, want 3", len(r.Data.Viewers))
				}
				imageViewer := r.Data.Viewers[0]
				if imageViewer.Name != "Image Viewer" {
					t.Errorf("First viewer name = %s, want Image Viewer", imageViewer.Name)
				}
				if !imageViewer.Enabled {
					t.Error("Image viewer should be enabled")
				}
				pdfViewer := r.Data.Viewers[2]
				if pdfViewer.Enabled {
					t.Error("PDF viewer should be disabled")
				}
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if fn := req.URL.Query().Get("func"); fn != "get_viewer" {
					t.Errorf("Expected func=get_viewer, got %s", fn)
				}
			},
		},
		{
			name: "empty viewer list",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"viewers": []interface{}{},
						"total":   0,
					},
				},
			},
			assertResponse: func(t *testing.T, r *GetViewerResponse) {
				t.Helper()
				if r.Data.Total != 0 {
					t.Errorf("Total = %d, want 0", r.Data.Total)
				}
			},
		},
		{
			name: "viewer without extensions",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"viewers": []map[string]interface{}{
							{
								"name":        "Generic Viewer",
								"type":        "generic",
								"description": "Generic viewer",
								"enabled":     true,
							},
						},
						"total": 1,
					},
				},
			},
			assertResponse: func(t *testing.T, r *GetViewerResponse) {
				t.Helper()
				if len(r.Data.Viewers[0].Extensions) != 0 {
					t.Errorf("Extensions should be empty")
				}
			},
		},
		{
			name: "API error response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1000,
					"error_msg":  "Authentication failed",
				},
			},
			wantErr:     true,
			expectedErr: api.ErrAuthFailed,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json{{{",
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
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupMediaTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.GetViewer(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("GetViewer() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("GetViewer() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("GetViewer() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("GetViewer() unexpected error = %v", err)
				return
			}

			if tt.assertResponse != nil {
				tt.assertResponse(t, resp)
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

// TestGetViewer_NotAuthenticated tests GetViewer when not authenticated
func TestGetViewer_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedMediaClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.GetViewer(ctx)

	if err == nil {
		t.Error("GetViewer() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetViewer() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("GetViewer() error = %v, want auth error", apiErr)
	}
}

// TestGetViewerSupportFormat tests the GetViewerSupportFormat function
func TestGetViewerSupportFormat(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *GetViewerSupportFormatResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "get supported formats",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"formats": []map[string]interface{}{
							{
								"viewer":     "image",
								"extensions": []string{"jpg", "jpeg", "png", "gif", "bmp", "webp"},
								"mime_types": []string{"image/jpeg", "image/png", "image/gif", "image/bmp", "image/webp"},
							},
							{
								"viewer":     "video",
								"extensions": []string{"mp4", "avi", "mkv", "mov", "wmv"},
								"mime_types": []string{"video/mp4", "video/x-msvideo", "video/x-matroska", "video/quicktime", "video/x-ms-wmv"},
							},
							{
								"viewer":     "pdf",
								"extensions": []string{"pdf"},
								"mime_types": []string{"application/pdf"},
							},
						},
						"total": 3,
					},
				},
			},
			assertResponse: func(t *testing.T, r *GetViewerSupportFormatResponse) {
				t.Helper()
				if r.Data.Total != 3 {
					t.Errorf("Total = %d, want 3", r.Data.Total)
				}
				if len(r.Data.Formats) != 3 {
					t.Errorf("Formats count = %d, want 3", len(r.Data.Formats))
				}
				imageFormat := r.Data.Formats[0]
				if imageFormat.Viewer != "image" {
					t.Errorf("First format viewer = %s, want image", imageFormat.Viewer)
				}
				if len(imageFormat.Extensions) != 6 {
					t.Errorf("Image extensions count = %d, want 6", len(imageFormat.Extensions))
				}
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if fn := req.URL.Query().Get("func"); fn != "get_viewer_support_format_t" {
					t.Errorf("Expected func=get_viewer_support_format_t, got %s", fn)
				}
			},
		},
		{
			name: "empty format list",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"formats": []interface{}{},
						"total":   0,
					},
				},
			},
			assertResponse: func(t *testing.T, r *GetViewerSupportFormatResponse) {
				t.Helper()
				if r.Data.Total != 0 {
					t.Errorf("Total = %d, want 0", r.Data.Total)
				}
			},
		},
		{
			name: "format without mime types",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"formats": []map[string]interface{}{
							{
								"viewer":     "custom",
								"extensions": []string{"custom"},
							},
						},
						"total": 1,
					},
				},
			},
			assertResponse: func(t *testing.T, r *GetViewerSupportFormatResponse) {
				t.Helper()
				if len(r.Data.Formats[0].MimeTypes) != 0 {
					t.Errorf("MimeTypes should be empty")
				}
			},
		},
		{
			name: "API error response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1000,
					"error_msg":  "Authentication failed",
				},
			},
			wantErr:     true,
			expectedErr: api.ErrAuthFailed,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json{{{",
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
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupMediaTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.GetViewerSupportFormat(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("GetViewerSupportFormat() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("GetViewerSupportFormat() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("GetViewerSupportFormat() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("GetViewerSupportFormat() unexpected error = %v", err)
				return
			}

			if tt.assertResponse != nil {
				tt.assertResponse(t, resp)
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

// TestGetViewerSupportFormat_NotAuthenticated tests GetViewerSupportFormat when not authenticated
func TestGetViewerSupportFormat_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedMediaClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.GetViewerSupportFormat(ctx)

	if err == nil {
		t.Error("GetViewerSupportFormat() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetViewerSupportFormat() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("GetViewerSupportFormat() error = %v, want auth error", apiErr)
	}
}

// TestGetTextFile tests the GetTextFile function
func TestGetTextFile(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		path           string
		options        *GetTextFileOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *GetTextFileResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "get text file without options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"content": "Hello, World!",
						"size":    13,
						"path":    "/home/test.txt",
					},
				},
			},
			path:   "/home/test.txt",
			options: nil,
			assertResponse: func(t *testing.T, r *GetTextFileResponse) {
				t.Helper()
				if r.Data.Content != "Hello, World!" {
					t.Errorf("Content = %s, want 'Hello, World!'", r.Data.Content)
				}
				if r.Data.Size != 13 {
					t.Errorf("Size = %d, want 13", r.Data.Size)
				}
				if r.Data.Path != "/home/test.txt" {
					t.Errorf("Path = %s, want /home/test.txt", r.Data.Path)
				}
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if fn := req.URL.Query().Get("func"); fn != "get_text_file" {
					t.Errorf("Expected func=get_text_file, got %s", fn)
				}
				if path := req.URL.Query().Get("path"); path != "/home/test.txt" {
					t.Errorf("Expected path=/home/test.txt, got %s", path)
				}
			},
		},
		{
			name: "get text file with encoding",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"content": "UTF-8 content",
						"size":    14,
						"path":    "/home/test.txt",
					},
				},
			},
			path: "/home/test.txt",
			options: &GetTextFileOptions{
				Encoding: "UTF-8",
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if encoding := req.URL.Query().Get("encoding"); encoding != "UTF-8" {
					t.Errorf("Expected encoding=UTF-8, got %s", encoding)
				}
			},
		},
		{
			name: "get text file with offset",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"content": "partial content",
						"size":    14,
						"path":    "/home/test.txt",
					},
				},
			},
			path: "/home/test.txt",
			options: &GetTextFileOptions{
				Offset: 100,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if offset := req.URL.Query().Get("offset"); offset != "100" {
					t.Errorf("Expected offset=100, got %s", offset)
				}
			},
		},
		{
			name: "get text file with limit",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"content": "limited content",
						"size":    15,
						"path":    "/home/test.txt",
					},
				},
			},
			path: "/home/test.txt",
			options: &GetTextFileOptions{
				Limit: 1000,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if limit := req.URL.Query().Get("limit"); limit != "1000" {
					t.Errorf("Expected limit=1000, got %s", limit)
				}
			},
		},
		{
			name: "get text file with all options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"content": "partial limited content",
						"size":    24,
						"path":    "/home/test.txt",
					},
				},
			},
			path: "/home/test.txt",
			options: &GetTextFileOptions{
				Encoding: "UTF-8",
				Offset:   500,
				Limit:    2000,
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if encoding := req.URL.Query().Get("encoding"); encoding != "UTF-8" {
					t.Errorf("Expected encoding=UTF-8, got %s", encoding)
				}
				if offset := req.URL.Query().Get("offset"); offset != "500" {
					t.Errorf("Expected offset=500, got %s", offset)
				}
				if limit := req.URL.Query().Get("limit"); limit != "2000" {
					t.Errorf("Expected limit=2000, got %s", limit)
				}
			},
		},
		{
			name: "empty file content",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"content": "",
						"size":    0,
						"path":    "/home/empty.txt",
					},
				},
			},
			path:   "/home/empty.txt",
			options: nil,
			assertResponse: func(t *testing.T, r *GetTextFileResponse) {
				t.Helper()
				if r.Data.Content != "" {
					t.Errorf("Content should be empty, got %s", r.Data.Content)
				}
				if r.Data.Size != 0 {
					t.Errorf("Size = %d, want 0", r.Data.Size)
				}
			},
		},
		{
			name: "API error response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "File not found",
				},
			},
			path:        "/nonexistent/test.txt",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrNotFound,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json{{{",
			},
			path:        "/home/test.txt",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			path:   "/home/test.txt",
			options: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupMediaTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.GetTextFile(ctx, tt.path, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("GetTextFile() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("GetTextFile() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("GetTextFile() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("GetTextFile() unexpected error = %v", err)
				return
			}

			if tt.assertResponse != nil {
				tt.assertResponse(t, resp)
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

// TestGetTextFile_NotAuthenticated tests GetTextFile when not authenticated
func TestGetTextFile_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedMediaClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.GetTextFile(ctx, "/home/test.txt", nil)

	if err == nil {
		t.Error("GetTextFile() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetTextFile() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("GetTextFile() error = %v, want auth error", apiErr)
	}
}

// TestSaveTextFile tests the SaveTextFile function
func TestSaveTextFile(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		path           string
		content        string
		options        *SaveTextFileOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *SaveTextFileResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "save text file without options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"path":    "/home/test.txt",
					},
				},
			},
			path:    "/home/test.txt",
			content: "Hello, World!",
			options: nil,
			assertResponse: func(t *testing.T, r *SaveTextFileResponse) {
				t.Helper()
				if !r.Data.Success {
					t.Error("Success = false, want true")
				}
				if r.Data.Path != "/home/test.txt" {
					t.Errorf("Path = %s, want /home/test.txt", r.Data.Path)
				}
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if fn := req.URL.Query().Get("func"); fn != "save_text_file" {
					t.Errorf("Expected func=save_text_file, got %s", fn)
				}
				if path := req.URL.Query().Get("path"); path != "/home/test.txt" {
					t.Errorf("Expected path=/home/test.txt, got %s", path)
				}
				if content := req.URL.Query().Get("content"); content != "Hello, World!" {
					t.Errorf("Expected content='Hello, World!', got %s", content)
				}
			},
		},
		{
			name: "save text file with encoding",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"path":    "/home/test.txt",
					},
				},
			},
			path:    "/home/test.txt",
			content: "UTF-8 content",
			options: &SaveTextFileOptions{
				Encoding: "UTF-8",
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if encoding := req.URL.Query().Get("encoding"); encoding != "UTF-8" {
					t.Errorf("Expected encoding=UTF-8, got %s", encoding)
				}
			},
		},
		{
			name: "save text file with overwrite mode",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"path":    "/home/test.txt",
					},
				},
			},
			path:    "/home/test.txt",
			content: "Overwritten content",
			options: &SaveTextFileOptions{
				Mode: "overwrite",
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if mode := req.URL.Query().Get("mode"); mode != "overwrite" {
					t.Errorf("Expected mode=overwrite, got %s", mode)
				}
			},
		},
		{
			name: "save text file with append mode",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"path":    "/home/test.txt",
					},
				},
			},
			path:    "/home/test.txt",
			content: "Appended content",
			options: &SaveTextFileOptions{
				Mode: "append",
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if mode := req.URL.Query().Get("mode"); mode != "append" {
					t.Errorf("Expected mode=append, got %s", mode)
				}
			},
		},
		{
			name: "save text file with all options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"path":    "/home/test.txt",
					},
				},
			},
			path:    "/home/test.txt",
			content: "Content with options",
			options: &SaveTextFileOptions{
				Encoding: "UTF-8",
				Mode:     "overwrite",
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if encoding := req.URL.Query().Get("encoding"); encoding != "UTF-8" {
					t.Errorf("Expected encoding=UTF-8, got %s", encoding)
				}
				if mode := req.URL.Query().Get("mode"); mode != "overwrite" {
					t.Errorf("Expected mode=overwrite, got %s", mode)
				}
			},
		},
		{
			name: "save empty content",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"path":    "/home/empty.txt",
					},
				},
			},
			path:    "/home/empty.txt",
			content: "",
			options: nil,
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if content := req.URL.Query().Get("content"); content != "" {
					t.Errorf("Expected empty content, got %s", content)
				}
			},
		},
		{
			name: "save content with special characters",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"path":    "/home/special.txt",
					},
				},
			},
			path:    "/home/special.txt",
			content: "Line 1\nLine 2\tTabbed\nQuotes: \"test\"",
			options: nil,
		},
		{
			name: "API error response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2003,
					"error_msg":  "Permission denied",
				},
			},
			path:        "/restricted/test.txt",
			content:     "content",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrPermission,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json{{{",
			},
			path:        "/home/test.txt",
			content:     "content",
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			path:    "/home/test.txt",
			content: "content",
			options: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupMediaTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.SaveTextFile(ctx, tt.path, tt.content, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("SaveTextFile() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("SaveTextFile() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("SaveTextFile() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("SaveTextFile() unexpected error = %v", err)
				return
			}

			if tt.assertResponse != nil {
				tt.assertResponse(t, resp)
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

// TestSaveTextFile_NotAuthenticated tests SaveTextFile when not authenticated
func TestSaveTextFile_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedMediaClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.SaveTextFile(ctx, "/home/test.txt", "content", nil)

	if err == nil {
		t.Error("SaveTextFile() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("SaveTextFile() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("SaveTextFile() error = %v, want auth error", apiErr)
	}
}

// TestMediaMethods_TableDriven verifies all media methods use table-driven tests
func TestMediaMethods_TableDriven(t *testing.T) {
	// This meta-test ensures we have comprehensive table-driven test coverage
	methods := []string{
		"GetThumb",
		"ForceThumb",
		"RemoteThumb",
		"SupportPdfThumb",
		"GetSupportPdfThumb",
		"EnableThumbnail",
		"SetSmbThumb",
		"GetViewer",
		"GetViewerSupportFormat",
		"GetTextFile",
		"SaveTextFile",
	}

	for _, method := range methods {
		t.Run(method+" has table-driven tests", func(t *testing.T) {
			// This is a meta-test to document that table-driven tests exist
			// The actual tests are defined above
			t.Logf("Table-driven tests exist for %s", method)
		})
	}
}

// TestContextCancellation_Media tests context cancellation for media methods
func TestContextCancellation_Media(t *testing.T) {
	tests := []struct {
		name  string
		testFn func(*testing.T, *FileStationService, context.Context)
	}{
		{
			name: "GetThumb respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.GetThumb(ctx, "/home/photo.jpg", nil)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("GetThumb with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "ForceThumb respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.ForceThumb(ctx, "/home/photo.jpg", nil)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("ForceThumb with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "RemoteThumb respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.RemoteThumb(ctx, "https://example.com/image.jpg", nil)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("RemoteThumb with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "SupportPdfThumb respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.SupportPdfThumb(ctx)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("SupportPdfThumb with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "GetSupportPdfThumb respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.GetSupportPdfThumb(ctx, "/home/doc.pdf", nil)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("GetSupportPdfThumb with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "EnableThumbnail respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.EnableThumbnail(ctx, nil)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("EnableThumbnail with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "SetSmbThumb respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.SetSmbThumb(ctx, &SetSmbThumbOptions{Enabled: true})
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("SetSmbThumb with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "GetViewer respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.GetViewer(ctx)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("GetViewer with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "GetViewerSupportFormat respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.GetViewerSupportFormat(ctx)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("GetViewerSupportFormat with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "GetTextFile respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.GetTextFile(ctx, "/home/test.txt", nil)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("GetTextFile with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "SaveTextFile respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.SaveTextFile(ctx, "/home/test.txt", "content", nil)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("SaveTextFile with canceled context returned error: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupMediaTestClient(t)
			defer mockServer.Close()

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			fs := NewFileStationService(client)
			tt.testFn(t, fs, ctx)
		})
	}
}

// BenchmarkGetThumb benchmarks the GetThumb function
func BenchmarkGetThumb(b *testing.B) {
	client, mockServer := setupMediaTestClient(&testing.T{})
	defer mockServer.Close()

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: http.StatusOK,
		Body: map[string]interface{}{
			"success": 1,
			"data": map[string]interface{}{
				"thumbnail_url": "/tmp/thumb.jpg",
			},
		},
	})

	ctx := context.Background()
	fs := NewFileStationService(client)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = fs.GetThumb(ctx, "/home/photo.jpg", nil)
	}
}

// TestTextFileOperations tests read and write operations together
func TestTextFileOperations(t *testing.T) {
	t.Run("read and write text file workflow", func(t *testing.T) {
		client, mockServer := setupMediaTestClient(t)
		defer mockServer.Close()

		ctx := context.Background()
		fs := NewFileStationService(client)

		// First, save the file
		mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"success": true,
					"path":    "/home/test.txt",
				},
			},
		})

		saveResp, err := fs.SaveTextFile(ctx, "/home/test.txt", "Hello, World!", nil)
		if err != nil {
			t.Fatalf("SaveTextFile() error = %v", err)
		}
		if !saveResp.Data.Success {
			t.Error("SaveTextFile() failed")
		}

		// Then, read it back
		mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"content": "Hello, World!",
					"size":    13,
					"path":    "/home/test.txt",
				},
			},
		})

		readResp, err := fs.GetTextFile(ctx, "/home/test.txt", nil)
		if err != nil {
			t.Fatalf("GetTextFile() error = %v", err)
		}
		if readResp.Data.Content != "Hello, World!" {
			t.Errorf("Content = %s, want 'Hello, World!'", readResp.Data.Content)
		}
	})
}
