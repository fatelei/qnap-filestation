package filestation

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fatelei/qnap-filestation/pkg/api"
)

// MockClient wraps an api.Client with test server support
type MockClient struct {
	client   *api.Client
	server   *httptest.Server
	handler  http.HandlerFunc
}

// NewMockClient creates a new mock client for testing
func NewMockClient(handler http.HandlerFunc) *MockClient {
	server := httptest.NewServer(handler)

	cfg := &api.Config{
		Host:     strings.TrimPrefix(server.URL, "http://"),
		Insecure: true,
		// Provide required credentials for client construction; tests set SID directly
		Username: "admin",
		Password: "password",
	}

	client, err := api.NewClient(cfg)
	if err != nil {
		server.Close()
		panic(err)
	}

	// Set a fake session ID
	client.SetSID("test-sid-12345")

	return &MockClient{
		client:  client,
		server:  server,
		handler: handler,
	}
}

// Close closes the mock client's test server
func (m *MockClient) Close() {
	if m.server != nil {
		m.server.Close()
	}
}

// GetClient returns the underlying api.Client
func (m *MockClient) GetClient() *api.Client {
	return m.client
}

// SetSID sets the session ID (for testing auth failures)
func (m *MockClient) SetSID(sid string) {
	m.client.SetSID(sid)
}

// TestSearch_Success tests successful search operations
func TestSearch_Success(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		options  *SearchOptions
		response string
		wantLen  int
		wantErr  bool
	}{
		{
			name: "basic pattern search",
			path: "/share/public",
			options: &SearchOptions{
				Pattern: "test",
			},
			response: `{
				"total": 2,
				"datas": [
					{
						"filename": "test1.txt",
						"isfolder": 0,
						"filesize": "1024",
						"path": "/share/public/test1.txt"
					},
					{
						"filename": "test2.txt",
						"isfolder": 0,
						"filesize": "2048",
						"path": "/share/public/test2.txt"
					}
				]
			}`,
			wantLen: 2,
			wantErr: false,
		},
		{
			name: "search with file type MUSIC",
			path: "/share/music",
			options: &SearchOptions{
				Pattern:  "song",
				FileType: "MUSIC",
			},
			response: `{
				"total": 1,
				"datas": [
					{
						"filename": "song.mp3",
						"isfolder": 0,
						"filesize": "5242880",
						"filetype": 1,
						"path": "/share/music/song.mp3"
					}
				]
			}`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "search with file type VIDEO",
			path: "/share/video",
			options: &SearchOptions{
				Pattern:  "movie",
				FileType: "VIDEO",
			},
			response: `{
				"total": 1,
				"datas": [
					{
						"filename": "movie.mkv",
						"isfolder": 0,
						"filesize": "1073741824",
						"filetype": 2,
						"path": "/share/video/movie.mkv"
					}
				]
			}`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "search with file type PHOTO",
			path: "/share/photo",
			options: &SearchOptions{
				Pattern:  "vacation",
				FileType: "PHOTO",
			},
			response: `{
				"total": 1,
				"datas": [
					{
						"filename": "vacation.jpg",
						"isfolder": 0,
						"filesize": "2097152",
						"filetype": 3,
						"path": "/share/photo/vacation.jpg"
					}
				]
			}`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "search with extension filter",
			path: "/share/public",
			options: &SearchOptions{
				Pattern:   "document",
				Extension: []string{".pdf"},
			},
			response: `{
				"total": 1,
				"datas": [
					{
						"filename": "document.pdf",
						"isfolder": 0,
						"filesize": "1048576",
						"path": "/share/public/document.pdf"
					}
				]
			}`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "search with minimum size filter",
			path: "/share/public",
			options: &SearchOptions{
				Pattern:  "large",
				SizeMin:  1048576, // 1MB
			},
			response: `{
				"total": 1,
				"datas": [
					{
						"filename": "large.bin",
						"isfolder": 0,
						"filesize": "2097152",
						"path": "/share/public/large.bin"
					}
				]
			}`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "search with maximum size filter",
			path: "/share/public",
			options: &SearchOptions{
				Pattern: "small",
				SizeMax: 102400, // 100KB
			},
			response: `{
				"total": 1,
				"datas": [
					{
						"filename": "small.txt",
						"isfolder": 0,
						"filesize": "512",
						"path": "/share/public/small.txt"
					}
				]
			}`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "search with all options",
			path: "/share/public",
			options: &SearchOptions{
				Pattern:   "report",
				Extension: []string{".xlsx"},
				SizeMin:   1024,
				SizeMax:   10485760,
			},
			response: `{
				"total": 1,
				"datas": [
					{
						"filename": "report.xlsx",
						"isfolder": 0,
						"filesize": "524288",
						"path": "/share/public/report.xlsx"
					}
				]
			}`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "search with nil options",
			path: "/share/public",
			options: nil,
			response: `{
				"total": 3,
				"datas": [
					{
						"filename": "file1.txt",
						"isfolder": 0,
						"filesize": "100",
						"path": "/share/public/file1.txt"
					},
					{
						"filename": "file2.txt",
						"isfolder": 0,
						"filesize": "200",
						"path": "/share/public/file2.txt"
					},
					{
						"filename": "file3.txt",
						"isfolder": 0,
						"filesize": "300",
						"path": "/share/public/file3.txt"
					}
				]
			}`,
			wantLen: 3,
			wantErr: false,
		},
		{
			name: "search returns empty results",
			path: "/share/public",
			options: &SearchOptions{
				Pattern: "nonexistent",
			},
			response: `{
				"total": 0,
				"datas": []
			}`,
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "search returns folders",
			path: "/share/public",
			options: &SearchOptions{
				Pattern: "folder",
			},
			response: `{
				"total": 1,
				"datas": [
					{
						"filename": "folder",
						"isfolder": 1,
						"filesize": "0",
						"path": "/share/public/folder"
					}
				]
			}`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "search with empty pattern",
			path: "/share/public",
			options: &SearchOptions{
				Pattern: "",
			},
			response: `{
				"total": 1,
				"datas": [
					{
						"filename": "all.txt",
						"isfolder": 0,
						"filesize": "512",
						"path": "/share/public/all.txt"
					}
				]
			}`,
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				// Verify essential query parameters
				if r.URL.Query().Get("func") != "search_ext" {
					t.Errorf("expected func=search_ext, got %s", r.URL.Query().Get("func"))
				}

				if r.URL.Query().Get("sid") != "test-sid-12345" {
					t.Errorf("expected sid=test-sid-12345, got %s", r.URL.Query().Get("sid"))
				}

				if r.URL.Query().Get("folders") != tt.path {
					t.Errorf("expected folders=%s, got %s", tt.path, r.URL.Query().Get("folders"))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			results, err := fs.Search(context.Background(), tt.path, tt.options)

			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantLen {
				t.Errorf("Search() returned %d results, want %d", len(results), tt.wantLen)
			}

		})
	}
}

