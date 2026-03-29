package filestation

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/fatelei/qnap-filestation/pkg/api"
	"github.com/fatelei/qnap-filestation/internal/testutil"
)

// setupTestClient creates a test client and mock server for testing
func setupShareTestClient(t *testing.T) (*api.Client, *testutil.MockServer) {
	t.Helper()

	mockServer := testutil.NewMockServer()
	url := mockServer.URL()
	host := url[7:] // Remove "http://"

	config := &api.Config{
		Host:     host,
		Port:     0,
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn})),
	}

	client, err := api.NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	return client, mockServer
}

// assertErrorCode is a helper to check API error codes
func assertErrorCode(t *testing.T, err error, wantCode api.ErrorCode) {
	t.Helper()
	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Errorf("expected *api.APIError, got %T", err)
		return
	}
	if apiErr.Code != wantCode {
		t.Errorf("error code = %d, want %d", apiErr.Code, wantCode)
	}
}

// assertAuthError is a helper to check if an error is an auth error
func assertAuthError(t *testing.T, err error) {
	t.Helper()
	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Errorf("expected *api.APIError, got %T", err)
		return
	}
	if !apiErr.IsAuthError() {
		t.Errorf("expected auth error, got %v", err)
	}
}

// TestCreateShareLink_Success tests successful share link creation
func TestCreateShareLink_Success(t *testing.T) {
	client, mockServer := setupShareTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"list": []map[string]string{
				{
					"name": "test.txt",
					"url":  "https://example.com/share/test123",
				},
			},
		},
	})

	shareLink, err := fs.CreateShareLink(ctx, "/home/test.txt", nil)

	if err != nil {
		t.Errorf("CreateShareLink() unexpected error = %v", err)
	}

	if shareLink == nil {
		t.Fatal("CreateShareLink() returned nil shareLink")
	}

	if shareLink.URL != "https://example.com/share/test123" {
		t.Errorf("CreateShareLink() URL = %s, want https://example.com/share/test123", shareLink.URL)
	}
}

// TestCreateShareLink_TableDriven tests CreateShareLink with various scenarios
func TestCreateShareLink_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		mockResponse   testutil.MockResponse
		wantURL        string
		wantErr        bool
		errCode        api.ErrorCode
		checkRequest   func(*testing.T, *http.Request)
	}{
		{
			name: "success with file in root",
			path: "/test.txt",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"list": []map[string]string{
						{
							"name": "test.txt",
							"url":  "https://example.com/share/root123",
						},
					},
				},
			},
			wantURL: "https://example.com/share/root123",
			wantErr: false,
		},
		{
			name: "success with nested path",
			path: "/home/user/documents/file.pdf",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"list": []map[string]string{
						{
							"name": "file.pdf",
							"url":  "https://example.com/share/pdf456",
						},
					},
				},
			},
			wantURL: "https://example.com/share/pdf456",
			wantErr: false,
		},
		{
			name: "empty list response",
			path: "/home/empty.txt",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"list": []interface{}{},
				},
			},
			wantErr: true,
			errCode: api.ErrUnknown,
		},
		{
			name: "API error response",
			path: "/home/error.txt",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "file not found",
				},
			},
			wantErr: true,
			errCode: api.ErrUnknown,
		},
		{
			name: "invalid path - too short",
			path: "/",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body:       map[string]interface{}{},
			},
			wantErr: true,
			errCode: api.ErrUnknown,
		},
		{
			name: "network error",
			path: "/home/network.txt",
			mockResponse: testutil.MockResponse{
				StatusCode: 500,
				Error:      "network error",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupShareTestClient(t)
			defer mockServer.Close()

			ctx := context.Background()
			fs := NewFileStationService(client)

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			shareLink, err := fs.CreateShareLink(ctx, tt.path, nil)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateShareLink() expected error but got none")
					return
				}
				if tt.errCode != 0 {
					assertErrorCode(t, err, tt.errCode)
				}
				return
			}

			if err != nil {
				t.Errorf("CreateShareLink() unexpected error = %v", err)
				return
			}

			if shareLink.URL != tt.wantURL {
				t.Errorf("CreateShareLink() URL = %s, want %s", shareLink.URL, tt.wantURL)
			}

			if tt.checkRequest != nil {
				lastReq := mockServer.GetLastRequest()
				if lastReq != nil {
					tt.checkRequest(t, lastReq)
				}
			}
		})
	}
}

// TestCreateShareLink_NotAuthenticated tests CreateShareLink without authentication
func TestCreateShareLink_NotAuthenticated(t *testing.T) {
	client, _ := setupShareTestClient(t)
	client.SetSID("") // Clear SID to simulate not authenticated

	ctx := context.Background()
	fs := NewFileStationService(client)

	_, err := fs.CreateShareLink(ctx, "/home/test.txt", nil)

	if err == nil {
		t.Error("CreateShareLink() expected error when not authenticated")
	}

	assertAuthError(t, err)
}

// TestDeleteShareLink_Success tests successful share link deletion
func TestDeleteShareLink_Success(t *testing.T) {
	client, mockServer := setupShareTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
		},
	})

	err := fs.DeleteShareLink(ctx, "test-share")

	if err != nil {
		t.Errorf("DeleteShareLink() unexpected error = %v", err)
	}
}

