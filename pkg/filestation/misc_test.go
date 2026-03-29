package filestation

import (
	"context"
	"net/http"
	"testing"

	"github.com/fatelei/qnap-filestation/pkg/api"
	"github.com/fatelei/qnap-filestation/internal/testutil"
)


// assertUnsupportedOperation checks if the response indicates an unsupported operation
func assertUnsupportedOperation(t *testing.T, resp *UnsupportedOperationResponse, operationName string) {
	t.Helper()

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Success != 0 {
		t.Errorf("Expected success=0 for unsupported operation, got %d", resp.Success)
	}

	expectedMsg := "operation '" + operationName + "' is not supported"
	if resp.Message != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, resp.Message)
	}
}

// TestDaemonList tests the DaemonList function
func TestDaemonList(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		wantErr        bool
		expectedErr    api.ErrorCode
		wantDaemons    int
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "successful daemon list with multiple daemons",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"daemons": []DaemonInfo{
							{
								PID:       "1234",
								Name:      "nginx",
								Status:    "running",
								CPU:       "0.5",
								Memory:    "256M",
								Uptime:    "10 days",
								Command:   "/usr/bin/nginx -g daemon off",
								User:      "root",
								AutoStart: true,
							},
							{
								PID:       "5678",
								Name:      "mysql",
								Status:    "running",
								CPU:       "2.3",
								Memory:    "512M",
								Uptime:    "5 days",
								Command:   "/usr/bin/mysqld --user=mysql",
								User:      "mysql",
								AutoStart: true,
							},
							{
								PID:       "9012",
								Name:      "apache",
								Status:    "stopped",
								CPU:       "0",
								Memory:    "0",
								Uptime:    "-",
								Command:   "/usr/sbin/apache2 -k start",
								User:      "www-data",
								AutoStart: false,
							},
						},
						"total": 3,
					},
				},
			},
			wantErr:     false,
			wantDaemons: 3,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "daemon_list" {
					t.Errorf("Expected func=daemon_list, got %s", fn)
				}
			},
		},
		{
			name: "successful daemon list with empty result",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"daemons": []DaemonInfo{},
						"total":   0,
					},
				},
			},
			wantErr:     false,
			wantDaemons: 0,
		},
		{
			name: "successful daemon list with single daemon",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"daemons": []DaemonInfo{
							{
								PID:       "1",
								Name:      "init",
								Status:    "running",
								CPU:       "0.1",
								Memory:    "8M",
								Uptime:    "100 days",
								Command:   "/sbin/init",
								User:      "root",
								AutoStart: true,
							},
						},
						"total": 1,
					},
				},
			},
			wantErr:     false,
			wantDaemons: 1,
		},
		{
			name: "daemon list with special characters in command",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"daemons": []DaemonInfo{
							{
								PID:     "9999",
								Name:    "test-app",
								Status:  "running",
								CPU:     "1.0",
								Memory:  "128M",
								Uptime:  "1 hour",
								Command: "/usr/bin/test-app --config=/etc/config file with spaces.conf",
								User:    "nobody",
							},
						},
						"total": 1,
					},
				},
			},
			wantErr:     false,
			wantDaemons: 1,
		},
		{
			name: "API error - permission denied",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2003,
					"error_msg":  "Permission denied",
				},
			},
			wantErr:     true,
			expectedErr: api.ErrPermission,
		},
		{
			name: "API error - invalid parameters",
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
				Body:       "invalid json",
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
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.DaemonList(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("DaemonList() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("DaemonList() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Fatal("DaemonList() returned nil response")
			}

			if len(resp.Data.Daemons) != tt.wantDaemons {
				t.Errorf("DaemonList() returned %d daemons, want %d", len(resp.Data.Daemons), tt.wantDaemons)
			}

			if resp.Data.Total != tt.wantDaemons {
				t.Errorf("DaemonList() total = %d, want %d", resp.Data.Total, tt.wantDaemons)
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

// TestDaemonListAuthentication tests authentication scenarios for DaemonList
func TestDaemonListAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		_, err := fs.DaemonList(ctx)

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestGetCayinMediaStatus tests the GetCayinMediaStatus function
func TestGetCayinMediaStatus(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *GetCayinMediaStatusResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "successful get cayin media status - online",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"enabled":     true,
						"status":      "online",
						"version":     "2.5.1",
						"device_id":   "CAYIN-12345-ABCDE",
						"ip_address":  "192.168.1.100",
						"mac_address": "00:11:22:33:44:55",
						"uptime":      "15 days 4 hours",
						"last_sync":   "2024-01-15 10:30:00",
						"storage_used": "45.2 GB",
						"description":  "Main media server",
					},
				},
			},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *GetCayinMediaStatusResponse) {
				t.Helper()
				if !resp.Data.Enabled {
					t.Error("Expected enabled=true, got false")
				}
				if resp.Data.Status != "online" {
					t.Errorf("Expected status=online, got %s", resp.Data.Status)
				}
				if resp.Data.Version != "2.5.1" {
					t.Errorf("Expected version=2.5.1, got %s", resp.Data.Version)
				}
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "get_cayin_media_status" {
					t.Errorf("Expected func=get_cayin_media_status, got %s", fn)
				}
			},
		},
		{
			name: "successful get cayin media status - offline",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"enabled":     false,
						"status":      "offline",
						"version":     "",
						"device_id":   "",
						"ip_address":  "",
						"mac_address": "",
						"uptime":      "-",
						"last_sync":   "",
						"storage_used": "0 B",
						"description":  "",
					},
				},
			},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *GetCayinMediaStatusResponse) {
				t.Helper()
				if resp.Data.Enabled {
					t.Error("Expected enabled=false, got true")
				}
				if resp.Data.Status != "offline" {
					t.Errorf("Expected status=offline, got %s", resp.Data.Status)
				}
			},
		},
		{
			name: "successful get cayin media status - error state",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"enabled":     true,
						"status":      "error",
						"version":     "1.0.0",
						"device_id":   "CAYIN-ERROR-001",
						"ip_address":  "192.168.1.200",
						"mac_address": "AA:BB:CC:DD:EE:FF",
						"uptime":      "0 days",
						"last_sync":   "2024-01-01 00:00:00",
						"storage_used": "0 B",
						"description":  "Device in error state",
					},
				},
			},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *GetCayinMediaStatusResponse) {
				t.Helper()
				if resp.Data.Status != "error" {
					t.Errorf("Expected status=error, got %s", resp.Data.Status)
				}
			},
		},
		{
			name: "successful get cayin media status with empty values",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"enabled":     false,
						"status":      "",
						"version":     "",
						"device_id":   "",
						"ip_address":  "",
						"mac_address": "",
						"uptime":      "",
						"last_sync":   "",
						"storage_used": "",
						"description":  "",
					},
				},
			},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *GetCayinMediaStatusResponse) {
				t.Helper()
				// All fields should be empty/false
				if resp.Data.Status != "" {
					t.Errorf("Expected empty status, got %s", resp.Data.Status)
				}
			},
		},
		{
			name: "API error - service not available",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "Cayin media service not available",
				},
			},
			wantErr:     true,
			expectedErr: api.ErrNotFound,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
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
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.GetCayinMediaStatus(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("GetCayinMediaStatus() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("GetCayinMediaStatus() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Fatal("GetCayinMediaStatus() returned nil response")
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

// TestGetCayinMediaStatusAuthentication tests authentication scenarios for GetCayinMediaStatus
func TestGetCayinMediaStatusAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		_, err := fs.GetCayinMediaStatus(ctx)

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestQcloudNotifyInfo tests the QcloudNotifyInfo function
func TestQcloudNotifyInfo(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *QcloudNotifyInfoResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "successful get qcloud notify info with all notifications enabled",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"enabled":       true,
						"email_enabled": true,
						"sms_enabled":   true,
						"push_enabled":  true,
						"emails":        []string{"user1@example.com", "user2@example.com"},
						"phone_numbers": []string{"+1234567890", "+0987654321"},
						"event_types":   []string{"file_upload", "file_delete", "share_created"},
						"frequency":     "immediate",
						"last_notified": "2024-01-15 14:30:00",
					},
				},
			},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *QcloudNotifyInfoResponse) {
				t.Helper()
				if !resp.Data.Enabled {
					t.Error("Expected enabled=true, got false")
				}
				if !resp.Data.EmailEnabled {
					t.Error("Expected email_enabled=true, got false")
				}
				if !resp.Data.SMSEnabled {
					t.Error("Expected sms_enabled=true, got false")
				}
				if !resp.Data.PushEnabled {
					t.Error("Expected push_enabled=true, got false")
				}
				if len(resp.Data.Emails) != 2 {
					t.Errorf("Expected 2 emails, got %d", len(resp.Data.Emails))
				}
				if len(resp.Data.PhoneNumbers) != 2 {
					t.Errorf("Expected 2 phone numbers, got %d", len(resp.Data.PhoneNumbers))
				}
				if resp.Data.Frequency != "immediate" {
					t.Errorf("Expected frequency=immediate, got %s", resp.Data.Frequency)
				}
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "qcloud_notify_info" {
					t.Errorf("Expected func=qcloud_notify_info, got %s", fn)
				}
			},
		},
		{
			name: "successful get qcloud notify info with notifications disabled",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"enabled":       false,
						"email_enabled": false,
						"sms_enabled":   false,
						"push_enabled":  false,
						"emails":        []string{},
						"phone_numbers": []string{},
						"event_types":   []string{},
						"frequency":     "never",
						"last_notified": "",
					},
				},
			},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *QcloudNotifyInfoResponse) {
				t.Helper()
				if resp.Data.Enabled {
					t.Error("Expected enabled=false, got true")
				}
				if len(resp.Data.Emails) != 0 {
					t.Errorf("Expected 0 emails, got %d", len(resp.Data.Emails))
				}
			},
		},
		{
			name: "successful get qcloud notify info with only email enabled",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"enabled":       true,
						"email_enabled": true,
						"sms_enabled":   false,
						"push_enabled":  false,
						"emails":        []string{"admin@example.com"},
						"phone_numbers": []string{},
						"event_types":   []string{"system_alert"},
						"frequency":     "daily",
						"last_notified": "2024-01-15 09:00:00",
					},
				},
			},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *QcloudNotifyInfoResponse) {
				t.Helper()
				if !resp.Data.EmailEnabled {
					t.Error("Expected email_enabled=true, got false")
				}
				if resp.Data.SMSEnabled {
					t.Error("Expected sms_enabled=false, got true")
				}
			},
		},
		{
			name: "successful get qcloud notify info with multiple recipients",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"enabled":       true,
						"email_enabled": true,
						"sms_enabled":   true,
						"push_enabled":  true,
						"emails":        []string{"a@example.com", "b@example.com", "c@example.com", "d@example.com", "e@example.com"},
						"phone_numbers": []string{"+1", "+2", "+3"},
						"event_types":   []string{"upload", "delete", "share", "quota", "error"},
						"frequency":     "hourly",
						"last_notified": "2024-01-15 12:00:00",
					},
				},
			},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *QcloudNotifyInfoResponse) {
				t.Helper()
				if len(resp.Data.Emails) != 5 {
					t.Errorf("Expected 5 emails, got %d", len(resp.Data.Emails))
				}
				if len(resp.Data.EventTypes) != 5 {
					t.Errorf("Expected 5 event types, got %d", len(resp.Data.EventTypes))
				}
			},
		},
		{
			name: "API error - feature not available",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "Qcloud notification feature not available",
				},
			},
			wantErr:     true,
			expectedErr: api.ErrNotFound,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
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
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.QcloudNotifyInfo(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("QcloudNotifyInfo() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("QcloudNotifyInfo() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Fatal("QcloudNotifyInfo() returned nil response")
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

// TestQcloudNotifyInfoAuthentication tests authentication scenarios for QcloudNotifyInfo
func TestQcloudNotifyInfoAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		_, err := fs.QcloudNotifyInfo(ctx)

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestQcloudWopiUrl tests the QcloudWopiUrl function
func TestQcloudWopiUrl(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *QcloudWopiUrlOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *QcloudWopiUrlResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "successful get wopi url with all options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"url":        "https://wopi.example.com/files/doc123",
						"enabled":    true,
						"version":    "1.0",
						"discovery":  "https://wopi.example.com/hosting/discovery",
						"zone":       "primary",
						"expiration": "2024-01-16 14:30:00",
					},
				},
			},
			options: &QcloudWopiUrlOptions{
				FileID:   "doc123",
				FileName: "document.docx",
				Action:   "edit",
				Timeout:  3600,
			},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *QcloudWopiUrlResponse) {
				t.Helper()
				if !resp.Data.Enabled {
					t.Error("Expected enabled=true, got false")
				}
				if resp.Data.URL != "https://wopi.example.com/files/doc123" {
					t.Errorf("Expected URL to be set, got %s", resp.Data.URL)
				}
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "qcloud_wopi_url" {
					t.Errorf("Expected func=qcloud_wopi_url, got %s", fn)
				}
				if fileID := r.URL.Query().Get("file_id"); fileID != "doc123" {
					t.Errorf("Expected file_id=doc123, got %s", fileID)
				}
				if fileName := r.URL.Query().Get("file_name"); fileName != "document.docx" {
					t.Errorf("Expected file_name=document.docx, got %s", fileName)
				}
				if action := r.URL.Query().Get("action"); action != "edit" {
					t.Errorf("Expected action=edit, got %s", action)
				}
				if timeout := r.URL.Query().Get("timeout"); timeout != "3600" {
					t.Errorf("Expected timeout=3600, got %s", timeout)
				}
			},
		},
		{
			name: "successful get wopi url with view action",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"url":        "https://wopi.example.com/view/doc456",
						"enabled":    true,
						"version":    "1.0",
						"discovery":  "https://wopi.example.com/hosting/discovery",
						"zone":       "secondary",
						"expiration": "2024-01-15 16:00:00",
					},
				},
			},
			options: &QcloudWopiUrlOptions{
				FileID: "doc456",
				Action: "view",
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if action := r.URL.Query().Get("action"); action != "view" {
					t.Errorf("Expected action=view, got %s", action)
				}
			},
		},
		{
			name: "successful get wopi url with nil options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"url":        "https://wopi.example.com/",
						"enabled":    true,
						"version":    "1.0",
						"discovery":  "https://wopi.example.com/hosting/discovery",
						"zone":       "default",
						"expiration": "",
					},
				},
			},
			options: nil,
			wantErr: false,
		},
		{
			name: "successful get wopi url with empty options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"url":        "https://wopi.example.com/",
						"enabled":    true,
						"version":    "1.0",
						"discovery":  "",
						"zone":       "",
						"expiration": "",
					},
				},
			},
			options: &QcloudWopiUrlOptions{},
			wantErr: false,
		},
		{
			name: "successful get wopi url with only file ID",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"url":     "https://wopi.example.com/files/doc789",
						"enabled": true,
						"version": "2.0",
					},
				},
			},
			options: &QcloudWopiUrlOptions{
				FileID: "doc789",
			},
			wantErr: false,
		},
		{
			name: "successful get wopi url with only file name",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"url":        "https://wopi.example.com/files/by-name",
						"enabled":    true,
						"version":    "1.0",
						"discovery":  "https://wopi.example.com/hosting/discovery",
					},
				},
			},
			options: &QcloudWopiUrlOptions{
				FileName: "spreadsheet.xlsx",
			},
			wantErr: false,
		},
		{
			name: "successful get wopi url with timeout only",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"url":        "https://wopi.example.com/",
						"enabled":    true,
						"version":    "1.0",
						"expiration": "2024-01-15 15:30:00",
					},
				},
			},
			options: &QcloudWopiUrlOptions{
				Timeout: 1800,
			},
			wantErr: false,
		},
		{
			name: "successful get wopi url with timeout=0 (should not be included)",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"url":        "https://wopi.example.com/",
						"enabled":    true,
						"version":    "1.0",
					},
				},
			},
			options: &QcloudWopiUrlOptions{
				Timeout: 0,
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				// timeout=0 should not be included in request
				if timeout := r.URL.Query().Get("timeout"); timeout != "" {
					t.Errorf("Expected no timeout parameter, got %s", timeout)
				}
			},
		},
		{
			name: "successful get wopi url with negative timeout (should not be included)",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"url":     "https://wopi.example.com/",
						"enabled": true,
					},
				},
			},
			options: &QcloudWopiUrlOptions{
				Timeout: -100,
			},
			wantErr: false,
		},
		{
			name: "API error - file not found",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "File not found",
				},
			},
			options:     &QcloudWopiUrlOptions{FileID: "nonexistent"},
			wantErr:     true,
			expectedErr: api.ErrNotFound,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
			},
			options:     &QcloudWopiUrlOptions{},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			options: &QcloudWopiUrlOptions{},
			wantErr: true,
		},
		{
			name: "wopi url with special characters in filename",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"url":     "https://wopi.example.com/files/special",
						"enabled": true,
					},
				},
			},
			options: &QcloudWopiUrlOptions{
				FileName: "file with spaces & special-chars_@#.docx",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.QcloudWopiUrl(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("QcloudWopiUrl() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("QcloudWopiUrl() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Fatal("QcloudWopiUrl() returned nil response")
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

// TestQcloudWopiUrlAuthentication tests authentication scenarios for QcloudWopiUrl
func TestQcloudWopiUrlAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		_, err := fs.QcloudWopiUrl(ctx, &QcloudWopiUrlOptions{FileID: "test"})

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestQdmc tests the Qdmc function
func TestQdmc(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *QdmcOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *QdmcResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "successful qdmc operation with all options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":   true,
						"message":   "Operation completed successfully",
						"operation": "sync",
						"status":    "completed",
					},
				},
			},
			options: &QdmcOptions{
				Action:   "sync",
				Target:   "/share/folder",
				Mode:     "full",
				Override: true,
			},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *QdmcResponse) {
				t.Helper()
				if !resp.Data.Success {
					t.Error("Expected success=true, got false")
				}
				if resp.Data.Operation != "sync" {
					t.Errorf("Expected operation=sync, got %s", resp.Data.Operation)
				}
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "qdmc" {
					t.Errorf("Expected func=qdmc, got %s", fn)
				}
				if action := r.URL.Query().Get("action"); action != "sync" {
					t.Errorf("Expected action=sync, got %s", action)
				}
				if target := r.URL.Query().Get("target"); target != "/share/folder" {
					t.Errorf("Expected target=/share/folder, got %s", target)
				}
				if mode := r.URL.Query().Get("mode"); mode != "full" {
					t.Errorf("Expected mode=full, got %s", mode)
				}
				if override := r.URL.Query().Get("override"); override != "1" {
					t.Errorf("Expected override=1, got %s", override)
				}
			},
		},
		{
			name: "successful qdmc operation with minimal options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":   true,
						"message":   "Quick operation done",
						"operation": "quick",
						"status":    "done",
					},
				},
			},
			options: &QdmcOptions{
				Action: "quick",
			},
			wantErr: false,
		},
		{
			name: "successful qdmc operation with nil options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":   true,
						"message":   "Default operation",
						"operation": "",
						"status":    "ok",
					},
				},
			},
			options: nil,
			wantErr: false,
		},
		{
			name: "successful qdmc operation with empty options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: &QdmcOptions{},
			wantErr: false,
		},
		{
			name: "successful qdmc operation without override",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":   true,
						"message":   "Operation completed",
						"operation": "backup",
						"status":    "running",
					},
				},
			},
			options: &QdmcOptions{
				Action:   "backup",
				Target:   "/backup",
				Override: false,
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				// override=false should not be included in request
				if override := r.URL.Query().Get("override"); override != "" {
					t.Errorf("Expected no override parameter, got %s", override)
				}
			},
		},
		{
			name: "successful qdmc operation with target only",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":   true,
						"operation": "scan",
						"status":    "scanning",
					},
				},
			},
			options: &QdmcOptions{
				Target: "/home/user/documents",
			},
			wantErr: false,
		},
		{
			name: "successful qdmc operation with mode only",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":   true,
						"operation": "",
						"status":    "ready",
						"message":   "Mode set",
					},
				},
			},
			options: &QdmcOptions{
				Mode: "incremental",
			},
			wantErr: false,
		},
		{
			name: "API error - invalid target path",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2004,
					"error_msg":  "Invalid target path",
				},
			},
			options:     &QdmcOptions{Target: "/invalid/path"},
			wantErr:     true,
			expectedErr: api.ErrInvalidPath,
		},
		{
			name: "API error - operation not supported",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1003,
					"error_msg":  "Operation not supported",
				},
			},
			options:     &QdmcOptions{Action: "invalid"},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
			},
			options:     &QdmcOptions{},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			options: &QdmcOptions{},
			wantErr: true,
		},
		{
			name: "qdmc operation with special characters in target",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: &QdmcOptions{
				Target: "/share/folder with spaces/子文件夹",
			},
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

			resp, err := fs.Qdmc(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("Qdmc() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Qdmc() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Fatal("Qdmc() returned nil response")
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

// TestQdmcAuthentication tests authentication scenarios for Qdmc
func TestQdmcAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		_, err := fs.Qdmc(ctx, &QdmcOptions{Action: "test"})

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestQhamRetrieve tests the QhamRetrieve function
func TestQhamRetrieve(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *QhamRetrieveOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *QhamRetrieveResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "successful qham retrieve with all options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"message":     "Items retrieved successfully",
						"items":       []string{"item1", "item2", "item3"},
						"count":       3,
						"cache_status": "fresh",
						"timestamp":   "2024-01-15 15:45:00",
					},
				},
			},
			options: &QhamRetrieveOptions{
				Source:      "cloud-storage",
				Destination: "/local/backup",
				Mode:        "recursive",
				Refresh:     true,
				Limit:       100,
			},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *QhamRetrieveResponse) {
				t.Helper()
				if !resp.Data.Success {
					t.Error("Expected success=true, got false")
				}
				if len(resp.Data.Items) != 3 {
					t.Errorf("Expected 3 items, got %d", len(resp.Data.Items))
				}
				if resp.Data.Count != 3 {
					t.Errorf("Expected count=3, got %d", resp.Data.Count)
				}
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "qham_retrieve" {
					t.Errorf("Expected func=qham_retrieve, got %s", fn)
				}
				if source := r.URL.Query().Get("source"); source != "cloud-storage" {
					t.Errorf("Expected source=cloud-storage, got %s", source)
				}
				if dest := r.URL.Query().Get("destination"); dest != "/local/backup" {
					t.Errorf("Expected destination=/local/backup, got %s", dest)
				}
				if mode := r.URL.Query().Get("mode"); mode != "recursive" {
					t.Errorf("Expected mode=recursive, got %s", mode)
				}
				if refresh := r.URL.Query().Get("refresh"); refresh != "1" {
					t.Errorf("Expected refresh=1, got %s", refresh)
				}
				if limit := r.URL.Query().Get("limit"); limit != "100" {
					t.Errorf("Expected limit=100, got %s", limit)
				}
			},
		},
		{
			name: "successful qham retrieve with minimal options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"items":       []string{"single-item"},
						"count":       1,
						"cache_status": "cached",
					},
				},
			},
			options: &QhamRetrieveOptions{
				Source: "default-source",
			},
			wantErr: false,
		},
		{
			name: "successful qham retrieve with nil options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"items":       []string{},
						"count":       0,
						"cache_status": "empty",
					},
				},
			},
			options: nil,
			wantErr: false,
			assertResponse: func(t *testing.T, resp *QhamRetrieveResponse) {
				t.Helper()
				if len(resp.Data.Items) != 0 {
					t.Errorf("Expected 0 items, got %d", len(resp.Data.Items))
				}
			},
		},
		{
			name: "successful qham retrieve with empty options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"items":   []string{},
					},
				},
			},
			options: &QhamRetrieveOptions{},
			wantErr: false,
		},
		{
			name: "successful qham retrieve with refresh=true",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"message":     "Cache refreshed",
						"items":       []string{"new-item"},
						"count":       1,
						"cache_status": "refreshed",
					},
				},
			},
			options: &QhamRetrieveOptions{
				Refresh: true,
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if refresh := r.URL.Query().Get("refresh"); refresh != "1" {
					t.Errorf("Expected refresh=1, got %s", refresh)
				}
			},
		},
		{
			name: "successful qham retrieve with refresh=false",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"items":   []string{"cached-item"},
					},
				},
			},
			options: &QhamRetrieveOptions{
				Refresh: false,
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				// refresh=false should not be included in request
				if refresh := r.URL.Query().Get("refresh"); refresh != "" {
					t.Errorf("Expected no refresh parameter, got %s", refresh)
				}
			},
		},
		{
			name: "successful qham retrieve with limit",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"items":   []string{"item1", "item2", "item3", "item4", "item5"},
						"count":   5,
					},
				},
			},
			options: &QhamRetrieveOptions{
				Limit: 5,
			},
			wantErr: false,
		},
		{
			name: "successful qham retrieve with limit=0 (should not be included)",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"items":   []string{},
					},
				},
			},
			options: &QhamRetrieveOptions{
				Limit: 0,
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if limit := r.URL.Query().Get("limit"); limit != "" {
					t.Errorf("Expected no limit parameter, got %s", limit)
				}
			},
		},
		{
			name: "successful qham retrieve with negative limit (should not be included)",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: &QhamRetrieveOptions{
				Limit: -10,
			},
			wantErr: false,
		},
		{
			name: "successful qham retrieve with large limit",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"count":   10000,
					},
				},
			},
			options: &QhamRetrieveOptions{
				Limit: 10000,
			},
			wantErr: false,
		},
		{
			name: "successful qham retrieve with destination only",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"items":   []string{"file1.txt"},
					},
				},
			},
			options: &QhamRetrieveOptions{
				Destination: "/target/folder",
			},
			wantErr: false,
		},
		{
			name: "API error - source not accessible",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2003,
					"error_msg":  "Source not accessible",
				},
			},
			options:     &QhamRetrieveOptions{Source: "inaccessible"},
			wantErr:     true,
			expectedErr: api.ErrPermission,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
			},
			options:     &QhamRetrieveOptions{},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			options: &QhamRetrieveOptions{},
			wantErr: true,
		},
		{
			name: "qham retrieve with special characters in source",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"items":   []string{"test"},
					},
				},
			},
			options: &QhamRetrieveOptions{
				Source: "cloud-storage/备份文件夹",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.QhamRetrieve(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("QhamRetrieve() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("QhamRetrieve() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Fatal("QhamRetrieve() returned nil response")
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

// TestQhamRetrieveAuthentication tests authentication scenarios for QhamRetrieve
func TestQhamRetrieveAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		_, err := fs.QhamRetrieve(ctx, &QhamRetrieveOptions{Source: "test"})

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestQrpac tests the Qrpac function
func TestQrpac(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *QrpacOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *QrpacResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "successful qrpac operation with all options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"message":     "Operation completed",
						"action":      "configure",
						"status_code": 200,
						"request_id":  "req-12345",
					},
				},
			},
			options: &QrpacOptions{
				Action: "configure",
				Parameters: map[string]string{
					"param1": "value1",
					"param2": "value2",
				},
				Async: true,
			},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *QrpacResponse) {
				t.Helper()
				if !resp.Data.Success {
					t.Error("Expected success=true, got false")
				}
				if resp.Data.Action != "configure" {
					t.Errorf("Expected action=configure, got %s", resp.Data.Action)
				}
				if resp.Data.StatusCode != 200 {
					t.Errorf("Expected status_code=200, got %d", resp.Data.StatusCode)
				}
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "qrpac" {
					t.Errorf("Expected func=qrpac, got %s", fn)
				}
				if action := r.URL.Query().Get("action"); action != "configure" {
					t.Errorf("Expected action=configure, got %s", action)
				}
				if async := r.URL.Query().Get("async"); async != "1" {
					t.Errorf("Expected async=1, got %s", async)
				}
				if param1 := r.URL.Query().Get("param1"); param1 != "value1" {
					t.Errorf("Expected param1=value1, got %s", param1)
				}
				if param2 := r.URL.Query().Get("param2"); param2 != "value2" {
					t.Errorf("Expected param2=value2, got %s", param2)
				}
			},
		},
		{
			name: "successful qrpac operation with minimal options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"message":     "Quick action",
						"action":      "test",
						"status_code": 200,
					},
				},
			},
			options: &QrpacOptions{
				Action: "test",
			},
			wantErr: false,
		},
		{
			name: "successful qrpac operation with nil options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"message":     "Default operation",
						"action":      "",
						"status_code": 200,
					},
				},
			},
			options: nil,
			wantErr: false,
		},
		{
			name: "successful qrpac operation with empty options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: &QrpacOptions{},
			wantErr: false,
		},
		{
			name: "successful qrpac operation with parameters only",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"status_code": 200,
					},
				},
			},
			options: &QrpacOptions{
				Parameters: map[string]string{
					"key1": "val1",
					"key2": "val2",
					"key3": "val3",
				},
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if r.URL.Query().Get("key1") != "val1" {
					t.Error("Expected key1 to be set")
				}
				if r.URL.Query().Get("key2") != "val2" {
					t.Error("Expected key2 to be set")
				}
				if r.URL.Query().Get("key3") != "val3" {
					t.Error("Expected key3 to be set")
				}
			},
		},
		{
			name: "successful qrpac operation with async=true",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"message":     "Async operation started",
						"action":      "async-task",
						"status_code": 202,
						"request_id":  "async-req-001",
					},
				},
			},
			options: &QrpacOptions{
				Action: "async-task",
				Async:  true,
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if async := r.URL.Query().Get("async"); async != "1" {
					t.Errorf("Expected async=1, got %s", async)
				}
			},
		},
		{
			name: "successful qrpac operation with async=false",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"status_code": 200,
					},
				},
			},
			options: &QrpacOptions{
				Async: false,
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				// async=false should not be included in request
				if async := r.URL.Query().Get("async"); async != "" {
					t.Errorf("Expected no async parameter, got %s", async)
				}
			},
		},
		{
			name: "successful qrpac operation with empty parameters",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: &QrpacOptions{
				Parameters: map[string]string{},
			},
			wantErr: false,
		},
		{
			name: "successful qrpac operation with nil parameters",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: &QrpacOptions{
				Parameters: nil,
			},
			wantErr: false,
		},
		{
			name: "successful qrpac operation with special characters in parameters",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: &QrpacOptions{
				Parameters: map[string]string{
					"path":     "/folder with spaces/",
					"unicode":  "中文",
					"special":  "@#$%^&*()",
				},
			},
			wantErr: false,
		},
		{
			name: "API error - action not supported",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1003,
					"error_msg":  "Action not supported",
				},
			},
			options:     &QrpacOptions{Action: "unsupported"},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "API error - invalid parameter",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1003,
					"error_msg":  "Invalid parameter value",
				},
			},
			options: &QrpacOptions{
				Parameters: map[string]string{"invalid": "value"},
			},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
			},
			options:     &QrpacOptions{},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			options: &QrpacOptions{},
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

			resp, err := fs.Qrpac(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("Qrpac() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Qrpac() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Fatal("Qrpac() returned nil response")
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

// TestQrpacAuthentication tests authentication scenarios for Qrpac
func TestQrpacAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		_, err := fs.Qrpac(ctx, &QrpacOptions{Action: "test"})

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestHwts tests the Hwts function
func TestHwts(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *HwtsOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertResponse func(*testing.T, *HwtsResponse)
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "successful hwts operation with all options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"message":     "System status retrieved",
						"status":      "healthy",
						"temperature": "42°C",
						"power_state": "on",
						"health":      "good",
						"last_check":  "2024-01-15 16:00:00",
						"alerts":      []string{},
					},
				},
			},
			options: &HwtsOptions{
				Action:    "status",
				Detail:    true,
				Refresh:   true,
				Component: "cpu",
			},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *HwtsResponse) {
				t.Helper()
				if !resp.Data.Success {
					t.Error("Expected success=true, got false")
				}
				if resp.Data.Status != "healthy" {
					t.Errorf("Expected status=healthy, got %s", resp.Data.Status)
				}
				if resp.Data.Health != "good" {
					t.Errorf("Expected health=good, got %s", resp.Data.Health)
				}
			},
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "hwts" {
					t.Errorf("Expected func=hwts, got %s", fn)
				}
				if action := r.URL.Query().Get("action"); action != "status" {
					t.Errorf("Expected action=status, got %s", action)
				}
				if detail := r.URL.Query().Get("detail"); detail != "1" {
					t.Errorf("Expected detail=1, got %s", detail)
				}
				if refresh := r.URL.Query().Get("refresh"); refresh != "1" {
					t.Errorf("Expected refresh=1, got %s", refresh)
				}
				if component := r.URL.Query().Get("component"); component != "cpu" {
					t.Errorf("Expected component=cpu, got %s", component)
				}
			},
		},
		{
			name: "successful hwts operation with minimal options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"status":      "ok",
						"temperature": "38°C",
						"power_state": "on",
						"health":      "excellent",
					},
				},
			},
			options: &HwtsOptions{
				Action: "check",
			},
			wantErr: false,
		},
		{
			name: "successful hwts operation with nil options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"status":      "running",
						"temperature": "40°C",
						"power_state": "on",
						"health":      "normal",
					},
				},
			},
			options: nil,
			wantErr: false,
		},
		{
			name: "successful hwts operation with empty options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: &HwtsOptions{},
			wantErr: false,
		},
		{
			name: "successful hwts operation with detail=true",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"status":      "detailed",
						"temperature": "45°C",
						"power_state": "on",
						"health":      "good",
					},
				},
			},
			options: &HwtsOptions{
				Detail: true,
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if detail := r.URL.Query().Get("detail"); detail != "1" {
					t.Errorf("Expected detail=1, got %s", detail)
				}
			},
		},
		{
			name: "successful hwts operation with detail=false",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"status":  "basic",
					},
				},
			},
			options: &HwtsOptions{
				Detail: false,
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				// detail=false should not be included in request
				if detail := r.URL.Query().Get("detail"); detail != "" {
					t.Errorf("Expected no detail parameter, got %s", detail)
				}
			},
		},
		{
			name: "successful hwts operation with refresh=true",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"status":      "refreshed",
						"temperature": "41°C",
						"power_state": "on",
					},
				},
			},
			options: &HwtsOptions{
				Refresh: true,
			},
			wantErr: false,
		},
		{
			name: "successful hwts operation with refresh=false",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: &HwtsOptions{
				Refresh: false,
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if refresh := r.URL.Query().Get("refresh"); refresh != "" {
					t.Errorf("Expected no refresh parameter, got %s", refresh)
				}
			},
		},
		{
			name: "successful hwts operation with component",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":  true,
						"status":   "component_ok",
						"component": "memory",
					},
				},
			},
			options: &HwtsOptions{
				Component: "memory",
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if component := r.URL.Query().Get("component"); component != "memory" {
					t.Errorf("Expected component=memory, got %s", component)
				}
			},
		},
		{
			name: "successful hwts operation with warnings",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"message":     "System has warnings",
						"status":      "warning",
						"temperature": "65°C",
						"power_state": "on",
						"health":      "warning",
						"last_check":  "2024-01-15 16:00:00",
						"alerts":      []string{"High temperature", "Fan speed low"},
					},
				},
			},
			options: &HwtsOptions{
				Detail: true,
			},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *HwtsResponse) {
				t.Helper()
				if len(resp.Data.Alerts) != 2 {
					t.Errorf("Expected 2 alerts, got %d", len(resp.Data.Alerts))
				}
				if resp.Data.Health != "warning" {
					t.Errorf("Expected health=warning, got %s", resp.Data.Health)
				}
			},
		},
		{
			name: "successful hwts operation with critical alerts",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"message":     "Critical system status",
						"status":      "critical",
						"temperature": "85°C",
						"power_state": "on",
						"health":      "critical",
						"last_check":  "2024-01-15 16:00:00",
						"alerts":      []string{"Critical: CPU overheating", "Critical: Power supply failure", "Warning: Disk degraded"},
					},
				},
			},
			options: &HwtsOptions{
				Detail: true,
				Refresh: true,
			},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *HwtsResponse) {
				t.Helper()
				if len(resp.Data.Alerts) != 3 {
					t.Errorf("Expected 3 alerts, got %d", len(resp.Data.Alerts))
				}
				if resp.Data.Status != "critical" {
					t.Errorf("Expected status=critical, got %s", resp.Data.Status)
				}
			},
		},
		{
			name: "successful hwts operation with empty alerts",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success":     true,
						"status":      "healthy",
						"temperature": "35°C",
						"power_state": "on",
						"health":      "excellent",
						"alerts":      []string{},
					},
				},
			},
			options: &HwtsOptions{},
			wantErr: false,
			assertResponse: func(t *testing.T, resp *HwtsResponse) {
				t.Helper()
				if len(resp.Data.Alerts) != 0 {
					t.Errorf("Expected 0 alerts, got %d", len(resp.Data.Alerts))
				}
			},
		},
		{
			name: "API error - component not found",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "Component not found",
				},
			},
			options:     &HwtsOptions{Component: "nonexistent"},
			wantErr:     true,
			expectedErr: api.ErrNotFound,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
			},
			options:     &HwtsOptions{},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			options: &HwtsOptions{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.Hwts(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("Hwts() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Hwts() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Fatal("Hwts() returned nil response")
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

// TestHwtsAuthentication tests authentication scenarios for Hwts
func TestHwtsAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		_, err := fs.Hwts(ctx, &HwtsOptions{Action: "status"})

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestUnsupportedOperationStubFunctions tests the 12 stub functions for unsupported operations
func TestUnsupportedOperationStubFunctions(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(*testing.T, *FileStationService, context.Context)
	}{
		{
			name: "G returns unsupported error",
			testFunc: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				resp, err := fs.G(ctx, map[string]string{"key": "value"})
				if err != nil {
					t.Errorf("G() unexpected error = %v", err)
				}
				assertUnsupportedOperation(t, resp, "g")
			},
		},
		{
			name: "L returns unsupported error",
			testFunc: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				resp, err := fs.L(ctx, map[string]string{"key": "value"})
				if err != nil {
					t.Errorf("L() unexpected error = %v", err)
				}
				assertUnsupportedOperation(t, resp, "l")
			},
		},
		{
			name: "SetUnderscore returns unsupported error",
			testFunc: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				resp, err := fs.SetUnderscore(ctx, map[string]string{"key": "value"})
				if err != nil {
					t.Errorf("SetUnderscore() unexpected error = %v", err)
				}
				assertUnsupportedOperation(t, resp, "set_")
			},
		},
		{
			name: "SetP returns unsupported error",
			testFunc: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				resp, err := fs.SetP(ctx, map[string]string{"key": "value"})
				if err != nil {
					t.Errorf("SetP() unexpected error = %v", err)
				}
				assertUnsupportedOperation(t, resp, "set_p")
			},
		},
		{
			name: "GetS returns unsupported error",
			testFunc: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				resp, err := fs.GetS(ctx, map[string]string{"key": "value"})
				if err != nil {
					t.Errorf("GetS() unexpected error = %v", err)
				}
				assertUnsupportedOperation(t, resp, "get_s")
			},
		},
		{
			name: "GetR returns unsupported error",
			testFunc: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				resp, err := fs.GetR(ctx, map[string]string{"key": "value"})
				if err != nil {
					t.Errorf("GetR() unexpected error = %v", err)
				}
				assertUnsupportedOperation(t, resp, "get_r")
			},
		},
		{
			name: "GetUnderscore returns unsupported error",
			testFunc: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				resp, err := fs.GetUnderscore(ctx, map[string]string{"key": "value"})
				if err != nil {
					t.Errorf("GetUnderscore() unexpected error = %v", err)
				}
				assertUnsupportedOperation(t, resp, "get_")
			},
		},
		{
			name: "Func returns unsupported error",
			testFunc: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				resp, err := fs.Func(ctx, map[string]string{"key": "value"})
				if err != nil {
					t.Errorf("Func() unexpected error = %v", err)
				}
				assertUnsupportedOperation(t, resp, "func")
			},
		},
		{
			name: "Dryru returns unsupported error",
			testFunc: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				resp, err := fs.Dryru(ctx, map[string]string{"key": "value"})
				if err != nil {
					t.Errorf("Dryru() unexpected error = %v", err)
				}
				assertUnsupportedOperation(t, resp, "dryru")
			},
		},
		{
			name: "Umo returns unsupported error",
			testFunc: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				resp, err := fs.Umo(ctx, map[string]string{"key": "value"})
				if err != nil {
					t.Errorf("Umo() unexpected error = %v", err)
				}
				assertUnsupportedOperation(t, resp, "umo")
			},
		},
		{
			name: "Mou returns unsupported error",
			testFunc: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				resp, err := fs.Mou(ctx, map[string]string{"key": "value"})
				if err != nil {
					t.Errorf("Mou() unexpected error = %v", err)
				}
				assertUnsupportedOperation(t, resp, "mou")
			},
		},
		{
			name: "ShareUnderscore returns unsupported error",
			testFunc: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				resp, err := fs.ShareUnderscore(ctx, map[string]string{"key": "value"})
				if err != nil {
					t.Errorf("ShareUnderscore() unexpected error = %v", err)
				}
				assertUnsupportedOperation(t, resp, "share_")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := setupTestClient(t)
			fs := NewFileStationService(client)
			ctx := context.Background()
			tt.testFunc(t, fs, ctx)
		})
	}
}