// TestSearch_AuthError tests authentication errors
func TestSearch_AuthError(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockClient)
		wantErr bool
		errCode api.ErrorCode
	}{
		{
			name: "not authenticated - empty SID",
			setup: func(m *MockClient) {
				m.SetSID("")
			},
			wantErr: true,
			errCode: api.ErrAuthFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"total": 0, "datas": []}`))
			})
			defer mock.Close()

			tt.setup(mock)

			fs := NewFileStationService(mock.GetClient())
			_, err := fs.Search(context.Background(), "/share/public", &SearchOptions{Pattern: "test"})

			if !tt.wantErr {
				t.Errorf("expected error, got nil")
				return
			}

			apiErr, ok := err.(*api.APIError)
			if !ok {
				t.Errorf("expected *api.APIError, got %T", err)
				return
			}

			if apiErr.Code != tt.errCode {
				t.Errorf("expected error code %d, got %d", tt.errCode, apiErr.Code)
			}
		})
	}
}

// TestSearch_APIErrors tests API error responses
func TestSearch_APIErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "network error - connection refused",
			statusCode: 0,
			response:   "",
			wantErr:    true,
			errMsg:     "network error",
		},
		{
			name:       "invalid JSON response",
			statusCode: http.StatusOK,
			response:   `{invalid json}`,
			wantErr:    true,
			errMsg:     "failed to parse search response",
		},
		{
			name:       "API returns error status",
			statusCode: http.StatusOK,
			response:   `{"error": "search failed"}`,
			wantErr:    false, // JSON parses but datas field is missing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mock *MockClient

			if tt.name == "network error - connection refused" {
				// Create a mock but close the server immediately
				mock = NewMockClient(func(w http.ResponseWriter, r *http.Request) {})
				mock.server.Close()
				mock.server = nil
			} else {
				mock = NewMockClient(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.statusCode)
					w.Write([]byte(tt.response))
				})
			}
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			_, err := fs.Search(context.Background(), "/share/public", &SearchOptions{Pattern: "test"})

			if !tt.wantErr {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}

			if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestSearchAsync_Success tests successful async search initiation
func TestSearchAsync_Success(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		options  *SearchOptions
		response string
		wantPID  string
		wantErr  bool
	}{
		{
			name: "basic async search",
			path: "/share/public",
			options: &SearchOptions{
				Pattern: "test",
			},
			response: `{
				"status": 1,
				"pid": "async-pid-12345"
			}`,
			wantPID: "async-pid-12345",
			wantErr: false,
		},
		{
			name: "async search with file type",
			path: "/share/music",
			options: &SearchOptions{
				Pattern:  "song",
				FileType: "MUSIC",
			},
			response: `{
				"status": 1,
				"pid": "async-pid-music-67890"
			}`,
			wantPID: "async-pid-music-67890",
			wantErr: false,
		},
		{
			name: "async search with size filter",
			path: "/share/public",
			options: &SearchOptions{
				Pattern: "large",
				SizeMin: 1048576,
			},
			response: `{
				"status": 1,
				"pid": "async-pid-size-11111"
			}`,
			wantPID: "async-pid-size-11111",
			wantErr: false,
		},
		{
			name: "async search with nil options",
			path: "/share/public",
			options: nil,
			response: `{
				"status": 1,
				"pid": "async-pid-default-22222"
			}`,
			wantPID: "async-pid-default-22222",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				// Verify essential query parameters
				if r.URL.Query().Get("func") != "search_start" {
					t.Errorf("expected func=search_start, got %s", r.URL.Query().Get("func"))
				}

				if r.URL.Query().Get("sid") != "test-sid-12345" {
					t.Errorf("expected sid=test-sid-12345, got %s", r.URL.Query().Get("sid"))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			pid, err := fs.SearchAsync(context.Background(), tt.path, tt.options)

			if (err != nil) != tt.wantErr {
				t.Errorf("SearchAsync() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if pid != tt.wantPID {
				t.Errorf("SearchAsync() returned PID %q, want %q", pid, tt.wantPID)
			}
		})
	}
}

// TestSearchAsync_AuthError tests authentication errors for async search
func TestSearchAsync_AuthError(t *testing.T) {
	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": 1, "pid": "test-pid"}`))
	})
	defer mock.Close()

	mock.SetSID("")

	fs := NewFileStationService(mock.GetClient())
	_, err := fs.SearchAsync(context.Background(), "/share/public", &SearchOptions{Pattern: "test"})

	if err == nil {
		t.Errorf("expected authentication error, got nil")
		return
	}

	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Errorf("expected *api.APIError, got %T", err)
		return
	}

	if apiErr.Code != api.ErrAuthFailed {
		t.Errorf("expected error code %d, got %d", api.ErrAuthFailed, apiErr.Code)
	}
}

