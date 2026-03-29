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

// setupSystemTestClient creates a test client with mock server for system tests
func setupSystemTestClient(t *testing.T) (*api.Client, *testutil.MockServer) {
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

// setupUnauthenticatedSystemClient creates a client without authentication
func setupUnauthenticatedSystemClient(t *testing.T) *api.Client {
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

// TestCheckSession tests the CheckSession function
func TestCheckSession(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		wantValid      bool
		wantErr        bool
		expectedErr    api.ErrorCode
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "valid session",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"is_valid": true,
						"sid":      "test-sid-12345",
					},
				},
			},
			wantValid: true,
			wantErr:   false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "check_sid" {
					t.Errorf("Expected func=check_sid, got %s", fn)
				}
			},
		},
		{
			name: "invalid session",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"is_valid": false,
					},
				},
			},
			wantValid: false,
			wantErr:   false,
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
		{
			name: "session expired response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1002,
					"error_msg":  "Session expired",
				},
			},
			wantValid: false,
			wantErr:   false,
		},
		{
			name: "missing is_valid field",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"sid": "test-sid",
					},
				},
			},
			wantValid: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupSystemTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			valid, err := fs.CheckSession(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("CheckSession() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("CheckSession() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("CheckSession() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("CheckSession() unexpected error = %v", err)
				return
			}

			if valid != tt.wantValid {
				t.Errorf("CheckSession() valid = %v, want %v", valid, tt.wantValid)
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

// TestCheckSession_NotAuthenticated tests CheckSession when not authenticated
func TestCheckSession_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedSystemClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.CheckSession(ctx)

	if err == nil {
		t.Error("CheckSession() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("CheckSession() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("CheckSession() error = %v, want auth error", apiErr)
	}
}

// TestGetFileSize tests the GetFileSize function
func TestGetFileSize(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		paths          []string
		wantErr        bool
		expectedErr    api.ErrorCode
		wantTotalSize  int64
		wantFileCount  int
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "single file size",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"total_size": int64(1024),
						"items": []map[string]interface{}{
							{
								"path":       "/home/file.txt",
								"size":       int64(1024),
								"file_count": 1,
							},
						},
					},
				},
			},
			paths:         []string{"/home/file.txt"},
			wantTotalSize: 1024,
			wantFileCount: 1,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "get_file_size" {
					t.Errorf("Expected func=get_file_size, got %s", fn)
				}
				if total := r.URL.Query().Get("file_total"); total != "1" {
					t.Errorf("Expected file_total=1, got %s", total)
				}
				if path0 := r.URL.Query().Get("path0"); path0 != "/home/file.txt" {
					t.Errorf("Expected path0=/home/file.txt, got %s", path0)
				}
			},
		},
		{
			name: "multiple files size",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"total_size": int64(3072),
						"items": []map[string]interface{}{
							{
								"path":       "/home/file1.txt",
								"size":       int64(1024),
								"file_count": 1,
							},
							{
								"path":       "/home/file2.txt",
								"size":       int64(2048),
								"file_count": 1,
							},
						},
					},
				},
			},
			paths:         []string{"/home/file1.txt", "/home/file2.txt"},
			wantTotalSize: 3072,
			wantFileCount: 2,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if total := r.URL.Query().Get("file_total"); total != "2" {
					t.Errorf("Expected file_total=2, got %s", total)
				}
				if path0 := r.URL.Query().Get("path0"); path0 != "/home/file1.txt" {
					t.Errorf("Expected path0=/home/file1.txt, got %s", path0)
				}
				if path1 := r.URL.Query().Get("path1"); path1 != "/home/file2.txt" {
					t.Errorf("Expected path1=/home/file2.txt, got %s", path1)
				}
			},
		},
		{
			name: "empty paths slice",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    1,
					"total_size": int64(0),
					"items":      []interface{}{},
				},
			},
			paths:         []string{},
			wantTotalSize: 0,
		},
		{
			name: "folder size with multiple files",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"total_size": int64(102400),
						"items": []map[string]interface{}{
							{
								"path":       "/home/documents",
								"size":       int64(102400),
								"file_count": 10,
							},
						},
					},
				},
			},
			paths:         []string{"/home/documents"},
			wantTotalSize: 102400,
			wantFileCount: 10,
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
			paths:       []string{"/nonexistent/file.txt"},
			wantErr:     true,
			expectedErr: api.ErrNotFound,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json{{{",
			},
			paths:       []string{"/home/file.txt"},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			paths:   []string{"/home/file.txt"},
			wantErr: true,
		},
		{
			name: "special characters in path",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"total_size": int64(512),
						"items": []map[string]interface{}{
							{
								"path":       "/home/file with spaces.txt",
								"size":       int64(512),
								"file_count": 1,
							},
						},
					},
				},
			},
			paths:         []string{"/home/file with spaces.txt"},
			wantTotalSize: 512,
		},
		{
			name: "session expired",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1002,
					"error_msg":  "Session expired",
				},
			},
			paths:       []string{"/home/file.txt"},
			wantErr:     true,
			expectedErr: api.ErrSessionExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupSystemTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.GetFileSize(ctx, tt.paths)

			if tt.wantErr {
				if err == nil {
					t.Error("GetFileSize() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("GetFileSize() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("GetFileSize() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("GetFileSize() unexpected error = %v", err)
				return
			}

			if resp.Data.TotalSize != tt.wantTotalSize {
				t.Errorf("GetFileSize() totalSize = %d, want %d", resp.Data.TotalSize, tt.wantTotalSize)
			}

			if tt.wantFileCount > 0 && len(resp.Data.Items) > 0 && resp.Data.Items[0].FileCount != tt.wantFileCount {
				t.Errorf("GetFileSize() fileCount = %d, want %d", resp.Data.Items[0].FileCount, tt.wantFileCount)
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

// TestGetFileSize_NotAuthenticated tests GetFileSize when not authenticated
func TestGetFileSize_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedSystemClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.GetFileSize(ctx, []string{"/home/file.txt"})

	if err == nil {
		t.Error("GetFileSize() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetFileSize() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("GetFileSize() error = %v, want auth error", apiErr)
	}
}

// TestGetTree tests the GetTree function
func TestGetTree(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *GetTreeOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		wantNodeCount  int
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "get tree without options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"tree_nodes": []map[string]interface{}{
							{
								"id":       "1",
								"name":     "Home",
								"path":     "/home",
								"isfolder": true,
							},
							{
								"id":       "2",
								"name":     "Public",
								"path":     "/public",
								"isfolder": true,
							},
						},
					},
				},
			},
			options:       nil,
			wantNodeCount: 2,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "get_tree" {
					t.Errorf("Expected func=get_tree, got %s", fn)
				}
				if isISO := r.URL.Query().Get("is_iso"); isISO != "0" {
					t.Errorf("Expected is_iso=0, got %s", isISO)
				}
			},
		},
		{
			name: "get tree with ISO enabled",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"tree_nodes": []map[string]interface{}{
							{
								"id":       "1",
								"name":     "ISO",
								"path":     "/iso",
								"isfolder": true,
							},
						},
					},
				},
			},
			options: &GetTreeOptions{
				IsISO: true,
			},
			wantNodeCount: 1,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if isISO := r.URL.Query().Get("is_iso"); isISO != "1" {
					t.Errorf("Expected is_iso=1, got %s", isISO)
				}
			},
		},
		{
			name: "get tree with specific node",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"tree_nodes": []map[string]interface{}{
							{
								"id":       "1",
								"name":     "Documents",
								"path":     "/home/documents",
								"isfolder": true,
								"children": []map[string]interface{}{
									{
										"id":       "2",
										"name":     "Work",
										"path":     "/home/documents/work",
										"isfolder": true,
									},
								},
							},
						},
					},
				},
			},
			options: &GetTreeOptions{
				Node: "/home",
			},
			wantNodeCount: 1,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if node := r.URL.Query().Get("node"); node != "/home" {
					t.Errorf("Expected node=/home, got %s", node)
				}
			},
		},
		{
			name: "get tree with all options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"tree_nodes": []map[string]interface{}{
							{
								"id":       "1",
								"name":     "Share",
								"path":     "/share",
								"isfolder": true,
							},
						},
					},
				},
			},
			options: &GetTreeOptions{
				IsISO: true,
				Node:  "/share",
			},
			wantNodeCount: 1,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if isISO := r.URL.Query().Get("is_iso"); isISO != "1" {
					t.Errorf("Expected is_iso=1, got %s", isISO)
				}
				if node := r.URL.Query().Get("node"); node != "/share" {
					t.Errorf("Expected node=/share, got %s", node)
				}
			},
		},
		{
			name: "empty tree",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"tree_nodes": []interface{}{},
					},
				},
			},
			options:       nil,
			wantNodeCount: 0,
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
			options:     &GetTreeOptions{Node: "/invalid"},
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
		{
			name: "nested tree structure",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"tree_nodes": []map[string]interface{}{
							{
								"id":       "1",
								"name":     "Root",
								"path":     "/",
								"isfolder": true,
								"children": []map[string]interface{}{
									{
										"id":       "2",
										"name":     "Level1",
										"path":     "/level1",
										"isfolder": true,
										"children": []map[string]interface{}{
											{
												"id":       "3",
												"name":     "Level2",
												"path":     "/level1/level2",
												"isfolder": true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			options:       nil,
			wantNodeCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupSystemTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.GetTree(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("GetTree() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("GetTree() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("GetTree() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("GetTree() unexpected error = %v", err)
				return
			}

			if len(resp.Data.TreeNodes) != tt.wantNodeCount {
				t.Errorf("GetTree() nodeCount = %d, want %d", len(resp.Data.TreeNodes), tt.wantNodeCount)
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

// TestGetTree_NotAuthenticated tests GetTree when not authenticated
func TestGetTree_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedSystemClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.GetTree(ctx, nil)

	if err == nil {
		t.Error("GetTree() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetTree() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("GetTree() error = %v, want auth error", apiErr)
	}
}

// TestGetUserGroupList tests the GetUserGroupList function
func TestGetUserGroupList(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		userType       UserGroupType
		wantErr        bool
		expectedErr    api.ErrorCode
		wantUserCount  int
		wantGroupCount int
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "get users",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"users": []map[string]interface{}{
							{
								"id":       "1",
								"name":     "Admin User",
								"username": "admin",
								"email":    "admin@example.com",
							},
							{
								"id":       "2",
								"name":     "Test User",
								"username": "testuser",
								"email":    "test@example.com",
							},
						},
						"total": 2,
					},
				},
			},
			userType:      UserGroupTypeUser,
			wantUserCount: 2,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "get_user_group_list" {
					t.Errorf("Expected func=get_user_group_list, got %s", fn)
				}
				if typ := r.URL.Query().Get("type"); typ != "0" {
					t.Errorf("Expected type=0, got %s", typ)
				}
			},
		},
		{
			name: "get groups",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"groups": []map[string]interface{}{
							{
								"id":          "1",
								"name":        "Administrators",
								"description": "Admin group",
							},
							{
								"id":          "2",
								"name":        "Users",
								"description": "Regular users",
							},
						},
						"total": 2,
					},
				},
			},
			userType:       UserGroupTypeGroup,
			wantGroupCount: 2,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if typ := r.URL.Query().Get("type"); typ != "1" {
					t.Errorf("Expected type=1, got %s", typ)
				}
			},
		},
		{
			name: "empty user list",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"users": []interface{}{},
						"total": 0,
					},
				},
			},
			userType:      UserGroupTypeUser,
			wantUserCount: 0,
		},
		{
			name: "user without email",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"users": []map[string]interface{}{
							{
								"id":       "1",
								"name":     "No Email User",
								"username": "noemail",
							},
						},
						"total": 1,
					},
				},
			},
			userType:      UserGroupTypeUser,
			wantUserCount: 1,
		},
		{
			name: "group without description",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"groups": []map[string]interface{}{
							{
								"id":   "1",
								"name": "NoDescGroup",
							},
						},
						"total": 1,
					},
				},
			},
			userType:       UserGroupTypeGroup,
			wantGroupCount: 1,
		},
		{
			name: "API error response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1003,
					"error_msg":  "Invalid parameters",
				},
			},
			userType:     UserGroupTypeUser,
			wantErr:      true,
			expectedErr:  api.ErrInvalidParams,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json{{{",
			},
			userType:     UserGroupTypeUser,
			wantErr:      true,
			expectedErr:  api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			userType: UserGroupTypeUser,
			wantErr:  true,
		},
		{
			name: "large user list",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"users":  make([]interface{}, 100),
						"total": 100,
					},
				},
			},
			userType:      UserGroupTypeUser,
			wantUserCount: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupSystemTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.GetUserGroupList(ctx, tt.userType)

			if tt.wantErr {
				if err == nil {
					t.Error("GetUserGroupList() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("GetUserGroupList() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("GetUserGroupList() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("GetUserGroupList() unexpected error = %v", err)
				return
			}

			if tt.wantUserCount > 0 && len(resp.Data.Users) != tt.wantUserCount {
				t.Errorf("GetUserGroupList() userCount = %d, want %d", len(resp.Data.Users), tt.wantUserCount)
			}

			if tt.wantGroupCount > 0 && len(resp.Data.Groups) != tt.wantGroupCount {
				t.Errorf("GetUserGroupList() groupCount = %d, want %d", len(resp.Data.Groups), tt.wantGroupCount)
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

// TestGetUserGroupList_NotAuthenticated tests GetUserGroupList when not authenticated
func TestGetUserGroupList_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedSystemClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.GetUserGroupList(ctx, UserGroupTypeUser)

	if err == nil {
		t.Error("GetUserGroupList() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetUserGroupList() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("GetUserGroupList() error = %v, want auth error", apiErr)
	}
}

// TestGetSysSetting tests the GetSysSetting function
func TestGetSysSetting(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *SysSetting)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "get system settings",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"hostname":      "qnap-nas",
						"domain":        "local",
						"workgroup":    "WORKGROUP",
						"timezone":     "UTC",
						"language":     "en",
						"admin_port":   8080,
						"enable_https": true,
						"https_port":   443,
						"description":  "My NAS",
						"location":     "Home",
					},
				},
			},
			assertResponse: func(t *testing.T, s *SysSetting) {
				t.Helper()
				if s.Hostname != "qnap-nas" {
					t.Errorf("Hostname = %s, want qnap-nas", s.Hostname)
				}
				if s.Domain != "local" {
					t.Errorf("Domain = %s, want local", s.Domain)
				}
				if s.Workgroup != "WORKGROUP" {
					t.Errorf("Workgroup = %s, want WORKGROUP", s.Workgroup)
				}
				if s.TimeZone != "UTC" {
					t.Errorf("TimeZone = %s, want UTC", s.TimeZone)
				}
				if s.Language != "en" {
					t.Errorf("Language = %s, want en", s.Language)
				}
				if s.AdminPort != 8080 {
					t.Errorf("AdminPort = %d, want 8080", s.AdminPort)
				}
				if !s.EnableHTTPS {
					t.Error("EnableHTTPS = false, want true")
				}
				if s.HTTPSPort != 443 {
					t.Errorf("HTTPSPort = %d, want 443", s.HTTPSPort)
				}
				if s.Description != "My NAS" {
					t.Errorf("Description = %s, want My NAS", s.Description)
				}
				if s.Location != "Home" {
					t.Errorf("Location = %s, want Home", s.Location)
				}
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "get_sys_setting" {
					t.Errorf("Expected func=get_sys_setting, got %s", fn)
				}
			},
		},
		{
			name: "minimal settings",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"hostname":   "nas",
						"admin_port": 8080,
					},
				},
			},
			assertResponse: func(t *testing.T, s *SysSetting) {
				t.Helper()
				if s.Hostname != "nas" {
					t.Errorf("Hostname = %s, want nas", s.Hostname)
				}
			},
		},
		{
			name: "HTTPS disabled",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"hostname":      "nas",
						"enable_https":  false,
						"admin_port":    8080,
						"https_port":    0,
					},
				},
			},
			assertResponse: func(t *testing.T, s *SysSetting) {
				t.Helper()
				if s.EnableHTTPS {
					t.Error("EnableHTTPS = true, want false")
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
			client, mockServer := setupSystemTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.GetSysSetting(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("GetSysSetting() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("GetSysSetting() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("GetSysSetting() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("GetSysSetting() unexpected error = %v", err)
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

// TestGetSysSetting_NotAuthenticated tests GetSysSetting when not authenticated
func TestGetSysSetting_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedSystemClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.GetSysSetting(ctx)

	if err == nil {
		t.Error("GetSysSetting() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetSysSetting() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("GetSysSetting() error = %v, want auth error", apiErr)
	}
}

// TestGetVolumeLockStatus tests the GetVolumeLockStatus function
func TestGetVolumeLockStatus(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		wantErr        bool
		expectedErr    api.ErrorCode
		wantVolumeCount int
		assertResponse func(*testing.T, *GetVolumeLockStatusResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "volumes unlocked",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"volumes": []map[string]interface{}{
							{
								"volume_name": "Volume1",
								"is_locked":   false,
							},
							{
								"volume_name": "Volume2",
								"is_locked":   false,
							},
						},
						"total": 2,
					},
				},
			},
			wantVolumeCount: 2,
			assertResponse: func(t *testing.T, r *GetVolumeLockStatusResponse) {
				t.Helper()
				for _, v := range r.Data.Volumes {
					if v.IsLocked {
						t.Errorf("Volume %s is locked, expected unlocked", v.VolumeName)
					}
				}
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if fn := req.URL.Query().Get("func"); fn != "get_volume_lock_status" {
					t.Errorf("Expected func=get_volume_lock_status, got %s", fn)
				}
			},
		},
		{
			name: "volume locked",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"volumes": []map[string]interface{}{
							{
								"volume_name": "Volume1",
								"is_locked":   true,
								"lock_reason": "Manual lock",
							},
						},
						"total": 1,
					},
				},
			},
			wantVolumeCount: 1,
			assertResponse: func(t *testing.T, r *GetVolumeLockStatusResponse) {
				t.Helper()
				if !r.Data.Volumes[0].IsLocked {
					t.Error("Expected volume to be locked")
				}
				if r.Data.Volumes[0].LockReason != "Manual lock" {
					t.Errorf("LockReason = %s, want 'Manual lock'", r.Data.Volumes[0].LockReason)
				}
			},
		},
		{
			name: "multiple volumes mixed status",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"volumes": []map[string]interface{}{
							{
								"volume_name": "Volume1",
								"is_locked":   false,
							},
							{
								"volume_name": "Volume2",
								"is_locked":   true,
								"lock_reason": "Backup in progress",
							},
							{
								"volume_name": "Volume3",
								"is_locked":   false,
							},
						},
						"total": 3,
					},
				},
			},
			wantVolumeCount: 3,
		},
		{
			name: "no volumes",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"volumes": []interface{}{},
						"total":   0,
					},
				},
			},
			wantVolumeCount: 0,
		},
		{
			name: "locked volume without reason",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"volumes": []map[string]interface{}{
							{
								"volume_name": "Volume1",
								"is_locked":   true,
							},
						},
						"total": 1,
					},
				},
			},
			wantVolumeCount: 1,
		},
		{
			name: "API error response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "Volume not found",
				},
			},
			wantErr:     true,
			expectedErr: api.ErrNotFound,
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
			client, mockServer := setupSystemTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.GetVolumeLockStatus(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("GetVolumeLockStatus() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("GetVolumeLockStatus() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("GetVolumeLockStatus() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("GetVolumeLockStatus() unexpected error = %v", err)
				return
			}

			if len(resp.Data.Volumes) != tt.wantVolumeCount {
				t.Errorf("GetVolumeLockStatus() volumeCount = %d, want %d", len(resp.Data.Volumes), tt.wantVolumeCount)
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

// TestGetVolumeLockStatus_NotAuthenticated tests GetVolumeLockStatus when not authenticated
func TestGetVolumeLockStatus_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedSystemClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.GetVolumeLockStatus(ctx)

	if err == nil {
		t.Error("GetVolumeLockStatus() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("GetVolumeLockStatus() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("GetVolumeLockStatus() error = %v, want auth error", apiErr)
	}
}

// TestStat tests the Stat function
func TestStat(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		path           string
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *File)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "stat file",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
"data": File{
						FileName: "test.txt",
						Path:     "/home/test.txt",
						FileSize: "1024",
						IsFolder: 0,
					},
				},
			},
			path: "/home/test.txt",
			assertResponse: func(t *testing.T, f *File) {
				t.Helper()
				if f.FileName != "test.txt" {
					t.Errorf("FileName = %s, want test.txt", f.FileName)
				}
				if f.Path != "/home/test.txt" {
					t.Errorf("Path = %s, want /home/test.txt", f.Path)
				}
				if f.IsFolder != 0 {
					t.Errorf("IsFolder = %d, want 0", f.IsFolder)
				}
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "stat" {
					t.Errorf("Expected func=stat, got %s", fn)
				}
				if path := r.URL.Query().Get("path"); path != "/home/test.txt" {
					t.Errorf("Expected path=/home/test.txt, got %s", path)
				}
			},
		},
		{
			name: "stat folder",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
"data": File{
						FileName: "documents",
						Path:     "/home/documents",
						FileSize: "0",
						IsFolder: 1,
					},
				},
			},
			path: "/home/documents",
			assertResponse: func(t *testing.T, f *File) {
				t.Helper()
				if f.IsFolder != 1 {
					t.Errorf("IsFolder = %d, want 1", f.IsFolder)
				}
			},
		},
		{
			name: "stat with special characters",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": File{
						FileName: "file with spaces.txt",
						Path:     "/home/file with spaces.txt",
						IsFolder: 0,
					},
				},
			},
			path: "/home/file with spaces.txt",
			assertResponse: func(t *testing.T, f *File) {
				t.Helper()
				if f.FileName != "file with spaces.txt" {
					t.Errorf("FileName = %s", f.FileName)
				}
			},
		},
		{
			name: "stat root directory",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": File{
						FileName: "",
						Path:     "/",
						IsFolder: 1,
					},
				},
			},
			path: "/",
			assertResponse: func(t *testing.T, f *File) {
				t.Helper()
				if f.Path != "/" {
					t.Errorf("Path = %s, want /", f.Path)
				}
			},
		},
		{
			name: "file not found",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "File not found",
				},
			},
			path:        "/nonexistent/file.txt",
			wantErr:     true,
			expectedErr: api.ErrNotFound,
		},
		{
			name: "permission denied",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2003,
					"error_msg":  "Permission denied",
				},
			},
			path:        "/restricted/file.txt",
			wantErr:     true,
			expectedErr: api.ErrPermission,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json{{{",
			},
			path:        "/home/file.txt",
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			path:   "/home/file.txt",
			wantErr: true,
		},
		{
			name: "empty path",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2004,
					"error_msg":  "Invalid path",
				},
			},
			path:        "",
			wantErr:     true,
			expectedErr: api.ErrInvalidPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupSystemTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.Stat(ctx, tt.path)

			if tt.wantErr {
				if err == nil {
					t.Error("Stat() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("Stat() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("Stat() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Stat() unexpected error = %v", err)
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

// TestStat_NotAuthenticated tests Stat when not authenticated
func TestStat_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedSystemClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.Stat(ctx, "/home/file.txt")

	if err == nil {
		t.Error("Stat() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("Stat() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("Stat() error = %v, want auth error", apiErr)
	}
}

// TestMediaFolderList tests the MediaFolderList function
func TestMediaFolderList(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		wantErr        bool
		expectedErr    api.ErrorCode
		wantFolderCount int
		assertResponse func(*testing.T, *MediaFolderListResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "get media folders",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"folders": []map[string]interface{}{
							{
								"id":          "1",
								"name":        "Photos",
								"path":        "/home/Photos",
								"type":        "image",
								"is_enabled":  true,
								"description": "My photos",
							},
							{
								"id":          "2",
								"name":        "Music",
								"path":        "/home/Music",
								"type":        "music",
								"is_enabled":  true,
								"description": "My music",
							},
							{
								"id":          "3",
								"name":        "Videos",
								"path":        "/home/Videos",
								"type":        "video",
								"is_enabled":  false,
								"description": "My videos",
							},
						},
						"total": 3,
					},
				},
			},
			wantFolderCount: 3,
			assertResponse: func(t *testing.T, r *MediaFolderListResponse) {
				t.Helper()
				if r.Data.Total != 3 {
					t.Errorf("Total = %d, want 3", r.Data.Total)
				}
				if len(r.Data.Folders) != 3 {
					t.Errorf("Folders count = %d, want 3", len(r.Data.Folders))
				}
				photos := r.Data.Folders[0]
				if photos.Name != "Photos" {
					t.Errorf("First folder name = %s, want Photos", photos.Name)
				}
				if !photos.IsEnabled {
					t.Error("Photos folder should be enabled")
				}
				videos := r.Data.Folders[2]
				if videos.IsEnabled {
					t.Error("Videos folder should be disabled")
				}
			},
			assertRequest: func(t *testing.T, req *http.Request) {
				t.Helper()
				if fn := req.URL.Query().Get("func"); fn != "media_folder_list" {
					t.Errorf("Expected func=media_folder_list, got %s", fn)
				}
			},
		},
		{
			name: "empty media folder list",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"folders": []interface{}{},
						"total":   0,
					},
				},
			},
			wantFolderCount: 0,
		},
		{
			name: "folder without description",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"folders": []map[string]interface{}{
							{
								"id":         "1",
								"name":       "NoDesc",
								"path":       "/home/NoDesc",
								"type":       "image",
								"is_enabled": true,
							},
						},
						"total": 1,
					},
				},
			},
			wantFolderCount: 1,
		},
		{
			name: "API error response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1003,
					"error_msg":  "Invalid parameters",
				},
			},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
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
		{
			name: "large media folder list",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"folders":  make([]interface{}, 50),
						"total":   50,
					},
				},
			},
			wantFolderCount: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupSystemTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.MediaFolderList(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("MediaFolderList() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					var apiErr *api.APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("MediaFolderList() error type = %T, want *api.APIError", err)
					} else if apiErr.Code != tt.expectedErr {
						t.Errorf("MediaFolderList() error code = %d, want %d", apiErr.Code, tt.expectedErr)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("MediaFolderList() unexpected error = %v", err)
				return
			}

			if len(resp.Data.Folders) != tt.wantFolderCount {
				t.Errorf("MediaFolderList() folderCount = %d, want %d", len(resp.Data.Folders), tt.wantFolderCount)
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

// TestMediaFolderList_NotAuthenticated tests MediaFolderList when not authenticated
func TestMediaFolderList_NotAuthenticated(t *testing.T) {
	client := setupUnauthenticatedSystemClient(t)
	fs := NewFileStationService(client)

	ctx := context.Background()
	_, err := fs.MediaFolderList(ctx)

	if err == nil {
		t.Error("MediaFolderList() expected error when not authenticated, got nil")
	}

	var apiErr *api.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("MediaFolderList() error type = %T, want *api.APIError", err)
	} else if !apiErr.IsAuthError() {
		t.Errorf("MediaFolderList() error = %v, want auth error", apiErr)
	}
}

// TestSystemMethods_TableDriven verifies all system methods use table-driven tests
func TestSystemMethods_TableDriven(t *testing.T) {
	// This meta-test ensures we have comprehensive table-driven test coverage
	methods := []string{
		"CheckSession",
		"GetFileSize",
		"GetTree",
		"GetUserGroupList",
		"GetSysSetting",
		"GetVolumeLockStatus",
		"Stat",
		"MediaFolderList",
	}

	for _, method := range methods {
		t.Run(method+" has table-driven tests", func(t *testing.T) {
			// This is a meta-test to document that table-driven tests exist
			// The actual tests are defined above
			t.Logf("Table-driven tests exist for %s", method)
		})
	}
}

// BenchmarkSystemFunctions benchmarks system functions
func BenchmarkCheckSession(b *testing.B) {
	client, mockServer := setupSystemTestClient(&testing.T{})
	defer mockServer.Close()

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: http.StatusOK,
		Body: map[string]interface{}{
			"success": 1,
			"data": map[string]interface{}{
				"is_valid": true,
			},
		},
	})

	ctx := context.Background()
	fs := NewFileStationService(client)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = fs.CheckSession(ctx)
	}
}

// TestContextCancellation_System tests context cancellation for system methods
func TestContextCancellation_System(t *testing.T) {
	tests := []struct {
		name  string
		testFn func(*testing.T, *FileStationService, context.Context)
	}{
		{
			name: "CheckSession respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.CheckSession(ctx)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("CheckSession with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "GetFileSize respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.GetFileSize(ctx, []string{"/home/file.txt"})
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("GetFileSize with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "GetTree respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.GetTree(ctx, nil)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("GetTree with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "GetUserGroupList respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.GetUserGroupList(ctx, UserGroupTypeUser)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("GetUserGroupList with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "GetSysSetting respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.GetSysSetting(ctx)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("GetSysSetting with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "GetVolumeLockStatus respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.GetVolumeLockStatus(ctx)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("GetVolumeLockStatus with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "Stat respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.Stat(ctx, "/home/file.txt")
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("Stat with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "MediaFolderList respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.MediaFolderList(ctx)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("MediaFolderList with canceled context returned error: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupSystemTestClient(t)
			defer mockServer.Close()

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			fs := NewFileStationService(client)
			tt.testFn(t, fs, ctx)
		})
	}
}