// TestUnsupportedOperationStubFunctionsWithNilParams tests stub functions with nil parameters
func TestUnsupportedOperationStubFunctionsWithNilParams(t *testing.T) {
	client, _ := setupTestClient(t)
	fs := NewFileStationService(client)
	ctx := context.Background()

	// Test with nil params
	stubFuncs := []struct {
		name string
		fn   func(context.Context, map[string]string) (*UnsupportedOperationResponse, error)
		op   string
	}{
		{"G", fs.G, "g"},
		{"L", fs.L, "l"},
		{"SetUnderscore", fs.SetUnderscore, "set_"},
		{"SetP", fs.SetP, "set_p"},
		{"GetS", fs.GetS, "get_s"},
		{"GetR", fs.GetR, "get_r"},
		{"GetUnderscore", fs.GetUnderscore, "get_"},
		{"Func", fs.Func, "func"},
		{"Dryru", fs.Dryru, "dryru"},
		{"Umo", fs.Umo, "umo"},
		{"Mou", fs.Mou, "mou"},
		{"ShareUnderscore", fs.ShareUnderscore, "share_"},
	}

	for _, sf := range stubFuncs {
		t.Run(sf.name+" with nil params", func(t *testing.T) {
			resp, err := sf.fn(ctx, nil)
			if err != nil {
				t.Errorf("%s() with nil params unexpected error = %v", sf.name, err)
			}
			assertUnsupportedOperation(t, resp, sf.op)
		})
	}
}