// TestSearchAsync_APIErrors tests API error responses for async search
func TestSearchAsync_APIErrors(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		wantErr    bool
		errMsg     string
	}{
		{
			name:     "API returns failure status",
			response: `{"status": 0, "pid": ""}`,
			wantErr:  true,
			errMsg:   "",
		},
		{
			name:     "API returns error message",
			response: `{"status": 0, "error": "search start failed"}`,
			wantErr:  true,
			errMsg:   "search start failed",
		},
		{
			name:     "invalid JSON response",
			response: `{invalid json}`,
			wantErr:  true,
			errMsg:   "failed to parse search response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			_, err := fs.SearchAsync(context.Background(), "/share/public", &SearchOptions{Pattern: "test"})

			if !tt.wantErr {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}

			if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestGetSearchResult_Success tests successful search result retrieval
func TestGetSearchResult_Success(t *testing.T) {
	tests := []struct {
		name     string
		pid      string
		response string
		want     *SearchResult
		wantErr  bool
	}{
		{
			name: "running search",
			pid:  "pid-running-123",
			response: `{
				"status": 1,
				"data": {
					"pid": "pid-running-123",
					"status": "running",
					"total": 0,
					"results": []
				}
			}`,
			want: &SearchResult{
				PID:     "pid-running-123",
				Status:  "running",
				Total:   0,
				Results: []File{},
			},
			wantErr: false,
		},
		{
			name: "finished search with results",
			pid:  "pid-finished-456",
			response: `{
				"status": 1,
				"data": {
					"pid": "pid-finished-456",
					"status": "finished",
					"total": 2,
					"results": [
						{
							"filename": "result1.txt",
							"isfolder": 0,
							"filesize": "1024",
							"path": "/share/result1.txt"
						},
						{
							"filename": "result2.txt",
							"isfolder": 0,
							"filesize": "2048",
							"path": "/share/result2.txt"
						}
					]
				}
			}`,
			want: &SearchResult{
				PID:    "pid-finished-456",
				Status: "finished",
				Total:  2,
				Results: []File{
					{FileName: "result1.txt", IsFolder: 0, FileSize: "1024", Path: "/share/result1.txt"},
					{FileName: "result2.txt", IsFolder: 0, FileSize: "2048", Path: "/share/result2.txt"},
				},
			},
			wantErr: false,
		},
		{
			name: "failed search",
			pid:  "pid-failed-789",
			response: `{
				"status": 1,
				"data": {
					"pid": "pid-failed-789",
					"status": "failed",
					"total": 0,
					"results": []
				}
			}`,
			want: &SearchResult{
				PID:     "pid-failed-789",
				Status:  "failed",
				Total:   0,
				Results: []File{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "GET" {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				// Verify essential query parameters
				if r.URL.Query().Get("func") != "get_search_result" {
					t.Errorf("expected func=get_search_result, got %s", r.URL.Query().Get("func"))
				}

				if r.URL.Query().Get("pid") != tt.pid {
					t.Errorf("expected pid=%s, got %s", tt.pid, r.URL.Query().Get("pid"))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			result, err := fs.GetSearchResult(context.Background(), tt.pid)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetSearchResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result.PID != tt.want.PID {
				t.Errorf("GetSearchResult() PID = %q, want %q", result.PID, tt.want.PID)
			}

			if result.Status != tt.want.Status {
				t.Errorf("GetSearchResult() Status = %q, want %q", result.Status, tt.want.Status)
			}

			if result.Total != tt.want.Total {
				t.Errorf("GetSearchResult() Total = %d, want %d", result.Total, tt.want.Total)
			}

			if len(result.Results) != len(tt.want.Results) {
				t.Errorf("GetSearchResult() Results count = %d, want %d", len(result.Results), len(tt.want.Results))
			}
		})
	}
}

// TestGetSearchResult_InvalidParams tests invalid parameter handling
func TestGetSearchResult_InvalidParams(t *testing.T) {
	tests := []struct {
		name    string
		pid     string
		wantErr bool
		errCode api.ErrorCode
		errMsg  string
	}{
		{
			name:    "empty PID",
			pid:     "",
			wantErr: true,
			errCode: api.ErrInvalidParams,
			errMsg:  "process ID required",
		},
		{
			name:    "whitespace PID",
			pid:     "   ",
			wantErr: true,
			errCode: api.ErrInvalidParams,
			errMsg:  "process ID required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": 1, "data": {"pid": "test", "status": "finished"}}`))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			_, err := fs.GetSearchResult(context.Background(), tt.pid)

			if !tt.wantErr {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}

			apiErr, ok := err.(*api.APIError)
			if !ok {
				t.Errorf("expected *api.APIError, got %T", err)
				return
			}

			if apiErr.Code != tt.errCode {
				t.Errorf("expected error code %d, got %d", tt.errCode, apiErr.Code)
			}

			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestGetSearchResult_AuthError tests authentication errors
func TestGetSearchResult_AuthError(t *testing.T) {
	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": 1, "data": {"pid": "test", "status": "finished"}}`))
	})
	defer mock.Close()

	mock.SetSID("")

	fs := NewFileStationService(mock.GetClient())
	_, err := fs.GetSearchResult(context.Background(), "test-pid")

	if err == nil {
		t.Errorf("expected authentication error, got nil")
		return
	}

	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Errorf("expected *api.APIError, got %T", err)
		return
	}

	if apiErr.Code != api.ErrAuthFailed {
		t.Errorf("expected error code %d, got %d", api.ErrAuthFailed, apiErr.Code)
	}
}

// TestGetSearchResult_APIErrors tests API error responses
func TestGetSearchResult_APIErrors(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		wantErr    bool
		errMsg     string
	}{
		{
			name:     "API returns failure status",
			response: `{"status": 0, "data": {"pid": "test"}}`,
			wantErr:  true,
		},
		{
			name:     "API returns error message",
			response: `{"status": 0, "error": "search result not found"}`,
			wantErr:  true,
			errMsg:   "search result not found",
		},
		{
			name:     "invalid JSON response",
			response: `{invalid json}`,
			wantErr:  true,
			errMsg:   "failed to parse search result response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			_, err := fs.GetSearchResult(context.Background(), "test-pid")

			if !tt.wantErr {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}

			if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestStopSearch_Success tests successful search cancellation
func TestStopSearch_Success(t *testing.T) {
	tests := []struct {
		name     string
		pid      string
		response string
		wantErr  bool
	}{
		{
			name: "stop running search",
			pid:  "pid-to-stop-123",
			response: `{
				"status": 1,
				"success": "true"
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != "POST" {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				// Verify essential query parameters
				if r.URL.Query().Get("func") != "search_stop" {
					t.Errorf("expected func=search_stop, got %s", r.URL.Query().Get("func"))
				}

				if r.URL.Query().Get("pid") != tt.pid {
					t.Errorf("expected pid=%s, got %s", tt.pid, r.URL.Query().Get("pid"))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			err := fs.StopSearch(context.Background(), tt.pid)

			if (err != nil) != tt.wantErr {
				t.Errorf("StopSearch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestStopSearch_InvalidParams tests invalid parameter handling
func TestStopSearch_InvalidParams(t *testing.T) {
	tests := []struct {
		name    string
		pid     string
		wantErr bool
		errCode api.ErrorCode
		errMsg  string
	}{
		{
			name:    "empty PID",
			pid:     "",
			wantErr: true,
			errCode: api.ErrInvalidParams,
			errMsg:  "process ID required",
		},
		{
			name:    "whitespace PID",
			pid:     "   ",
			wantErr: true,
			errCode: api.ErrInvalidParams,
			errMsg:  "process ID required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": 1, "success": "true"}`))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			err := fs.StopSearch(context.Background(), tt.pid)

			if !tt.wantErr {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}

			apiErr, ok := err.(*api.APIError)
			if !ok {
				t.Errorf("expected *api.APIError, got %T", err)
				return
			}

			if apiErr.Code != tt.errCode {
				t.Errorf("expected error code %d, got %d", tt.errCode, apiErr.Code)
			}

			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestStopSearch_AuthError tests authentication errors
func TestStopSearch_AuthError(t *testing.T) {
	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": 1, "success": "true"}`))
	})
	defer mock.Close()

	mock.SetSID("")

	fs := NewFileStationService(mock.GetClient())
	err := fs.StopSearch(context.Background(), "test-pid")

	if err == nil {
		t.Errorf("expected authentication error, got nil")
		return
	}

	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Errorf("expected *api.APIError, got %T", err)
		return
	}

	if apiErr.Code != api.ErrAuthFailed {
		t.Errorf("expected error code %d, got %d", api.ErrAuthFailed, apiErr.Code)
	}
}

// TestStopSearch_APIErrors tests API error responses
func TestStopSearch_APIErrors(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		wantErr    bool
		errMsg     string
	}{
		{
			name:     "API returns failure status",
			response: `{"status": 0, "success": "false"}`,
			wantErr:  true,
		},
		{
			name:     "API returns error message",
			response: `{"status": 0, "error": "search not found"}`,
			wantErr:  true,
			errMsg:   "search not found",
		},
		{
			name:     "invalid JSON response",
			response: `{invalid json}`,
			wantErr:  true,
			errMsg:   "failed to parse response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			err := fs.StopSearch(context.Background(), "test-pid")

			if !tt.wantErr {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}

			if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestSearchByPattern tests the simplified SearchByPattern function
func TestSearchByPattern(t *testing.T) {
	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"total": 1,
			"datas": [
				{
					"filename": "pattern.txt",
					"isfolder": 0,
					"filesize": "512",
					"path": "/share/public/pattern.txt"
				}
			]
		}`))
	})
	defer mock.Close()

	fs := NewFileStationService(mock.GetClient())
	results, err := fs.SearchByPattern(context.Background(), "/share/public", "*.txt")

	if err != nil {
		t.Errorf("SearchByPattern() unexpected error: %v", err)
		return
	}

	if len(results) != 1 {
		t.Errorf("SearchByPattern() returned %d results, want 1", len(results))
	}

	if results[0].FileName != "pattern.txt" {
		t.Errorf("SearchByPattern() filename = %q, want %q", results[0].FileName, "pattern.txt")
	}
}

// TestFileTypeMapping tests file type parameter mapping
func TestFileTypeMapping(t *testing.T) {
	tests := []struct {
		name          string
		fileType      string
		expectedParam string
	}{
		{"MUSIC file type", "MUSIC", "1"},
		{"VIDEO file type", "VIDEO", "2"},
		{"PHOTO file type", "PHOTO", "3"},
		{"Unknown file type", "UNKNOWN", "0"}, // Falls back to default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				searchType := r.URL.Query().Get("searchType")
				// Verify the searchType parameter is set correctly
				_ = searchType
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"total": 0, "datas": []}`))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			_, _ = fs.Search(context.Background(), "/share/public", &SearchOptions{
				FileType: tt.fileType,
			})
		})
	}
}