// TestDeleteShareLink_TableDriven tests DeleteShareLink with various scenarios
func TestDeleteShareLink_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		shareName    string
		mockResponse testutil.MockResponse
		wantErr      bool
		errCode      api.ErrorCode
	}{
		{
			name:      "success",
			shareName: "my-share",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
				},
			},
			wantErr: false,
		},
		{
			name:      "share not found",
			shareName: "nonexistent-share",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "share not found",
				},
			},
			wantErr: true,
			errCode: api.ErrNotFound,
		},
		{
			name:      "permission denied",
			shareName: "protected-share",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2003,
					"error_msg":  "permission denied",
				},
			},
			wantErr: true,
			errCode: api.ErrPermission,
		},
		{
			name:      "invalid share name",
			shareName: "",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1003,
					"error_msg":  "invalid parameter",
				},
			},
			wantErr: true,
			errCode: api.ErrInvalidParams,
		},
		{
			name:      "network error",
			shareName: "network-share",
			mockResponse: testutil.MockResponse{
				StatusCode: 500,
				Error:      "connection refused",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupShareTestClient(t)
			defer mockServer.Close()

			ctx := context.Background()
			fs := NewFileStationService(client)

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			err := fs.DeleteShareLink(ctx, tt.shareName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteShareLink() expected error but got none")
					return
				}
				if tt.errCode != 0 {
					assertErrorCode(t, err, tt.errCode)
				}
				return
			}

			if err != nil {
				t.Errorf("DeleteShareLink() unexpected error = %v", err)
			}
		})
	}
}

// TestDeleteShareLink_NotAuthenticated tests DeleteShareLink without authentication
func TestDeleteShareLink_NotAuthenticated(t *testing.T) {
	client, _ := setupShareTestClient(t)
	client.SetSID("")

	ctx := context.Background()
	fs := NewFileStationService(client)

	err := fs.DeleteShareLink(ctx, "test-share")

	if err == nil {
		t.Error("DeleteShareLink() expected error when not authenticated")
	}

	assertAuthError(t, err)
}

// TestUpdateShareLink_Success tests successful share link update
func TestUpdateShareLink_Success(t *testing.T) {
	client, mockServer := setupShareTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
			"data": ShareLink{
				ID:    "share-123",
				URL:   "https://example.com/share/updated",
				Name:  "Updated Share",
				Expires: time.Now().Add(24 * time.Hour),
			},
		},
	})

	options := &UpdateShareLinkOptions{
		SSID:       "share-123",
		ExpireTime: time.Now().Add(24 * time.Hour).Unix(),
		Password:   "newpass123",
		ValidDays:  7,
	}

	shareLink, err := fs.UpdateShareLink(ctx, options)

	if err != nil {
		t.Errorf("UpdateShareLink() unexpected error = %v", err)
	}

	if shareLink == nil {
		t.Fatal("UpdateShareLink() returned nil shareLink")
	}

	if shareLink.ID != "share-123" {
		t.Errorf("UpdateShareLink() ID = %s, want share-123", shareLink.ID)
	}
}

// TestUpdateShareLink_TableDriven tests UpdateShareLink with various scenarios
func TestUpdateShareLink_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		options      *UpdateShareLinkOptions
		mockResponse testutil.MockResponse
		wantErr      bool
		errCode      api.ErrorCode
	}{
		{
			name: "update with expire time",
			options: &UpdateShareLinkOptions{
				SSID:       "share-123",
				ExpireTime: time.Now().Add(24 * time.Hour).Unix(),
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": ShareLink{
						ID: "share-123",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "update with password",
			options: &UpdateShareLinkOptions{
				SSID:     "share-456",
				Password: "secure123",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": ShareLink{
						ID: "share-456",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "update with valid days",
			options: &UpdateShareLinkOptions{
				SSID:      "share-789",
				ValidDays: 30,
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": ShareLink{
						ID: "share-789",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "update all fields",
			options: &UpdateShareLinkOptions{
				SSID:       "share-all",
				ExpireTime: time.Now().Add(48 * time.Hour).Unix(),
				Password:   "allfields",
				ValidDays:  14,
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": ShareLink{
						ID: "share-all",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty options",
			options: &UpdateShareLinkOptions{
				SSID: "",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1003,
					"error_msg":  "invalid parameter",
				},
			},
			wantErr: true,
			errCode: api.ErrInvalidParams,
		},
		{
			name: "share not found",
			options: &UpdateShareLinkOptions{
				SSID: "nonexistent",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "share not found",
				},
			},
			wantErr: true,
			errCode: api.ErrNotFound,
		},
		{
			name: "network error",
			options: &UpdateShareLinkOptions{
				SSID: "network-error",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 500,
				Error:      "timeout",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupShareTestClient(t)
			defer mockServer.Close()

			ctx := context.Background()
			fs := NewFileStationService(client)

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			shareLink, err := fs.UpdateShareLink(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateShareLink() expected error but got none")
					return
				}
				if tt.errCode != 0 {
					assertErrorCode(t, err, tt.errCode)
				}
				return
			}

			if err != nil {
				t.Errorf("UpdateShareLink() unexpected error = %v", err)
				return
			}

			if shareLink == nil {
				t.Error("UpdateShareLink() returned nil shareLink")
			}
		})
	}
}

// TestUpdateShareLink_NotAuthenticated tests UpdateShareLink without authentication
func TestUpdateShareLink_NotAuthenticated(t *testing.T) {
	client, _ := setupShareTestClient(t)
	client.SetSID("")

	ctx := context.Background()
	fs := NewFileStationService(client)

	options := &UpdateShareLinkOptions{
		SSID: "test-share",
	}

	_, err := fs.UpdateShareLink(ctx, options)

	if err == nil {
		t.Error("UpdateShareLink() expected error when not authenticated")
	}

	assertAuthError(t, err)
}