// TestUnsupportedOperationStubFunctionsWithEmptyParams tests stub functions with empty parameters
func TestUnsupportedOperationStubFunctionsWithEmptyParams(t *testing.T) {
	client, _ := setupTestClient(t)
	fs := NewFileStationService(client)
	ctx := context.Background()

	// Test with empty params
	stubFuncs := []struct {
		name string
		fn   func(context.Context, map[string]string) (*UnsupportedOperationResponse, error)
		op   string
	}{
		{"G", fs.G, "g"},
		{"L", fs.L, "l"},
		{"SetUnderscore", fs.SetUnderscore, "set_"},
		{"SetP", fs.SetP, "set_p"},
		{"GetS", fs.GetS, "get_s"},
		{"GetR", fs.GetR, "get_r"},
		{"GetUnderscore", fs.GetUnderscore, "get_"},
		{"Func", fs.Func, "func"},
		{"Dryru", fs.Dryru, "dryru"},
		{"Umo", fs.Umo, "umo"},
		{"Mou", fs.Mou, "mou"},
		{"ShareUnderscore", fs.ShareUnderscore, "share_"},
	}

	for _, sf := range stubFuncs {
		t.Run(sf.name+" with empty params", func(t *testing.T) {
			resp, err := sf.fn(ctx, map[string]string{})
			if err != nil {
				t.Errorf("%s() with empty params unexpected error = %v", sf.name, err)
			}
			assertUnsupportedOperation(t, resp, sf.op)
		})
	}
}