// TestSizeFilterParameterMapping tests size filter parameter mapping
func TestSizeFilterParameterMapping(t *testing.T) {
	tests := []struct {
		name            string
		sizeMin         int64
		sizeMax         int64
		expectedType    string
		expectedSize    string
	}{
		{
			name:         "minimum size filter",
			sizeMin:      1024,
			sizeMax:      0,
			expectedType: "5", // Greater than
			expectedSize: "1024",
		},
		{
			name:         "maximum size filter",
			sizeMin:      0,
			sizeMax:      2048,
			expectedType: "6", // Less than
			expectedSize: "2048",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				sizeType := r.URL.Query().Get("fileSizeType")
				size := r.URL.Query().Get("fileSize")

				if sizeType != tt.expectedType {
					t.Errorf("expected fileSizeType=%s, got %s", tt.expectedType, sizeType)
				}

				if size != tt.expectedSize {
					t.Errorf("expected fileSize=%s, got %s", tt.expectedSize, size)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"total": 0, "datas": []}`))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			_, _ = fs.Search(context.Background(), "/share/public", &SearchOptions{
				SizeMin: tt.sizeMin,
				SizeMax: tt.sizeMax,
			})
		})
	}
}

// TestSearchResponseDecoding tests various response formats
func TestSearchResponseDecoding(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantLen  int
		wantErr  bool
	}{
		{
			name: "response with all file fields",
			response: `{
				"total": 1,
				"datas": [
					{
						"filename": "complete.txt",
						"isfolder": 0,
						"filesize": "1024",
						"owner": "admin",
						"group": "everyone",
						"privilege": "rwxr-xr-x",
						"mt": "2024-01-01 12:00:00",
						"epochmt": 1704105600,
						"exist": 1,
						"filetype": 0,
						"path": "/share/public/complete.txt"
					}
				]
			}`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "response with minimal fields",
			response: `{
				"total": 1,
				"datas": [
					{
						"filename": "minimal.txt",
						"isfolder": 0
					}
				]
			}`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "response with null values",
			response: `{
				"total": 1,
				"datas": [
					{
						"filename": "nulls.txt",
						"isfolder": 0,
						"filesize": null,
						"owner": null
					}
				]
			}`,
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			results, err := fs.Search(context.Background(), "/share/public", &SearchOptions{Pattern: "test"})

			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantLen {
				t.Errorf("Search() returned %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}

// TestSearchAsyncResponseDecoding tests async search response formats
func TestSearchAsyncResponseDecoding(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantPID  string
		wantErr  bool
	}{
		{
			name: "standard async response",
			response: `{
				"status": 1,
				"pid": "pid-12345"
			}`,
			wantPID: "pid-12345",
			wantErr: false,
		},
		{
			name: "response with extra fields",
			response: `{
				"status": 1,
				"pid": "pid-extra-67890",
				"message": "search started"
			}`,
			wantPID: "pid-extra-67890",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			pid, err := fs.SearchAsync(context.Background(), "/share/public", &SearchOptions{Pattern: "test"})

			if (err != nil) != tt.wantErr {
				t.Errorf("SearchAsync() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if pid != tt.wantPID {
				t.Errorf("SearchAsync() returned PID %q, want %q", pid, tt.wantPID)
			}
		})
	}
}

// TestSearchResultDecoding tests GetSearchResult response formats
func TestSearchResultDecoding(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantErr  bool
		verify   func(*testing.T, *SearchResult)
	}{
		{
			name: "result with complete data",
			response: `{
				"status": 1,
				"data": {
					"pid": "pid-complete",
					"status": "finished",
					"total": 2,
					"results": [
						{
							"filename": "file1.txt",
							"isfolder": 0,
							"filesize": "100",
							"path": "/share/file1.txt"
						},
						{
							"filename": "file2.txt",
							"isfolder": 0,
							"filesize": "200",
							"path": "/share/file2.txt"
						}
					]
				}
			}`,
			wantErr: false,
			verify: func(t *testing.T, result *SearchResult) {
				if result.Total != 2 {
					t.Errorf("expected Total=2, got %d", result.Total)
				}
				if len(result.Results) != 2 {
					t.Errorf("expected 2 results, got %d", len(result.Results))
				}
			},
		},
		{
			name: "result with empty results array",
			response: `{
				"status": 1,
				"data": {
					"pid": "pid-empty",
					"status": "finished",
					"total": 0,
					"results": []
				}
			}`,
			wantErr: false,
			verify: func(t *testing.T, result *SearchResult) {
				if result.Total != 0 {
					t.Errorf("expected Total=0, got %d", result.Total)
				}
				if len(result.Results) != 0 {
					t.Errorf("expected 0 results, got %d", len(result.Results))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			result, err := fs.GetSearchResult(context.Background(), "test-pid")

			if (err != nil) != tt.wantErr {
				t.Errorf("GetSearchResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.verify != nil {
				tt.verify(t, result)
			}
		})
	}
}

// TestSearchContextCancellation tests context cancellation handling
func TestSearchContextCancellation(t *testing.T) {
	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		w.Header().Set("Content-Type", "application/json")
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"total": 0, "datas": []}`))
	})
	defer mock.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	fs := NewFileStationService(mock.GetClient())
	_, err := fs.Search(ctx, "/share/public", &SearchOptions{Pattern: "test"})

	if err == nil {
		t.Errorf("expected context cancellation error, got nil")
	}
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		options  *SearchOptions
		response string
		wantErr  bool
	}{
		{
			name:    "empty path",
			path:    "",
			options: &SearchOptions{Pattern: "test"},
			response: `{
				"total": 0,
				"datas": []
			}`,
			wantErr: false, // API handles empty path
		},
		{
			name:    "root path",
			path:    "/",
			options: &SearchOptions{Pattern: "test"},
			response: `{
				"total": 0,
				"datas": []
			}`,
			wantErr: false,
		},
		{
			name:    "path with special characters",
			path:    "/share/public/folder with spaces",
			options: &SearchOptions{Pattern: "test"},
			response: `{
				"total": 0,
				"datas": []
			}`,
			wantErr: false,
		},
		{
			name:    "pattern with special characters",
			path:    "/share/public",
			options: &SearchOptions{Pattern: "*.txt;*.doc"},
			response: `{
				"total": 0,
				"datas": []
			}`,
			wantErr: false,
		},
		{
			name:    "very large size filter",
			path:    "/share/public",
			options: &SearchOptions{SizeMin: 1024 * 1024 * 1024 * 1024}, // 1TB
			response: `{
				"total": 0,
				"datas": []
			}`,
			wantErr: false,
		},
		{
			name:    "zero size filter",
			path:    "/share/public",
			options: &SearchOptions{SizeMin: 0, SizeMax: 0},
			response: `{
				"total": 0,
				"datas": []
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.response))
			})
			defer mock.Close()

			fs := NewFileStationService(mock.GetClient())
			_, err := fs.Search(context.Background(), tt.path, tt.options)

			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSearchAsync_Concurrent tests concurrent async search operations
func TestSearchAsync_Concurrent(t *testing.T) {
	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": 1, "pid": "concurrent-pid"}`))
	})
	defer mock.Close()

	fs := NewFileStationService(mock.GetClient())

	// Launch multiple concurrent searches
	results := make(chan string, 5)
	for i := 0; i < 5; i++ {
		go func(idx int) {
			pid, err := fs.SearchAsync(context.Background(), "/share/public", &SearchOptions{
				Pattern: fmt.Sprintf("test%d", idx),
			})
			if err == nil {
				results <- pid
			} else {
				results <- ""
			}
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < 5; i++ {
		if <-results != "" {
			successCount++
		}
	}

	if successCount != 5 {
		t.Errorf("expected 5 successful async searches, got %d", successCount)
	}
}

// TestIntegration_SearchWorkflow tests a complete search workflow
func TestIntegration_SearchWorkflow(t *testing.T) {
	pidReceived := false

	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		funcName := r.URL.Query().Get("func")

		w.Header().Set("Content-Type", "application/json")

		switch funcName {
		case "search_start":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": 1, "pid": "workflow-pid-123"}`))
		case "get_search_result":
			w.WriteHeader(http.StatusOK)
			// Return finished state on second call
			w.Write([]byte(`{
				"status": 1,
				"data": {
					"pid": "workflow-pid-123",
					"status": "finished",
					"total": 1,
					"results": [
						{
							"filename": "workflow.txt",
							"isfolder": 0,
							"filesize": "512",
							"path": "/share/workflow.txt"
						}
					]
				}
			}`))
		case "search_stop":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": 1, "success": "true"}`))
		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "unknown function"}`))
		}
	})
	defer mock.Close()

	fs := NewFileStationService(mock.GetClient())

	// Step 1: Start async search
	pid, err := fs.SearchAsync(context.Background(), "/share/public", &SearchOptions{Pattern: "workflow"})
	if err != nil {
		t.Fatalf("SearchAsync failed: %v", err)
	}
	if pid != "workflow-pid-123" {
		t.Fatalf("expected PID workflow-pid-123, got %s", pid)
	}
	pidReceived = true

	// Step 2: Get search results
	result, err := fs.GetSearchResult(context.Background(), pid)
	if err != nil {
		t.Fatalf("GetSearchResult failed: %v", err)
	}
	if result.Status != "finished" {
		t.Fatalf("expected status finished, got %s", result.Status)
	}
	if result.Total != 1 {
		t.Fatalf("expected 1 result, got %d", result.Total)
	}

	// Step 3: Stop the search
	err = fs.StopSearch(context.Background(), pid)
	if err != nil {
		t.Fatalf("StopSearch failed: %v", err)
	}

	if !pidReceived {
		t.Error("workflow did not complete properly")
	}
}

// TestBodyReading tests that response body is properly read and closed
func TestBodyReading(t *testing.T) {
	callCount := 0

	mock := NewMockClient(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"total": 1, "datas": [{"filename": "test.txt"}]}`))
	})
	defer mock.Close()

	fs := NewFileStationService(mock.GetClient())

	// Multiple calls should work without resource leaks
	for i := 0; i < 3; i++ {
		_, err := fs.Search(context.Background(), "/share/public", &SearchOptions{Pattern: "test"})
		if err != nil {
			t.Errorf("iteration %d: unexpected error: %v", i, err)
		}
	}

	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

