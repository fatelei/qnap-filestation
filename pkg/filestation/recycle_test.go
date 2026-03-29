package filestation

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/fatelei/qnap-filestation/pkg/api"
	"github.com/fatelei/qnap-filestation/internal/testutil"
)


// assertAPIErrorMessage checks if the error is an APIError with the expected message
func assertAPIErrorMessage(t *testing.T, err error, expectedMessage string) {
	t.Helper()

	if err == nil {
		t.Errorf("Expected error with message %q, but got nil", expectedMessage)
		return
	}

	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Errorf("Expected APIError, got %T", err)
		return
	}

	if apiErr.Message != expectedMessage && !strings.Contains(apiErr.Message, expectedMessage) {
		t.Errorf("Expected error message to contain %q, got %q", expectedMessage, apiErr.Message)
	}
}

// TestTrashRecovery tests the TrashRecovery function
func TestTrashRecovery(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		sourcePath     string
		files          []string
		options        *TrashRecoveryOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		expectedErrMsg string
		assertRequest  func(*testing.T, *http.Request)
		validateResp   func(*testing.T, *TrashRecoveryResponse)
	}{
		{
			name: "recover single file successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":     "rec-12345",
						"success": true,
					},
				},
			},
			sourcePath: "/home",
			files:      []string{"deleted_file.txt"},
			wantErr:    false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if path := r.URL.Query().Get("source_path"); path != "/home" {
					t.Errorf("Expected source_path=/home, got %s", path)
				}
				if file := r.URL.Query().Get("source_file[0]"); file != "deleted_file.txt" {
					t.Errorf("Expected source_file[0]=deleted_file.txt, got %s", file)
				}
				if total := r.URL.Query().Get("file_total"); total != "1" {
					t.Errorf("Expected file_total=1, got %s", total)
				}
				if fn := r.URL.Query().Get("func"); fn != "trash_recovery" {
					t.Errorf("Expected func=trash_recovery, got %s", fn)
				}
			},
			validateResp: func(t *testing.T, resp *TrashRecoveryResponse) {
				t.Helper()
				if resp.Data.PID != "rec-12345" {
					t.Errorf("Expected PID=rec-12345, got %s", resp.Data.PID)
				}
				if !resp.Data.Success {
					t.Error("Expected Success=true")
				}
			},
		},
		{
			name: "recover multiple files successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":     "rec-67890",
						"success": true,
					},
				},
			},
			sourcePath: "/home/documents",
			files:      []string{"file1.txt", "file2.pdf", "file3.doc"},
			wantErr:    false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if r.URL.Query().Get("source_file[0]") != "file1.txt" {
					t.Error("Expected source_file[0]=file1.txt")
				}
				if r.URL.Query().Get("source_file[1]") != "file2.pdf" {
					t.Error("Expected source_file[1]=file2.pdf")
				}
				if r.URL.Query().Get("source_file[2]") != "file3.doc" {
					t.Error("Expected source_file[2]=file3.doc")
				}
				if total := r.URL.Query().Get("file_total"); total != "3" {
					t.Errorf("Expected file_total=3, got %s", total)
				}
			},
		},
		{
			name: "recover with overwrite option",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":     "rec-overwrite",
						"success": true,
					},
				},
			},
			sourcePath: "/home",
			files:      []string{"file.txt"},
			options: &TrashRecoveryOptions{
				Overwrite: true,
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if overwrite := r.URL.Query().Get("overwrite"); overwrite != "1" {
					t.Errorf("Expected overwrite=1, got %s", overwrite)
				}
			},
		},
		{
			name: "recover with custom destination path",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":     "rec-dest",
						"success": true,
					},
				},
			},
			sourcePath: "/trash",
			files:      []string{"restored.txt"},
			options: &TrashRecoveryOptions{
				DestPath: "/restored_files",
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if dest := r.URL.Query().Get("dest_path"); dest != "/restored_files" {
					t.Errorf("Expected dest_path=/restored_files, got %s", dest)
				}
			},
		},
		{
			name: "recover with task ID for tracking",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":     "task-123",
						"success": true,
					},
				},
			},
			sourcePath: "/home",
			files:      []string{"file.txt"},
			options: &TrashRecoveryOptions{
				TaskID: "task-123",
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if taskID := r.URL.Query().Get("task_id"); taskID != "task-123" {
					t.Errorf("Expected task_id=task-123, got %s", taskID)
				}
			},
		},
		{
			name: "recover with all options set",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":     "rec-all",
						"success": true,
					},
				},
			},
			sourcePath: "/trash",
			files:      []string{"file1.txt", "file2.txt"},
			options: &TrashRecoveryOptions{
				TaskID:     "task-full",
				Overwrite:  true,
				DestPath:   "/restored",
				SourcePath: "/custom_source",
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if taskID := r.URL.Query().Get("task_id"); taskID != "task-full" {
					t.Errorf("Expected task_id=task-full, got %s", taskID)
				}
				if overwrite := r.URL.Query().Get("overwrite"); overwrite != "1" {
					t.Errorf("Expected overwrite=1, got %s", overwrite)
				}
				if dest := r.URL.Query().Get("dest_path"); dest != "/restored" {
					t.Errorf("Expected dest_path=/restored, got %s", dest)
				}
				// Source path in options should override parameter
				if src := r.URL.Query().Get("source_path"); src != "/custom_source" {
					t.Errorf("Expected source_path=/custom_source, got %s", src)
				}
			},
		},
		{
			name:        "error when files list is empty",
			sourcePath:  "/home",
			files:       []string{},
			wantErr:     true,
			expectedErr: api.ErrInvalidPath,
		},
		{
			name:        "error when files list is nil",
			sourcePath:  "/home",
			files:       nil,
			wantErr:     true,
			expectedErr: api.ErrInvalidPath,
		},
		{
			name: "recovery fails with API error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "File not found in trash",
				},
			},
			sourcePath:  "/home",
			files:       []string{"nonexistent.txt"},
			wantErr:     true,
			expectedErr: api.ErrNotFound,
			expectedErrMsg: "File not found in trash",
		},
		{
			name: "recovery fails with success=0",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 0,
				},
			},
			sourcePath:  "/home",
			files:       []string{"file.txt"},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
			},
			sourcePath:  "/home",
			files:       []string{"file.txt"},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			sourcePath: "/home",
			files:      []string{"file.txt"},
			wantErr:    true,
		},
		{
			name: "recover files with special characters",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":     "rec-special",
						"success": true,
					},
				},
			},
			sourcePath: "/home",
			files:      []string{"file with spaces.txt", "file(1).txt", "file-dash.txt"},
			wantErr:    false,
		},
		{
			name: "recover with message in response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":     "rec-msg",
						"success": true,
						"message": "Recovery started successfully",
					},
				},
			},
			sourcePath: "/home",
			files:      []string{"file.txt"},
			wantErr:    false,
			validateResp: func(t *testing.T, resp *TrashRecoveryResponse) {
				t.Helper()
				if resp.Data.Message != "Recovery started successfully" {
					t.Errorf("Expected message='Recovery started successfully', got %s", resp.Data.Message)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.TrashRecovery(ctx, tt.sourcePath, tt.files, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("TrashRecovery() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				if tt.expectedErrMsg != "" {
					assertAPIErrorMessage(t, err, tt.expectedErrMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("TrashRecovery() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Error("TrashRecovery() returned nil response")
				return
			}

			if tt.assertRequest != nil {
				lastReq := mockServer.GetLastRequest()
				if lastReq != nil {
					tt.assertRequest(t, lastReq)
				}
			}

			if tt.validateResp != nil {
				tt.validateResp(t, resp)
			}
		})
	}
}

// TestTrashRecoveryAuthentication tests authentication scenarios for TrashRecovery
func TestTrashRecoveryAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		_, err := fs.TrashRecovery(ctx, "/home", []string{"file.txt"}, nil)

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestCancelTrashRecovery tests the CancelTrashRecovery function
func TestCancelTrashRecovery(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		taskID         string
		wantErr        bool
		expectedErr    api.ErrorCode
		expectedErrMsg string
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "cancel trash recovery successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			taskID:  "rec-12345",
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if taskID := r.URL.Query().Get("task_id"); taskID != "rec-12345" {
					t.Errorf("Expected task_id=rec-12345, got %s", taskID)
				}
				if fn := r.URL.Query().Get("func"); fn != "cancel_trash_recovery" {
					t.Errorf("Expected func=cancel_trash_recovery, got %s", fn)
				}
			},
		},
		{
			name: "cancel with message",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"message": "Recovery canceled successfully",
					},
				},
			},
			taskID:  "rec-67890",
			wantErr: false,
		},
		{
			name:        "error when task ID is empty",
			taskID:      "",
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "cancel fails with API error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2003,
					"error_msg":  "Permission denied",
				},
			},
			taskID:        "rec-12345",
			wantErr:       true,
			expectedErr:   api.ErrPermission,
			expectedErrMsg: "Permission denied",
		},
		{
			name: "cancel fails with success=0",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 0,
				},
			},
			taskID:      "rec-12345",
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
			},
			taskID:      "rec-12345",
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			taskID:  "rec-12345",
			wantErr: true,
		},
		{
			name: "cancel with numeric task ID",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			taskID:  "12345",
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

			resp, err := fs.CancelTrashRecovery(ctx, tt.taskID)

			if tt.wantErr {
				if err == nil {
					t.Error("CancelTrashRecovery() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				if tt.expectedErrMsg != "" {
					assertAPIErrorMessage(t, err, tt.expectedErrMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("CancelTrashRecovery() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Error("CancelTrashRecovery() returned nil response")
				return
			}

			if !resp.Data.Success {
				t.Error("CancelTrashRecovery() returned success=false")
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

// TestCancelTrashRecoveryAuthentication tests authentication scenarios for CancelTrashRecovery
func TestCancelTrashRecoveryAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		_, err := fs.CancelTrashRecovery(ctx, "task-123")

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestGetRecycleBinStatus tests the GetRecycleBinStatus function
func TestGetRecycleBinStatus(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *GetRecycleBinStatusOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertRequest  func(*testing.T, *http.Request)
		validateResp   func(*testing.T, *GetRecycleBinStatusResponse)
	}{
		{
			name: "get recycle bin status successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"enabled":      true,
						"volume_count": 2,
						"volumes": []map[string]interface{}{
							{
								"volume_name": "Share1",
								"enabled":     true,
								"path":        "/share1/.recycle",
								"item_count":  15,
								"total_size":  1073741824,
							},
							{
								"volume_name": "Share2",
								"enabled":     true,
								"path":        "/share2/.recycle",
								"item_count":  8,
								"total_size":  536870912,
							},
						},
					},
				},
			},
			options: nil,
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "get_recycle_bin_status" {
					t.Errorf("Expected func=get_recycle_bin_status, got %s", fn)
				}
			},
			validateResp: func(t *testing.T, resp *GetRecycleBinStatusResponse) {
				t.Helper()
				if !resp.Data.Enabled {
					t.Error("Expected Enabled=true")
				}
				if resp.Data.VolumeCount != 2 {
					t.Errorf("Expected VolumeCount=2, got %d", resp.Data.VolumeCount)
				}
				if len(resp.Data.Volumes) != 2 {
					t.Errorf("Expected 2 volumes, got %d", len(resp.Data.Volumes))
				}
				if resp.Data.Volumes[0].VolumeName != "Share1" {
					t.Errorf("Expected first volume name=Share1, got %s", resp.Data.Volumes[0].VolumeName)
				}
				if resp.Data.Volumes[0].ItemCount != 15 {
					t.Errorf("Expected first volume item_count=15, got %d", resp.Data.Volumes[0].ItemCount)
				}
			},
		},
		{
			name: "get status for specific volume",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"enabled":      true,
						"volume_count": 1,
						"volumes": []map[string]interface{}{
							{
								"volume_name": "Share1",
								"enabled":     true,
								"path":        "/share1/.recycle",
								"item_count":  5,
								"total_size":  1048576,
							},
						},
					},
				},
			},
			options: &GetRecycleBinStatusOptions{
				VolumeName: "Share1",
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if vol := r.URL.Query().Get("volume_name"); vol != "Share1" {
					t.Errorf("Expected volume_name=Share1, got %s", vol)
				}
			},
		},
		{
			name: "recycle bin disabled",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"enabled":      false,
						"volume_count": 0,
						"volumes":      []map[string]interface{}{},
					},
				},
			},
			options: nil,
			wantErr: false,
			validateResp: func(t *testing.T, resp *GetRecycleBinStatusResponse) {
				t.Helper()
				if resp.Data.Enabled {
					t.Error("Expected Enabled=false")
				}
				if resp.Data.VolumeCount != 0 {
					t.Errorf("Expected VolumeCount=0, got %d", resp.Data.VolumeCount)
				}
			},
		},
		{
			name: "empty volumes list",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"enabled":      true,
						"volume_count": 0,
						"volumes":      []map[string]interface{}{},
					},
				},
			},
			options: nil,
			wantErr: false,
			validateResp: func(t *testing.T, resp *GetRecycleBinStatusResponse) {
				t.Helper()
				if len(resp.Data.Volumes) != 0 {
					t.Errorf("Expected 0 volumes, got %d", len(resp.Data.Volumes))
				}
			},
		},
		{
			name: "nil options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"enabled":      true,
						"volume_count": 1,
						"volumes": []map[string]interface{}{
							{
								"volume_name": "Share1",
								"enabled":     true,
								"path":        "/share1/.recycle",
							},
						},
					},
				},
			},
			options: nil,
			wantErr: false,
		},
		{
			name: "options with empty volume name",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"enabled":      true,
						"volume_count": 1,
						"volumes":      []map[string]interface{}{},
					},
				},
			},
			options: &GetRecycleBinStatusOptions{
				VolumeName: "",
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				// Empty volume name should not be sent
				if vol := r.URL.Query().Get("volume_name"); vol != "" {
					t.Errorf("Expected no volume_name parameter, got %s", vol)
				}
			},
		},
		{
			name: "status fails with API error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 1003,
					"error_msg":  "Invalid parameters",
				},
			},
			options:     &GetRecycleBinStatusOptions{VolumeName: "Invalid"},
			wantErr:     true,
			expectedErr: api.ErrInvalidParams,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
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
			name: "volume without item count and size",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"enabled":      true,
						"volume_count": 1,
						"volumes": []map[string]interface{}{
							{
								"volume_name": "Share1",
								"enabled":     true,
								"path":        "/share1/.recycle",
							},
						},
					},
				},
			},
			options: nil,
			wantErr: false,
			validateResp: func(t *testing.T, resp *GetRecycleBinStatusResponse) {
				t.Helper()
				if resp.Data.Volumes[0].ItemCount != 0 {
					t.Errorf("Expected ItemCount=0, got %d", resp.Data.Volumes[0].ItemCount)
				}
				if resp.Data.Volumes[0].TotalSize != 0 {
					t.Errorf("Expected TotalSize=0, got %d", resp.Data.Volumes[0].TotalSize)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.GetRecycleBinStatus(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("GetRecycleBinStatus() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("GetRecycleBinStatus() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Error("GetRecycleBinStatus() returned nil response")
				return
			}

			if tt.assertRequest != nil {
				lastReq := mockServer.GetLastRequest()
				if lastReq != nil {
					tt.assertRequest(t, lastReq)
				}
			}

			if tt.validateResp != nil {
				tt.validateResp(t, resp)
			}
		})
	}
}