// TestGetShareList_Success tests successful share list retrieval
func TestGetShareList_Success(t *testing.T) {
	client, mockServer := setupShareTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
			"data": map[string]interface{}{
				"shares": []ShareLink{
					{
						ID:   "share-1",
						Name: "Share 1",
						URL:  "https://example.com/share/1",
					},
					{
						ID:   "share-2",
						Name: "Share 2",
						URL:  "https://example.com/share/2",
					},
				},
				"total": 2,
			},
		},
	})

	shares, err := fs.GetShareList(ctx)

	if err != nil {
		t.Errorf("GetShareList() unexpected error = %v", err)
	}

	if len(shares) != 2 {
		t.Errorf("GetShareList() returned %d shares, want 2", len(shares))
	}

	if len(shares) > 0 && shares[0].ID != "share-1" {
		t.Errorf("GetShareList() first share ID = %s, want share-1", shares[0].ID)
	}
}

// TestGetShareList_TableDriven tests GetShareList with various scenarios
func TestGetShareList_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		mockResponse testutil.MockResponse
		wantCount    int
		wantErr      bool
		errCode      api.ErrorCode
	}{
		{
			name: "success with multiple shares",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"shares": []ShareLink{
							{ID: "1", Name: "Share 1"},
							{ID: "2", Name: "Share 2"},
							{ID: "3", Name: "Share 3"},
						},
						"total": 3,
					},
				},
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name: "empty share list",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"shares": []ShareLink{},
						"total":   0,
					},
				},
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "API error",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1000,
					"error_msg":  "authentication failed",
				},
			},
			wantErr: true,
			errCode: api.ErrAuthFailed,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: 500,
				Error:      "network error",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupShareTestClient(t)
			defer mockServer.Close()

			ctx := context.Background()
			fs := NewFileStationService(client)

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			shares, err := fs.GetShareList(ctx)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetShareList() expected error but got none")
					return
				}
				if tt.errCode != 0 {
					assertErrorCode(t, err, tt.errCode)
				}
				return
			}

			if err != nil {
				t.Errorf("GetShareList() unexpected error = %v", err)
				return
			}

			if len(shares) != tt.wantCount {
				t.Errorf("GetShareList() returned %d shares, want %d", len(shares), tt.wantCount)
			}
		})
	}
}

// TestGetShareList_NotAuthenticated tests GetShareList without authentication
func TestGetShareList_NotAuthenticated(t *testing.T) {
	client, _ := setupShareTestClient(t)
	client.SetSID("")

	ctx := context.Background()
	fs := NewFileStationService(client)

	_, err := fs.GetShareList(ctx)

	if err == nil {
		t.Error("GetShareList() expected error when not authenticated")
	}

	assertAuthError(t, err)
}

// TestGetShareSublist_Success tests successful share sublist retrieval
func TestGetShareSublist_Success(t *testing.T) {
	client, mockServer := setupShareTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
			"data": map[string]interface{}{
				"members": []ShareMember{
					{
						ID:     "user-1",
						Name:   "John Doe",
						Type:   "user",
						Access: "read",
					},
					{
						ID:     "group-1",
						Name:   "Developers",
						Type:   "group",
						Access: "write",
					},
				},
				"total": 2,
			},
		},
	})

	members, err := fs.GetShareSublist(ctx, "my-share")

	if err != nil {
		t.Errorf("GetShareSublist() unexpected error = %v", err)
	}

	if len(members) != 2 {
		t.Errorf("GetShareSublist() returned %d members, want 2", len(members))
	}

	if len(members) > 0 && members[0].Name != "John Doe" {
		t.Errorf("GetShareSublist() first member name = %s, want John Doe", members[0].Name)
	}
}

// TestGetShareSublist_TableDriven tests GetShareSublist with various scenarios
func TestGetShareSublist_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		shareName    string
		mockResponse testutil.MockResponse
		wantCount    int
		wantErr      bool
		errCode      api.ErrorCode
	}{
		{
			name:      "success with members",
			shareName: "team-share",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"members": []ShareMember{
							{ID: "1", Name: "Alice", Type: "user", Access: "read"},
							{ID: "2", Name: "Bob", Type: "user", Access: "write"},
						},
						"total": 2,
					},
				},
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "empty member list",
			shareName: "empty-share",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"members": []ShareMember{},
						"total":   0,
					},
				},
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "share not found",
			shareName: "nonexistent",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "share not found",
				},
			},
			wantErr: true,
			errCode: api.ErrNotFound,
		},
		{
			name:      "network error",
			shareName: "network-share",
			mockResponse: testutil.MockResponse{
				StatusCode: 500,
				Error:      "timeout",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupShareTestClient(t)
			defer mockServer.Close()

			ctx := context.Background()
			fs := NewFileStationService(client)

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			members, err := fs.GetShareSublist(ctx, tt.shareName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetShareSublist() expected error but got none")
					return
				}
				if tt.errCode != 0 {
					assertErrorCode(t, err, tt.errCode)
				}
				return
			}

			if err != nil {
				t.Errorf("GetShareSublist() unexpected error = %v", err)
				return
			}

			if len(members) != tt.wantCount {
				t.Errorf("GetShareSublist() returned %d members, want %d", len(members), tt.wantCount)
			}
		})
	}
}

// TestGetShareSublist_NotAuthenticated tests GetShareSublist without authentication
func TestGetShareSublist_NotAuthenticated(t *testing.T) {
	client, _ := setupShareTestClient(t)
	client.SetSID("")

	ctx := context.Background()
	fs := NewFileStationService(client)

	_, err := fs.GetShareSublist(ctx, "test-share")

	if err == nil {
		t.Error("GetShareSublist() expected error when not authenticated")
	}

	assertAuthError(t, err)
}