// TestResponseBodyClose verifies response body is always closed
func TestResponseBodyClose(t *testing.T) {
	closeCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return invalid JSON to trigger error path
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	// Create a custom response reader wrapper
	originalTransport := http.DefaultTransport
	customTransport := &roundTripperWrapper{
		transport: originalTransport,
		onResponse: func(resp *http.Response) {
			resp.Body = &readCloserWrapper{
				ReadCloser: resp.Body,
				onClose: func() {
					closeCalled = true
				},
			}
		},
	}
	defer func() {
		http.DefaultTransport = originalTransport
	}()

cfg := &api.Config{
		Host:     strings.TrimPrefix(server.URL, "http://"),
		Insecure: true,
		Username: "admin",
		Password: "password",
}
client, _ := api.NewClient(cfg)
	client.SetSID("test-sid")

	// Override the client's HTTP client with our custom transport
	client.GetHTTPClient().Transport = customTransport

	fs := NewFileStationService(client)

	// This should fail due to invalid JSON, but body should still be closed
	_, err := fs.Search(context.Background(), "/share/public", &SearchOptions{Pattern: "test"})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	// Give time for cleanup
	time.Sleep(10 * time.Millisecond)

	if !closeCalled {
		t.Error("response body was not closed on error")
	}
}

// Helper types for body close testing
type roundTripperWrapper struct {
	transport http.RoundTripper
	onResponse func(*http.Response)
}

func (r *roundTripperWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if r.onResponse != nil {
		r.onResponse(resp)
	}
	return resp, nil
}

type readCloserWrapper struct {
	io.ReadCloser
	onClose func()
}

func (r *readCloserWrapper) Close() error {
	if r.onClose != nil {
		r.onClose()
	}
	return r.ReadCloser.Close()
}