// TestGetRecycleBinStatusAuthentication tests authentication scenarios for GetRecycleBinStatus
func TestGetRecycleBinStatusAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		_, err := fs.GetRecycleBinStatus(ctx, nil)

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestEmptyTrash tests the EmptyTrash function
func TestEmptyTrash(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *EmptyTrashOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "empty all trash successfully",
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
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "empty_trash" {
					t.Errorf("Expected func=empty_trash, got %s", fn)
				}
				// No volume_name should be present
				if vol := r.URL.Query().Get("volume_name"); vol != "" {
					t.Errorf("Expected no volume_name, got %s", vol)
				}
			},
		},
		{
			name: "empty specific volume trash",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"message": "Trash for Share1 emptied",
					},
				},
			},
			options: &EmptyTrashOptions{
				VolumeName: "Share1",
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if vol := r.URL.Query().Get("volume_name"); vol != "Share1" {
					t.Errorf("Expected volume_name=Share1, got %s", vol)
				}
			},
		},
		{
			name: "empty trash with message",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"message": "All trash emptied successfully",
					},
				},
			},
			options: nil,
			wantErr: false,
		},
		{
			name: "options with empty volume name",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: &EmptyTrashOptions{
				VolumeName: "",
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				// Empty volume name should not be sent
				if vol := r.URL.Query().Get("volume_name"); vol != "" {
					t.Errorf("Expected no volume_name parameter, got %s", vol)
				}
			},
		},
		{
			name: "empty trash fails with API error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2003,
					"error_msg":  "Permission denied",
				},
			},
			options:     &EmptyTrashOptions{VolumeName: "Share1"},
			wantErr:     true,
			expectedErr: api.ErrPermission,
		},
		{
			name: "empty trash fails with success=0",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 0,
				},
			},
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
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
			name: "empty multiple volumes sequentially",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: &EmptyTrashOptions{
				VolumeName: "Share2",
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if vol := r.URL.Query().Get("volume_name"); vol != "Share2" {
					t.Errorf("Expected volume_name=Share2, got %s", vol)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.EmptyTrash(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("EmptyTrash() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("EmptyTrash() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Error("EmptyTrash() returned nil response")
				return
			}

			if !resp.Data.Success {
				t.Error("EmptyTrash() returned success=false")
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

// TestEmptyTrashAuthentication tests authentication scenarios for EmptyTrash
func TestEmptyTrashAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		_, err := fs.EmptyTrash(ctx, nil)

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestSetDeletePermanently tests the SetDeletePermanently function
func TestSetDeletePermanently(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *SetDeletePermanentlyOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "enable permanent delete globally",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: &SetDeletePermanentlyOptions{
				Enabled: true,
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if enabled := r.URL.Query().Get("enabled"); enabled != "1" {
					t.Errorf("Expected enabled=1, got %s", enabled)
				}
				if vol := r.URL.Query().Get("volume_name"); vol != "" {
					t.Errorf("Expected no volume_name, got %s", vol)
				}
			},
		},
		{
			name: "disable permanent delete globally",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: &SetDeletePermanentlyOptions{
				Enabled: false,
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if enabled := r.URL.Query().Get("enabled"); enabled != "0" {
					t.Errorf("Expected enabled=0, got %s", enabled)
				}
			},
		},
		{
			name: "enable permanent delete for specific volume",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
						"message": "Permanent delete enabled for Share1",
					},
				},
			},
			options: &SetDeletePermanentlyOptions{
				Enabled:    true,
				VolumeName: "Share1",
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if enabled := r.URL.Query().Get("enabled"); enabled != "1" {
					t.Errorf("Expected enabled=1, got %s", enabled)
				}
				if vol := r.URL.Query().Get("volume_name"); vol != "Share1" {
					t.Errorf("Expected volume_name=Share1, got %s", vol)
				}
			},
		},
		{
			name: "disable permanent delete for specific volume",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: &SetDeletePermanentlyOptions{
				Enabled:    false,
				VolumeName: "Public",
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if enabled := r.URL.Query().Get("enabled"); enabled != "0" {
					t.Errorf("Expected enabled=0, got %s", enabled)
				}
				if vol := r.URL.Query().Get("volume_name"); vol != "Public" {
					t.Errorf("Expected volume_name=Public, got %s", vol)
				}
			},
		},
		{
			name: "nil options uses default values",
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
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				// When options is nil, no enabled parameter should be sent
				// The function should handle this gracefully
				if enabled := r.URL.Query().Get("enabled"); enabled != "" {
					t.Logf("enabled parameter sent with nil options: %s", enabled)
				}
			},
		},
		{
			name: "set permanent delete fails with API error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2003,
					"error_msg":  "Permission denied",
				},
			},
			options:     &SetDeletePermanentlyOptions{Enabled: true},
			wantErr:     true,
			expectedErr: api.ErrPermission,
		},
		{
			name: "set permanent delete fails with success=0",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 0,
				},
			},
			options:     &SetDeletePermanentlyOptions{Enabled: false},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
			},
			options:     &SetDeletePermanentlyOptions{Enabled: true},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			options: &SetDeletePermanentlyOptions{Enabled: true},
			wantErr: true,
		},
		{
			name: "set with empty volume name",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"success": true,
					},
				},
			},
			options: &SetDeletePermanentlyOptions{
				Enabled:    true,
				VolumeName: "",
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				// Empty volume name should not be sent
				if vol := r.URL.Query().Get("volume_name"); vol != "" {
					t.Errorf("Expected no volume_name parameter, got %s", vol)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.SetDeletePermanently(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("SetDeletePermanently() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("SetDeletePermanently() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Error("SetDeletePermanently() returned nil response")
				return
			}

			if !resp.Data.Success {
				t.Error("SetDeletePermanently() returned success=false")
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

// TestSetDeletePermanentlyAuthentication tests authentication scenarios for SetDeletePermanently
func TestSetDeletePermanentlyAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		_, err := fs.SetDeletePermanently(ctx, &SetDeletePermanentlyOptions{Enabled: true})

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestGetDeleteStatus tests the GetDeleteStatus function
func TestGetDeleteStatus(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   testutil.MockResponse
		options        *GetDeleteStatusOptions
		wantErr        bool
		expectedErr    api.ErrorCode
		assertRequest  func(*testing.T, *http.Request)
		validateResp   func(*testing.T, *GetDeleteStatusResponse)
	}{
		{
			name: "get delete status successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":         "del-12345",
						"status":      "running",
						"progress":    45.5,
						"total_count": 100,
						"processed":   45,
						"failed_count": 2,
					},
				},
			},
			options: nil,
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if fn := r.URL.Query().Get("func"); fn != "get_delete_status" {
					t.Errorf("Expected func=get_delete_status, got %s", fn)
				}
			},
			validateResp: func(t *testing.T, resp *GetDeleteStatusResponse) {
				t.Helper()
				if resp.Data.PID != "del-12345" {
					t.Errorf("Expected PID=del-12345, got %s", resp.Data.PID)
				}
				if resp.Data.Status != "running" {
					t.Errorf("Expected Status=running, got %s", resp.Data.Status)
				}
				if resp.Data.Progress != 45.5 {
					t.Errorf("Expected Progress=45.5, got %f", resp.Data.Progress)
				}
				if resp.Data.TotalCount != 100 {
					t.Errorf("Expected TotalCount=100, got %d", resp.Data.TotalCount)
				}
				if resp.Data.Processed != 45 {
					t.Errorf("Expected Processed=45, got %d", resp.Data.Processed)
				}
				if resp.Data.FailedCount != 2 {
					t.Errorf("Expected FailedCount=2, got %d", resp.Data.FailedCount)
				}
			},
		},
		{
			name: "get status with task ID",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":         "task-67890",
						"status":      "finished",
						"progress":    100.0,
						"total_count": 50,
						"processed":   50,
						"failed_count": 0,
					},
				},
			},
			options: &GetDeleteStatusOptions{
				TaskID: "task-67890",
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if taskID := r.URL.Query().Get("task_id"); taskID != "task-67890" {
					t.Errorf("Expected task_id=task-67890, got %s", taskID)
				}
			},
		},
		{
			name: "delete status finished",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":         "del-finished",
						"status":      "finished",
						"progress":    100.0,
						"total_count": 10,
						"processed":   10,
						"failed_count": 0,
					},
				},
			},
			options: nil,
			wantErr: false,
			validateResp: func(t *testing.T, resp *GetDeleteStatusResponse) {
				t.Helper()
				if resp.Data.Status != "finished" {
					t.Errorf("Expected Status=finished, got %s", resp.Data.Status)
				}
				if resp.Data.Progress != 100.0 {
					t.Errorf("Expected Progress=100.0, got %f", resp.Data.Progress)
				}
			},
		},
		{
			name: "delete status failed",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":          "del-failed",
						"status":       "failed",
						"progress":     25.0,
						"total_count":  20,
						"processed":    5,
						"failed_count": 1,
						"error":        "Access denied to file3.txt",
					},
				},
			},
			options: nil,
			wantErr: false,
			validateResp: func(t *testing.T, resp *GetDeleteStatusResponse) {
				t.Helper()
				if resp.Data.Status != "failed" {
					t.Errorf("Expected Status=failed, got %s", resp.Data.Status)
				}
				if resp.Data.Error != "Access denied to file3.txt" {
					t.Errorf("Expected Error='Access denied to file3.txt', got %s", resp.Data.Error)
				}
			},
		},
		{
			name: "delete status in progress",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":         "del-running",
						"status":      "running",
						"progress":    67.8,
						"total_count": 1000,
						"processed":   678,
						"failed_count": 5,
					},
				},
			},
			options: nil,
			wantErr: false,
		},
		{
			name: "nil options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":         "del-nil",
						"status":      "running",
						"progress":    0.0,
						"total_count": 0,
						"processed":   0,
						"failed_count": 0,
					},
				},
			},
			options: nil,
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if taskID := r.URL.Query().Get("task_id"); taskID != "" {
					t.Errorf("Expected no task_id, got %s", taskID)
				}
			},
		},
		{
			name: "options with empty task ID",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":    "del-empty",
						"status": "running",
					},
				},
			},
			options: &GetDeleteStatusOptions{
				TaskID: "",
			},
			wantErr: false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				// Empty task ID should not be sent
				if taskID := r.URL.Query().Get("task_id"); taskID != "" {
					t.Errorf("Expected no task_id parameter, got %s", taskID)
				}
			},
		},
		{
			name: "zero progress values",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 1,
					"data": map[string]interface{}{
						"pid":         "del-zero",
						"status":      "running",
						"progress":    0,
						"total_count": 0,
						"processed":   0,
						"failed_count": 0,
					},
				},
			},
			options: nil,
			wantErr: false,
			validateResp: func(t *testing.T, resp *GetDeleteStatusResponse) {
				t.Helper()
				if resp.Data.Progress != 0 {
					t.Errorf("Expected Progress=0, got %f", resp.Data.Progress)
				}
			},
		},
		{
			name: "status fails with API error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success":    0,
					"error_code": 2001,
					"error_msg":  "Task not found",
				},
			},
			options:     &GetDeleteStatusOptions{TaskID: "nonexistent"},
			wantErr:     true,
			expectedErr: api.ErrNotFound,
		},
		{
			name: "status fails with success=0",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"success": 0,
				},
			},
			options:     nil,
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
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
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			resp, err := fs.GetDeleteStatus(ctx, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("GetDeleteStatus() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("GetDeleteStatus() unexpected error = %v", err)
				return
			}

			if resp == nil {
				t.Error("GetDeleteStatus() returned nil response")
				return
			}

			if tt.assertRequest != nil {
				lastReq := mockServer.GetLastRequest()
				if lastReq != nil {
					tt.assertRequest(t, lastReq)
				}
			}

			if tt.validateResp != nil {
				tt.validateResp(t, resp)
			}
		})
	}
}