// TestAddShareSublist_Success tests successful member addition to share
func TestAddShareSublist_Success(t *testing.T) {
	client, mockServer := setupShareTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
		},
	})

	options := &AddShareSublistOptions{
		ShareName: "my-share",
		UserID:    "user-123",
		Access:    "read",
		IsGroup:   false,
	}

	err := fs.AddShareSublist(ctx, options)

	if err != nil {
		t.Errorf("AddShareSublist() unexpected error = %v", err)
	}
}

// TestAddShareSublist_TableDriven tests AddShareSublist with various scenarios
func TestAddShareSublist_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		options      *AddShareSublistOptions
		mockResponse testutil.MockResponse
		wantErr      bool
		errCode      api.ErrorCode
	}{
		{
			name: "add user with read access",
			options: &AddShareSublistOptions{
				ShareName: "read-share",
				UserID:    "user-1",
				Access:    "read",
				IsGroup:   false,
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
				},
			},
			wantErr: false,
		},
		{
			name: "add user with write access",
			options: &AddShareSublistOptions{
				ShareName: "write-share",
				UserID:    "user-2",
				Access:    "write",
				IsGroup:   false,
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
				},
			},
			wantErr: false,
		},
		{
			name: "add group",
			options: &AddShareSublistOptions{
				ShareName: "group-share",
				UserID:    "group-1",
				Access:    "read",
				IsGroup:   true,
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
				},
			},
			wantErr: false,
		},
		{
			name: "share not found",
			options: &AddShareSublistOptions{
				ShareName: "nonexistent",
				UserID:    "user-1",
				Access:    "read",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "share not found",
				},
			},
			wantErr: true,
			errCode: api.ErrNotFound,
		},
		{
			name: "user not found",
			options: &AddShareSublistOptions{
				ShareName: "my-share",
				UserID:    "nonexistent-user",
				Access:    "read",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "user not found",
				},
			},
			wantErr: true,
			errCode: api.ErrNotFound,
		},
		{
			name: "network error",
			options: &AddShareSublistOptions{
				ShareName: "network-share",
				UserID:    "user-1",
				Access:    "read",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 500,
				Error:      "network error",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupShareTestClient(t)
			defer mockServer.Close()

			ctx := context.Background()
			fs := NewFileStationService(client)

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			err := fs.AddShareSublist(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Errorf("AddShareSublist() expected error but got none")
					return
				}
				if tt.errCode != 0 {
					assertErrorCode(t, err, tt.errCode)
				}
				return
			}

			if err != nil {
				t.Errorf("AddShareSublist() unexpected error = %v", err)
			}
		})
	}
}

// TestAddShareSublist_NotAuthenticated tests AddShareSublist without authentication
func TestAddShareSublist_NotAuthenticated(t *testing.T) {
	client, _ := setupShareTestClient(t)
	client.SetSID("")

	ctx := context.Background()
	fs := NewFileStationService(client)

	options := &AddShareSublistOptions{
		ShareName: "my-share",
		UserID:    "user-1",
		Access:    "read",
	}

	err := fs.AddShareSublist(ctx, options)

	if err == nil {
		t.Error("AddShareSublist() expected error when not authenticated")
	}

	assertAuthError(t, err)
}

// TestDeleteShareSublist_Success tests successful member removal from share
func TestDeleteShareSublist_Success(t *testing.T) {
	client, mockServer := setupShareTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
		},
	})

	err := fs.DeleteShareSublist(ctx, "my-share", "user-123")

	if err != nil {
		t.Errorf("DeleteShareSublist() unexpected error = %v", err)
	}
}

// TestDeleteShareSublist_TableDriven tests DeleteShareSublist with various scenarios
func TestDeleteShareSublist_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		shareName    string
		userID       string
		mockResponse testutil.MockResponse
		wantErr      bool
		errCode      api.ErrorCode
	}{
		{
			name:      "success",
			shareName: "my-share",
			userID:    "user-123",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
				},
			},
			wantErr: false,
		},
		{
			name:      "remove from nonexistent share",
			shareName: "nonexistent",
			userID:    "user-1",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "share not found",
				},
			},
			wantErr: true,
			errCode: api.ErrNotFound,
		},
		{
			name:      "user not in share",
			shareName: "my-share",
			userID:    "nonexistent-user",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "user not found in share",
				},
			},
			wantErr: true,
			errCode: api.ErrNotFound,
		},
		{
			name:      "network error",
			shareName: "network-share",
			userID:    "user-1",
			mockResponse: testutil.MockResponse{
				StatusCode: 500,
				Error:      "network error",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupShareTestClient(t)
			defer mockServer.Close()

			ctx := context.Background()
			fs := NewFileStationService(client)

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			err := fs.DeleteShareSublist(ctx, tt.shareName, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteShareSublist() expected error but got none")
					return
				}
				if tt.errCode != 0 {
					assertErrorCode(t, err, tt.errCode)
				}
				return
			}

			if err != nil {
				t.Errorf("DeleteShareSublist() unexpected error = %v", err)
			}
		})
	}
}

// TestDeleteShareSublist_NotAuthenticated tests DeleteShareSublist without authentication
func TestDeleteShareSublist_NotAuthenticated(t *testing.T) {
	client, _ := setupShareTestClient(t)
	client.SetSID("")

	ctx := context.Background()
	fs := NewFileStationService(client)

	err := fs.DeleteShareSublist(ctx, "my-share", "user-1")

	if err == nil {
		t.Error("DeleteShareSublist() expected error when not authenticated")
	}

	assertAuthError(t, err)
}