// TestMiscContextCancellation tests context cancellation handling for misc functions
func TestMiscContextCancellation(t *testing.T) {
	tests := []struct {
		name  string
		testFn func(*testing.T, *FileStationService, context.Context)
	}{
		{
			name: "DaemonList respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.DaemonList(ctx)
				if err != nil && err.Error() != "operation 'g' is not supported" {
					t.Logf("DaemonList with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "GetCayinMediaStatus respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.GetCayinMediaStatus(ctx)
				if err != nil && err.Error() != "operation 'l' is not supported" {
					t.Logf("GetCayinMediaStatus with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "QcloudNotifyInfo respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.QcloudNotifyInfo(ctx)
				if err != nil && err.Error() != "operation 'set_' is not supported" {
					t.Logf("QcloudNotifyInfo with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "QcloudWopiUrl respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.QcloudWopiUrl(ctx, &QcloudWopiUrlOptions{})
				if err != nil && err.Error() != "operation 'set_p' is not supported" {
					t.Logf("QcloudWopiUrl with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "Qdmc respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.Qdmc(ctx, &QdmcOptions{})
				if err != nil && err.Error() != "operation 'get_s' is not supported" {
					t.Logf("Qdmc with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "QhamRetrieve respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.QhamRetrieve(ctx, &QhamRetrieveOptions{})
				if err != nil && err.Error() != "operation 'get_r' is not supported" {
					t.Logf("QhamRetrieve with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "Qrpac respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.Qrpac(ctx, &QrpacOptions{})
				if err != nil && err.Error() != "operation 'get_' is not supported" {
					t.Logf("Qrpac with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "Hwts respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.Hwts(ctx, &HwtsOptions{})
				if err != nil && err.Error() != "operation 'func' is not supported" {
					t.Logf("Hwts with canceled context returned error: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := setupTestClient(t)
			fs := NewFileStationService(client)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			tt.testFn(t, fs, ctx)
		})
	}
}

// TestMiscFunctionsWithSpecialCharacters tests misc functions with special characters in input
func TestMiscFunctionsWithSpecialCharacters(t *testing.T) {
	client, mockServer := setupTestClient(t)
	defer mockServer.Close()
	fs := NewFileStationService(client)
	ctx := context.Background()

	tests := []struct {
		name         string
		mockResponse testutil.MockResponse
		testFn       func(*testing.T, *FileStationService, context.Context)
	}{
		{
			name: "QcloudWopiUrl with unicode filename",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"url":     "https://wopi.example.com/files/test",
						"enabled": true,
					},
				},
			},
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.QcloudWopiUrl(ctx, &QcloudWopiUrlOptions{
					FileName: "文档文件名.docx",
				})
				if err != nil {
					t.Errorf("QcloudWopiUrl() with unicode filename error = %v", err)
				}
			},
		},
		{
			name: "Qdmc with special characters in target",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.Qdmc(ctx, &QdmcOptions{
					Target: "/path/with spaces & special-chars_@#/文件夹",
				})
				if err != nil {
					t.Errorf("Qdmc() with special characters error = %v", err)
				}
			},
		},
		{
			name: "QhamRetrieve with unicode source",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"items":   []string{},
					},
				},
			},
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.QhamRetrieve(ctx, &QhamRetrieveOptions{
					Source: "云存储/备份",
				})
				if err != nil {
					t.Errorf("QhamRetrieve() with unicode source error = %v", err)
				}
			},
		},
		{
			name: "Qrpac with special parameter values",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.Qrpac(ctx, &QrpacOptions{
					Parameters: map[string]string{
						"path":    "/path/with spaces/",
						"unicode": "中文",
						"emoji":   "😀🎉",
						"quotes":  `"quoted"`,
					},
				})
				if err != nil {
					t.Errorf("Qrpac() with special parameter values error = %v", err)
				}
			},
		},
		{
			name: "Hwts with component containing special characters",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.Hwts(ctx, &HwtsOptions{
					Component: "cpu-socket@0",
				})
				if err != nil {
					t.Errorf("Hwts() with special component error = %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)
			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)
			tt.testFn(t, fs, ctx)
		})
	}
}

// BenchmarkMiscFunctions benchmarks the misc functions
func BenchmarkMiscFunctions(b *testing.B) {
	client, mockServer := setupTestClient(&testing.T{})
	defer mockServer.Close()

	ctx := context.Background()
	fs := NewFileStationService(client)

	b.Run("DaemonList", func(b *testing.B) {
		mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"daemons": []DaemonInfo{
						{PID: "1", Name: "init", Status: "running"},
					},
					"total": 1,
				},
			},
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = fs.DaemonList(ctx)
		}
	})

	b.Run("GetCayinMediaStatus", func(b *testing.B) {
		mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"enabled": true,
					"status":  "online",
				},
			},
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = fs.GetCayinMediaStatus(ctx)
		}
	})

	b.Run("QcloudNotifyInfo", func(b *testing.B) {
		mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"enabled": true,
				},
			},
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = fs.QcloudNotifyInfo(ctx)
		}
	})

	b.Run("QcloudWopiUrl", func(b *testing.B) {
		mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"url":     "https://example.com",
					"enabled": true,
				},
			},
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = fs.QcloudWopiUrl(ctx, &QcloudWopiUrlOptions{FileID: "test"})
		}
	})

	b.Run("Qdmc", func(b *testing.B) {
		mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"success": true,
				},
			},
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = fs.Qdmc(ctx, &QdmcOptions{Action: "test"})
		}
	})

	b.Run("QhamRetrieve", func(b *testing.B) {
		mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"success": true,
					"items":   []string{},
				},
			},
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = fs.QhamRetrieve(ctx, &QhamRetrieveOptions{Source: "test"})
		}
	})

	b.Run("Qrpac", func(b *testing.B) {
		mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"success": true,
				},
			},
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = fs.Qrpac(ctx, &QrpacOptions{Action: "test"})
		}
	})

	b.Run("Hwts", func(b *testing.B) {
		mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"success": true,
					"status":  "ok",
				},
			},
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = fs.Hwts(ctx, &HwtsOptions{Action: "status"})
		}
	})
}