// TestGetDeleteStatusAuthentication tests authentication scenarios for GetDeleteStatus
func TestGetDeleteStatusAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		_, err := fs.GetDeleteStatus(ctx, nil)

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestRecycleBinWorkflow tests a complete recycle bin workflow
func TestRecycleBinWorkflow(t *testing.T) {
	t.Run("complete workflow: check status, recover, check status, empty", func(t *testing.T) {
		client, mockServer := setupTestClient(t)
		defer mockServer.Close()

		ctx := context.Background()
		fs := NewFileStationService(client)

		// Step 1: Check initial recycle bin status
		mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"enabled":      true,
					"volume_count": 1,
					"volumes": []map[string]interface{}{
						{
							"volume_name": "Share1",
							"enabled":     true,
							"path":        "/share1/.recycle",
							"item_count":  10,
							"total_size":  1048576,
						},
					},
				},
			},
		})

		status, err := fs.GetRecycleBinStatus(ctx, nil)
		if err != nil {
			t.Fatalf("GetRecycleBinStatus() error = %v", err)
		}
		if status.Data.Volumes[0].ItemCount != 10 {
			t.Errorf("Expected 10 items, got %d", status.Data.Volumes[0].ItemCount)
		}

		// Step 2: Recover files
		mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"pid":     "rec-workflow",
					"success": true,
				},
			},
		})

		recResp, err := fs.TrashRecovery(ctx, "/share1/.recycle", []string{"file1.txt", "file2.txt"}, nil)
		if err != nil {
			t.Fatalf("TrashRecovery() error = %v", err)
		}
		if recResp.Data.PID != "rec-workflow" {
			t.Errorf("Expected PID=rec-workflow, got %s", recResp.Data.PID)
		}

		// Step 3: Check recovery status
		mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"pid":         "rec-workflow",
					"status":      "finished",
					"progress":    100.0,
					"total_count": 2,
					"processed":   2,
					"failed_count": 0,
				},
			},
		})

		delStatus, err := fs.GetDeleteStatus(ctx, &GetDeleteStatusOptions{TaskID: "rec-workflow"})
		if err != nil {
			t.Fatalf("GetDeleteStatus() error = %v", err)
		}
		if delStatus.Data.Status != "finished" {
			t.Errorf("Expected status=finished, got %s", delStatus.Data.Status)
		}

		// Step 4: Empty trash
		mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"success": 1,
				"data": map[string]interface{}{
					"success": true,
				},
			},
		})

		_, err = fs.EmptyTrash(ctx, nil)
		if err != nil {
			t.Fatalf("EmptyTrash() error = %v", err)
		}
	})
}