// TestShareAccessControl_Success tests successful share access control update
func TestShareAccessControl_Success(t *testing.T) {
	client, mockServer := setupShareTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
		},
	})

	options := &ShareAccessControlOptions{
		ShareName:   "my-share",
		AccessLevel: "private",
		ReadOnly:    true,
		Writeable:   false,
	}

	err := fs.ShareAccessControl(ctx, options)

	if err != nil {
		t.Errorf("ShareAccessControl() unexpected error = %v", err)
	}
}

// TestShareAccessControl_TableDriven tests ShareAccessControl with various scenarios
func TestShareAccessControl_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		options      *ShareAccessControlOptions
		mockResponse testutil.MockResponse
		wantErr      bool
		errCode      api.ErrorCode
	}{
		{
			name: "read only access",
			options: &ShareAccessControlOptions{
				ShareName:   "readonly-share",
				AccessLevel: "private",
				ReadOnly:    true,
				Writeable:   false,
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
				},
			},
			wantErr: false,
		},
		{
			name: "read write access",
			options: &ShareAccessControlOptions{
				ShareName:   "readwrite-share",
				AccessLevel: "private",
				ReadOnly:    false,
				Writeable:   true,
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
				},
			},
			wantErr: false,
		},
		{
			name: "full access",
			options: &ShareAccessControlOptions{
				ShareName:   "full-share",
				AccessLevel: "public",
				ReadOnly:    true,
				Writeable:   true,
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
				},
			},
			wantErr: false,
		},
		{
			name: "share not found",
			options: &ShareAccessControlOptions{
				ShareName:   "nonexistent",
				AccessLevel: "private",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "share not found",
				},
			},
			wantErr: true,
			errCode: api.ErrNotFound,
		},
		{
			name: "permission denied",
			options: &ShareAccessControlOptions{
				ShareName:   "protected-share",
				AccessLevel: "private",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2003,
					"error_msg":  "permission denied",
				},
			},
			wantErr: true,
			errCode: api.ErrPermission,
		},
		{
			name: "network error",
			options: &ShareAccessControlOptions{
				ShareName:   "network-share",
				AccessLevel: "private",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 500,
				Error:      "network error",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupShareTestClient(t)
			defer mockServer.Close()

			ctx := context.Background()
			fs := NewFileStationService(client)

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			err := fs.ShareAccessControl(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ShareAccessControl() expected error but got none")
					return
				}
				if tt.errCode != 0 {
					assertErrorCode(t, err, tt.errCode)
				}
				return
			}

			if err != nil {
				t.Errorf("ShareAccessControl() unexpected error = %v", err)
			}
		})
	}
}

// TestShareAccessControl_NotAuthenticated tests ShareAccessControl without authentication
func TestShareAccessControl_NotAuthenticated(t *testing.T) {
	client, _ := setupShareTestClient(t)
	client.SetSID("")

	ctx := context.Background()
	fs := NewFileStationService(client)

	options := &ShareAccessControlOptions{
		ShareName:   "my-share",
		AccessLevel: "private",
	}

	err := fs.ShareAccessControl(ctx, options)

	if err == nil {
		t.Error("ShareAccessControl() expected error when not authenticated")
	}

	assertAuthError(t, err)
}

// TestSendShareMail_Success tests successful share mail sending
func TestSendShareMail_Success(t *testing.T) {
	client, mockServer := setupShareTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
		},
	})

	options := &SendShareMailOptions{
		ShareName: "my-share",
		To:        []string{"recipient@example.com"},
		Subject:   "Check out this file",
		Message:   "Here is the shared file you requested.",
	}

	err := fs.SendShareMail(ctx, options)

	if err != nil {
		t.Errorf("SendShareMail() unexpected error = %v", err)
	}
}

// TestSendShareMail_TableDriven tests SendShareMail with various scenarios
func TestSendShareMail_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		options      *SendShareMailOptions
		mockResponse testutil.MockResponse
		wantErr      bool
		errCode      api.ErrorCode
	}{
		{
			name: "send to single recipient",
			options: &SendShareMailOptions{
				ShareName: "my-share",
				To:        []string{"user@example.com"},
				Subject:   "Shared file",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
				},
			},
			wantErr: false,
		},
		{
			name: "send to multiple recipients",
			options: &SendShareMailOptions{
				ShareName: "my-share",
				To:        []string{"user1@example.com", "user2@example.com", "user3@example.com"},
				CC:        []string{"cc@example.com"},
				Subject:   "Multiple recipients",
				Message:   "This is sent to multiple people",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
				},
			},
			wantErr: false,
		},
		{
			name: "send with all fields",
			options: &SendShareMailOptions{
				ShareName: "full-share",
				To:        []string{"recipient@example.com"},
				CC:        []string{"cc1@example.com", "cc2@example.com"},
				Subject:   "Complete email",
				Message:   "This is a complete email test",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
				},
			},
			wantErr: false,
		},
		{
			name: "share not found",
			options: &SendShareMailOptions{
				ShareName: "nonexistent",
				To:        []string{"user@example.com"},
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "share not found",
				},
			},
			wantErr: true,
			errCode: api.ErrNotFound,
		},
		{
			name: "invalid email address",
			options: &SendShareMailOptions{
				ShareName: "my-share",
				To:        []string{"invalid-email"},
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1003,
					"error_msg":  "invalid email address",
				},
			},
			wantErr: true,
			errCode: api.ErrInvalidParams,
		},
		{
			name: "mail service unavailable",
			options: &SendShareMailOptions{
				ShareName: "my-share",
				To:        []string{"user@example.com"},
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 3001,
					"error_msg":  "mail service unavailable",
				},
			},
			wantErr: true,
			errCode: api.ErrNetwork,
		},
		{
			name: "network error",
			options: &SendShareMailOptions{
				ShareName: "network-share",
				To:        []string{"user@example.com"},
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 500,
				Error:      "network error",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupShareTestClient(t)
			defer mockServer.Close()

			ctx := context.Background()
			fs := NewFileStationService(client)

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			err := fs.SendShareMail(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SendShareMail() expected error but got none")
					return
				}
				if tt.errCode != 0 {
					assertErrorCode(t, err, tt.errCode)
				}
				return
			}

			if err != nil {
				t.Errorf("SendShareMail() unexpected error = %v", err)
			}
		})
	}
}