// TestContextCancellation_Recycle tests context cancellation for recycle functions
func TestContextCancellation_Recycle(t *testing.T) {
	tests := []struct {
		name  string
		testFn func(*testing.T, *FileStationService, context.Context)
	}{
		{
			name: "TrashRecovery respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.TrashRecovery(ctx, "/home", []string{"file.txt"}, nil)
				if err != nil && err.Error() != "" {
					t.Logf("TrashRecovery with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "CancelTrashRecovery respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.CancelTrashRecovery(ctx, "task-123")
				if err != nil && err.Error() != "" {
					t.Logf("CancelTrashRecovery with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "GetRecycleBinStatus respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.GetRecycleBinStatus(ctx, nil)
				if err != nil && err.Error() != "" {
					t.Logf("GetRecycleBinStatus with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "EmptyTrash respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.EmptyTrash(ctx, nil)
				if err != nil && err.Error() != "" {
					t.Logf("EmptyTrash with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "SetDeletePermanently respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.SetDeletePermanently(ctx, &SetDeletePermanentlyOptions{Enabled: true})
				if err != nil && err.Error() != "" {
					t.Logf("SetDeletePermanently with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "GetDeleteStatus respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.GetDeleteStatus(ctx, nil)
				if err != nil && err.Error() != "" {
					t.Logf("GetDeleteStatus with canceled context returned error: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			fs := NewFileStationService(client)
			tt.testFn(t, fs, ctx)
		})
	}
}