// TestSendShareMail_NotAuthenticated tests SendShareMail without authentication
func TestSendShareMail_NotAuthenticated(t *testing.T) {
	client, _ := setupShareTestClient(t)
	client.SetSID("")

	ctx := context.Background()
	fs := NewFileStationService(client)

	options := &SendShareMailOptions{
		ShareName: "my-share",
		To:        []string{"user@example.com"},
	}

	err := fs.SendShareMail(ctx, options)

	if err == nil {
		t.Error("SendShareMail() expected error when not authenticated")
	}

	assertAuthError(t, err)
}

// TestGetPersonalMailList_Success tests successful personal mail list retrieval
func TestGetPersonalMailList_Success(t *testing.T) {
	client, mockServer := setupShareTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
			"data": map[string]interface{}{
				"contacts": []MailContact{
					{
						ID:    "contact-1",
						Name:  "John Doe",
						Email: "john@example.com",
					},
					{
						ID:    "contact-2",
						Name:  "Jane Smith",
						Email: "jane@example.com",
					},
				},
				"total": 2,
			},
		},
	})

	contacts, err := fs.GetPersonalMailList(ctx)

	if err != nil {
		t.Errorf("GetPersonalMailList() unexpected error = %v", err)
	}

	if len(contacts) != 2 {
		t.Errorf("GetPersonalMailList() returned %d contacts, want 2", len(contacts))
	}

	if len(contacts) > 0 && contacts[0].Email != "john@example.com" {
		t.Errorf("GetPersonalMailList() first contact email = %s, want john@example.com", contacts[0].Email)
	}
}

// TestGetPersonalMailList_TableDriven tests GetPersonalMailList with various scenarios
func TestGetPersonalMailList_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		mockResponse testutil.MockResponse
		wantCount    int
		wantErr      bool
		errCode      api.ErrorCode
	}{
		{
			name: "success with contacts",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"contacts": []MailContact{
							{ID: "1", Name: "Alice", Email: "alice@example.com"},
							{ID: "2", Name: "Bob", Email: "bob@example.com"},
							{ID: "3", Name: "Charlie", Email: "charlie@example.com"},
						},
						"total": 3,
					},
				},
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name: "empty contact list",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"contacts": []MailContact{},
						"total":    0,
					},
				},
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "authentication failed",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1000,
					"error_msg":  "authentication failed",
				},
			},
			wantErr: true,
			errCode: api.ErrAuthFailed,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: 500,
				Error:      "network error",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupShareTestClient(t)
			defer mockServer.Close()

			ctx := context.Background()
			fs := NewFileStationService(client)

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			contacts, err := fs.GetPersonalMailList(ctx)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetPersonalMailList() expected error but got none")
					return
				}
				if tt.errCode != 0 {
					assertErrorCode(t, err, tt.errCode)
				}
				return
			}

			if err != nil {
				t.Errorf("GetPersonalMailList() unexpected error = %v", err)
				return
			}

			if len(contacts) != tt.wantCount {
				t.Errorf("GetPersonalMailList() returned %d contacts, want %d", len(contacts), tt.wantCount)
			}
		})
	}
}

// TestGetPersonalMailList_NotAuthenticated tests GetPersonalMailList without authentication
func TestGetPersonalMailList_NotAuthenticated(t *testing.T) {
	client, _ := setupShareTestClient(t)
	client.SetSID("")

	ctx := context.Background()
	fs := NewFileStationService(client)

	_, err := fs.GetPersonalMailList(ctx)

	if err == nil {
		t.Error("GetPersonalMailList() expected error when not authenticated")
	}

	assertAuthError(t, err)
}

// TestGetSharedWithMe_Success tests successful shared with me retrieval
func TestGetSharedWithMe_Success(t *testing.T) {
	client, mockServer := setupShareTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
			"data": map[string]interface{}{
				"shares": []ShareLink{
					{
						ID:   "shared-1",
						Name: "Shared Folder",
						URL:  "https://example.com/share/shared1",
					},
					{
						ID:   "shared-2",
						Name: "Another Share",
						URL:  "https://example.com/share/shared2",
					},
				},
				"total": 2,
			},
		},
	})

	shares, err := fs.GetSharedWithMe(ctx)

	if err != nil {
		t.Errorf("GetSharedWithMe() unexpected error = %v", err)
	}

	if len(shares) != 2 {
		t.Errorf("GetSharedWithMe() returned %d shares, want 2", len(shares))
	}

	if len(shares) > 0 && shares[0].ID != "shared-1" {
		t.Errorf("GetSharedWithMe() first share ID = %s, want shared-1", shares[0].ID)
	}
}

// TestGetSharedWithMe_TableDriven tests GetSharedWithMe with various scenarios
func TestGetSharedWithMe_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		mockResponse testutil.MockResponse
		wantCount    int
		wantErr      bool
		errCode      api.ErrorCode
	}{
		{
			name: "success with shares",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"shares": []ShareLink{
							{ID: "1", Name: "Share 1"},
							{ID: "2", Name: "Share 2"},
							{ID: "3", Name: "Share 3"},
						},
						"total": 3,
					},
				},
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name: "no shares",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"shares": []ShareLink{},
						"total":   0,
					},
				},
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "authentication failed",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1000,
					"error_msg":  "authentication failed",
				},
			},
			wantErr: true,
			errCode: api.ErrAuthFailed,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: 500,
				Error:      "network error",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupShareTestClient(t)
			defer mockServer.Close()

			ctx := context.Background()
			fs := NewFileStationService(client)

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			shares, err := fs.GetSharedWithMe(ctx)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetSharedWithMe() expected error but got none")
					return
				}
				if tt.errCode != 0 {
					assertErrorCode(t, err, tt.errCode)
				}
				return
			}

			if err != nil {
				t.Errorf("GetSharedWithMe() unexpected error = %v", err)
				return
			}

			if len(shares) != tt.wantCount {
				t.Errorf("GetSharedWithMe() returned %d shares, want %d", len(shares), tt.wantCount)
			}
		})
	}
}

// TestGetSharedWithMe_NotAuthenticated tests GetSharedWithMe without authentication
func TestGetSharedWithMe_NotAuthenticated(t *testing.T) {
	client, _ := setupShareTestClient(t)
	client.SetSID("")

	ctx := context.Background()
	fs := NewFileStationService(client)

	_, err := fs.GetSharedWithMe(ctx)

	if err == nil {
		t.Error("GetSharedWithMe() expected error when not authenticated")
	}

	assertAuthError(t, err)
}

// TestGetShareLinkInfo_Success tests successful share link info retrieval
func TestGetShareLinkInfo_Success(t *testing.T) {
	client, mockServer := setupShareTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
			"data": ShareLink{
				ID:        "share-123",
				Name:      "My Share",
				URL:       "https://example.com/share/test",
				Expires:   time.Now().Add(24 * time.Hour),
				Password:  true,
				Writeable: false,
			},
		},
	})

	info, err := fs.GetShareLinkInfo(ctx, "my-share")

	if err != nil {
		t.Errorf("GetShareLinkInfo() unexpected error = %v", err)
	}

	if info == nil {
		t.Fatal("GetShareLinkInfo() returned nil info")
	}

	if info.ID != "share-123" {
		t.Errorf("GetShareLinkInfo() ID = %s, want share-123", info.ID)
	}
}

// TestGetShareLinkInfo_TableDriven tests GetShareLinkInfo with various scenarios
func TestGetShareLinkInfo_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		shareName    string
		mockResponse testutil.MockResponse
		wantErr      bool
		errCode      api.ErrorCode
	}{
		{
			name:      "success",
			shareName: "my-share",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": ShareLink{
						ID:   "share-1",
						Name: "My Share",
					},
				},
			},
			wantErr: false,
		},
		{
			name:      "share not found",
			shareName: "nonexistent",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "share not found",
				},
			},
			wantErr: true,
			errCode: api.ErrNotFound,
		},
		{
			name:      "invalid share name",
			shareName: "",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1003,
					"error_msg":  "invalid parameter",
				},
			},
			wantErr: true,
			errCode: api.ErrInvalidParams,
		},
		{
			name:      "network error",
			shareName: "network-share",
			mockResponse: testutil.MockResponse{
				StatusCode: 500,
				Error:      "network error",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupShareTestClient(t)
			defer mockServer.Close()

			ctx := context.Background()
			fs := NewFileStationService(client)

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			info, err := fs.GetShareLinkInfo(ctx, tt.shareName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetShareLinkInfo() expected error but got none")
					return
				}
				if tt.errCode != 0 {
					assertErrorCode(t, err, tt.errCode)
				}
				return
			}

			if err != nil {
				t.Errorf("GetShareLinkInfo() unexpected error = %v", err)
				return
			}

			if info == nil {
				t.Error("GetShareLinkInfo() returned nil info")
			}
		})
	}
}

// TestGetShareLinkInfo_NotAuthenticated tests GetShareLinkInfo without authentication
func TestGetShareLinkInfo_NotAuthenticated(t *testing.T) {
	client, _ := setupShareTestClient(t)
	client.SetSID("")

	ctx := context.Background()
	fs := NewFileStationService(client)

	_, err := fs.GetShareLinkInfo(ctx, "test-share")

	if err == nil {
		t.Error("GetShareLinkInfo() expected error when not authenticated")
	}

	assertAuthError(t, err)
}

// TestSetShareNasUser_Success tests successful NAS user share setting
func TestSetShareNasUser_Success(t *testing.T) {
	client, mockServer := setupShareTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
		},
	})

	options := &SetShareNasUserOptions{
		ShareName: "my-share",
		Username:  "nasuser",
		Access:    "read",
	}

	err := fs.SetShareNasUser(ctx, options)

	if err != nil {
		t.Errorf("SetShareNasUser() unexpected error = %v", err)
	}
}

// TestSetShareNasUser_TableDriven tests SetShareNasUser with various scenarios
func TestSetShareNasUser_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		options      *SetShareNasUserOptions
		mockResponse testutil.MockResponse
		wantErr      bool
		errCode      api.ErrorCode
	}{
		{
			name: "set read access",
			options: &SetShareNasUserOptions{
				ShareName: "read-share",
				Username:  "reader",
				Access:    "read",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
				},
			},
			wantErr: false,
		},
		{
			name: "set write access",
			options: &SetShareNasUserOptions{
				ShareName: "write-share",
				Username:  "writer",
				Access:    "write",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
				},
			},
			wantErr: false,
		},
		{
			name: "set full access",
			options: &SetShareNasUserOptions{
				ShareName: "full-share",
				Username:  "admin",
				Access:    "full",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
				},
			},
			wantErr: false,
		},
		{
			name: "share not found",
			options: &SetShareNasUserOptions{
				ShareName: "nonexistent",
				Username:  "user",
				Access:    "read",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "share not found",
				},
			},
			wantErr: true,
			errCode: api.ErrNotFound,
		},
		{
			name: "user not found",
			options: &SetShareNasUserOptions{
				ShareName: "my-share",
				Username:  "nonexistent-user",
				Access:    "read",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "user not found",
				},
			},
			wantErr: true,
			errCode: api.ErrNotFound,
		},
		{
			name: "network error",
			options: &SetShareNasUserOptions{
				ShareName: "network-share",
				Username:  "user",
				Access:    "read",
			},
			mockResponse: testutil.MockResponse{
				StatusCode: 500,
				Error:      "network error",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupShareTestClient(t)
			defer mockServer.Close()

			ctx := context.Background()
			fs := NewFileStationService(client)

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			err := fs.SetShareNasUser(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SetShareNasUser() expected error but got none")
					return
				}
				if tt.errCode != 0 {
					assertErrorCode(t, err, tt.errCode)
				}
				return
			}

			if err != nil {
				t.Errorf("SetShareNasUser() unexpected error = %v", err)
			}
		})
	}
}

// TestSetShareNasUser_NotAuthenticated tests SetShareNasUser without authentication
func TestSetShareNasUser_NotAuthenticated(t *testing.T) {
	client, _ := setupShareTestClient(t)
	client.SetSID("")

	ctx := context.Background()
	fs := NewFileStationService(client)

	options := &SetShareNasUserOptions{
		ShareName: "my-share",
		Username:  "user",
		Access:    "read",
	}

	err := fs.SetShareNasUser(ctx, options)

	if err == nil {
		t.Error("SetShareNasUser() expected error when not authenticated")
	}

	assertAuthError(t, err)
}

// TestListShareLinks_Success tests successful share links listing
func TestListShareLinks_Success(t *testing.T) {
	client, mockServer := setupShareTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	mockServer.SetResponse("GET", "/filestation/share.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: map[string]interface{}{
			"success": 1,
			"data": map[string]interface{}{
				"shares": []ShareLink{
					{
						ID:   "share-1",
						Name: "Share 1",
						URL:  "https://example.com/share/1",
					},
					{
						ID:   "share-2",
						Name: "Share 2",
						URL:  "https://example.com/share/2",
					},
				},
				"total": 2,
			},
		},
	})

	links, err := fs.ListShareLinks(ctx)

	if err != nil {
		t.Errorf("ListShareLinks() unexpected error = %v", err)
	}

	if len(links) != 2 {
		t.Errorf("ListShareLinks() returned %d links, want 2", len(links))
	}

	if len(links) > 0 && links[0].ID != "share-1" {
		t.Errorf("ListShareLinks() first link ID = %s, want share-1", links[0].ID)
	}
}

// TestListShareLinks_TableDriven tests ListShareLinks with various scenarios
func TestListShareLinks_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		mockResponse testutil.MockResponse
		wantCount    int
		wantErr      bool
		errCode      api.ErrorCode
	}{
		{
			name: "success with links",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"shares": []ShareLink{
							{ID: "1", Name: "Link 1"},
							{ID: "2", Name: "Link 2"},
							{ID: "3", Name: "Link 3"},
						},
						"total": 3,
					},
				},
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name: "empty list",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"shares": []ShareLink{},
						"total":   0,
					},
				},
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "authentication failed",
			mockResponse: testutil.MockResponse{
				StatusCode: 200,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1000,
					"error_msg":  "authentication failed",
				},
			},
			wantErr: true,
			errCode: api.ErrAuthFailed,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: 500,
				Error:      "network error",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupShareTestClient(t)
			defer mockServer.Close()

			ctx := context.Background()
			fs := NewFileStationService(client)

			mockServer.SetResponse("GET", "/filestation/share.cgi", tt.mockResponse)

			links, err := fs.ListShareLinks(ctx)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ListShareLinks() expected error but got none")
					return
				}
				if tt.errCode != 0 {
					assertErrorCode(t, err, tt.errCode)
				}
				return
			}

			if err != nil {
				t.Errorf("ListShareLinks() unexpected error = %v", err)
				return
			}

			if len(links) != tt.wantCount {
				t.Errorf("ListShareLinks() returned %d links, want %d", len(links), tt.wantCount)
			}
		})
	}
}

// TestListShareLinks_NotAuthenticated tests ListShareLinks without authentication
func TestListShareLinks_NotAuthenticated(t *testing.T) {
	client, _ := setupShareTestClient(t)
	client.SetSID("")

	ctx := context.Background()
	fs := NewFileStationService(client)

	_, err := fs.ListShareLinks(ctx)

	if err == nil {
		t.Error("ListShareLinks() expected error when not authenticated")
	}

	assertAuthError(t, err)
}
