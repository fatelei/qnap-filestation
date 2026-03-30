package filestation

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/fatelei/qnap-filestation/internal/testutil"
	"github.com/fatelei/qnap-filestation/pkg/api"
)

// TestListFiles tests the ListFiles function
func TestListFiles(t *testing.T) {
	tests := []struct {
		name          string
		mockResponse  testutil.MockResponse
		path          string
		options       *ListOptions
		wantErr       bool
		expectedErr   api.ErrorCode
		wantFiles     int
		wantFirstFile string
		assertRequest func(*testing.T, *http.Request)
	}{
		{
			name: "successful file listing",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"total": 2,
					"datas": []File{
						{
							FileName: "file1.txt",
							Path:     "/home/file1.txt",
							FileSize: "1024",
							IsFolder: 0,
						},
						{
							FileName: "file2.txt",
							Path:     "/home/file2.txt",
							FileSize: "2048",
							IsFolder: 0,
						},
					},
				},
			},
			path:          "/home",
			options:       nil,
			wantErr:       false,
			wantFiles:     2,
			wantFirstFile: "file1.txt",
		},
		{
			name: "empty directory",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"total": 0,
					"datas": []File{},
				},
			},
			path:      "/empty",
			options:   nil,
			wantErr:   false,
			wantFiles: 0,
		},
		{
			name: "list with pagination offset",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"total": 100,
					"datas": []File{
						{
							FileName: "file50.txt",
							Path:     "/home/file50.txt",
							FileSize: "5120",
							IsFolder: 0,
						},
					},
				},
			},
			path: "/home",
			options: &ListOptions{
				Offset: 50,
				Limit:  10,
			},
			wantErr:       false,
			wantFiles:     1,
			wantFirstFile: "file50.txt",
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if start := r.URL.Query().Get("start"); start != "50" {
					t.Errorf("Expected start=50, got %s", start)
				}
				if limit := r.URL.Query().Get("limit"); limit != "10" {
					t.Errorf("Expected limit=10, got %s", limit)
				}
			},
		},
		{
			name: "list with sorting by name ascending",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"total": 3,
					"datas": []File{
						{FileName: "a.txt", IsFolder: 0},
						{FileName: "b.txt", IsFolder: 0},
						{FileName: "c.txt", IsFolder: 0},
					},
				},
			},
			path: "/home",
			options: &ListOptions{
				SortBy:    "filename",
				SortOrder: "ASC",
			},
			wantErr:   false,
			wantFiles: 3,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if sort := r.URL.Query().Get("sort"); sort != "filename" {
					t.Errorf("Expected sort=filename, got %s", sort)
				}
				if dir := r.URL.Query().Get("dir"); dir != "ASC" {
					t.Errorf("Expected dir=ASC, got %s", dir)
				}
			},
		},
		{
			name: "list with sorting by size descending",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"total": 3,
					"datas": []File{
						{FileName: "large.txt", FileSize: "9999"},
						{FileName: "medium.txt", FileSize: "5000"},
						{FileName: "small.txt", FileSize: "1000"},
					},
				},
			},
			path: "/home",
			options: &ListOptions{
				SortBy:    "size",
				SortOrder: "DESC",
			},
			wantErr:   false,
			wantFiles: 3,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if sort := r.URL.Query().Get("sort"); sort != "size" {
					t.Errorf("Expected sort=size, got %s", sort)
				}
				if dir := r.URL.Query().Get("dir"); dir != "DESC" {
					t.Errorf("Expected dir=DESC, got %s", dir)
				}
			},
		},
		{
			name: "list includes folders",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"total": 3,
					"datas": []File{
						{FileName: "folder1", IsFolder: 1},
						{FileName: "file1.txt", IsFolder: 0},
						{FileName: "folder2", IsFolder: 1},
					},
				},
			},
			path:      "/home",
			options:   nil,
			wantErr:   false,
			wantFiles: 3,
		},
		{
			name: "list with all options",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"total": 5,
					"datas": []File{
						{FileName: "result.txt", IsFolder: 0},
					},
				},
			},
			path: "/home",
			options: &ListOptions{
				Offset:    10,
				Limit:     5,
				SortBy:    "mtime",
				SortOrder: "DESC",
			},
			wantErr:   false,
			wantFiles: 1,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if start := r.URL.Query().Get("start"); start != "10" {
					t.Errorf("Expected start=10, got %s", start)
				}
				if limit := r.URL.Query().Get("limit"); limit != "5" {
					t.Errorf("Expected limit=5, got %s", limit)
				}
				if sort := r.URL.Query().Get("sort"); sort != "mtime" {
					t.Errorf("Expected sort=mtime, got %s", sort)
				}
				if dir := r.URL.Query().Get("dir"); dir != "DESC" {
					t.Errorf("Expected dir=DESC, got %s", dir)
				}
			},
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
			},
			path:        "/home",
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
			path:    "/home",
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

			files, err := fs.ListFiles(ctx, tt.path, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Error("ListFiles() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ListFiles() unexpected error = %v", err)
				return
			}

			if len(files) != tt.wantFiles {
				t.Errorf("ListFiles() returned %d files, want %d", len(files), tt.wantFiles)
			}

			if tt.wantFirstFile != "" && len(files) > 0 {
				if files[0].FileName != tt.wantFirstFile {
					t.Errorf("ListFiles() first file = %s, want %s", files[0].FileName, tt.wantFirstFile)
				}
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

// TestListFilesAuthentication tests authentication scenarios for ListFiles
func TestListFilesAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		_, err := fs.ListFiles(ctx, "/home", nil)

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestDeleteFiles tests the DeleteFiles function (func=delete via utilRequest.cgi)
func TestDeleteFiles(t *testing.T) {
	tests := []struct {
		name          string
		mockResponse  testutil.MockResponse
		sourcePath    string
		sourceFiles   []string
		wantErr       bool
		expectedErr   api.ErrorCode
		assertRequest func(*testing.T, *http.Request)
	}{
		{
			name: "delete single file successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			sourcePath:  "/home",
			sourceFiles: []string{"file1.txt"},
			wantErr:     false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if path := r.URL.Query().Get("path"); path != "/home" {
					t.Errorf("Expected path=/home, got %s", path)
				}
				if total := r.URL.Query().Get("file_total"); total != "1" {
					t.Errorf("Expected file_total=1, got %s", total)
				}
				files := r.URL.Query()["file_name"]
				if len(files) != 1 || files[0] != "file1.txt" {
					t.Errorf("Expected file_name=[file1.txt], got %v", files)
				}
			},
		},
		{
			name: "delete multiple files successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			sourcePath:  "/home",
			sourceFiles: []string{"file1.txt", "file2.txt", "file3.txt"},
			wantErr:     false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if total := r.URL.Query().Get("file_total"); total != "3" {
					t.Errorf("Expected file_total=3, got %s", total)
				}
				files := r.URL.Query()["file_name"]
				if len(files) != 3 {
					t.Errorf("Expected 3 file_name params, got %d", len(files))
				}
				expected := []string{"file1.txt", "file2.txt", "file3.txt"}
				for i, f := range expected {
					if i < len(files) && files[i] != f {
						t.Errorf("Expected file_name[%d]=%s, got %s", i, f, files[i])
					}
				}
			},
		},
		{
			name:        "error when source files is empty",
			sourcePath:  "/home",
			sourceFiles: []string{},
			wantErr:     true,
			expectedErr: api.ErrInvalidPath,
		},
		{
			name:        "error when source files is nil",
			sourcePath:  "/home",
			sourceFiles: nil,
			wantErr:     true,
			expectedErr: api.ErrInvalidPath,
		},
		{
			name: "delete fails with status 0",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  0,
					"success": "false",
				},
			},
			sourcePath:  "/home",
			sourceFiles: []string{"file1.txt"},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "delete fails with success=false",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "false",
				},
			},
			sourcePath:  "/home",
			sourceFiles: []string{"file1.txt"},
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
			sourceFiles: []string{"file1.txt"},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			sourcePath:  "/home",
			sourceFiles: []string{"file1.txt"},
			wantErr:     true,
		},
		{
			name: "delete with special characters in filename",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			sourcePath:  "/home",
			sourceFiles: []string{"file with spaces.txt", "file-with-dashes.txt"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			err := fs.DeleteFiles(ctx, tt.sourcePath, tt.sourceFiles)

			if tt.wantErr {
				if err == nil {
					t.Error("DeleteFiles() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("DeleteFiles() unexpected error = %v", err)
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

// TestDeleteFilesAuthentication tests authentication scenarios for DeleteFiles
func TestDeleteFilesAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		err := fs.DeleteFiles(ctx, "/home", []string{"file.txt"})

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestRenameFileUtil tests the RenameFileUtil function (func=rename via utilRequest.cgi)
func TestRenameFileUtil(t *testing.T) {
	tests := []struct {
		name          string
		mockResponse  testutil.MockResponse
		path          string
		sourceName    string
		destName      string
		wantErr       bool
		expectedErr   api.ErrorCode
		assertRequest func(*testing.T, *http.Request)
	}{
		{
			name: "rename file successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			path:       "/home",
			sourceName: "oldname.txt",
			destName:   "newname.txt",
			wantErr:    false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if path := r.URL.Query().Get("path"); path != "/home" {
					t.Errorf("Expected path=/home, got %s", path)
				}
				if src := r.URL.Query().Get("source_name"); src != "oldname.txt" {
					t.Errorf("Expected source_name=oldname.txt, got %s", src)
				}
				if dst := r.URL.Query().Get("dest_name"); dst != "newname.txt" {
					t.Errorf("Expected dest_name=newname.txt, got %s", dst)
				}
			},
		},
		{
			name: "rename folder successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			path:       "/home",
			sourceName: "oldfolder",
			destName:   "newfolder",
			wantErr:    false,
		},
		{
			name:        "error when source name is empty",
			path:        "/home",
			sourceName:  "",
			destName:    "newname.txt",
			wantErr:     true,
			expectedErr: api.ErrInvalidPath,
		},
		{
			name:        "error when destination name is empty",
			path:        "/home",
			sourceName:  "oldname.txt",
			destName:    "",
			wantErr:     true,
			expectedErr: api.ErrInvalidPath,
		},
		{
			name:        "error when both names are empty",
			path:        "/home",
			sourceName:  "",
			destName:    "",
			wantErr:     true,
			expectedErr: api.ErrInvalidPath,
		},
		{
			name: "rename fails with status 0",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  0,
					"success": "true",
				},
			},
			path:        "/home",
			sourceName:  "oldname.txt",
			destName:    "newname.txt",
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "rename fails with success=false",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "false",
				},
			},
			path:        "/home",
			sourceName:  "oldname.txt",
			destName:    "newname.txt",
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "invalid JSON response",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body:       "invalid json",
			},
			path:        "/home",
			sourceName:  "oldname.txt",
			destName:    "newname.txt",
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			path:       "/home",
			sourceName: "oldname.txt",
			destName:   "newname.txt",
			wantErr:    true,
		},
		{
			name: "rename with special characters in names",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			path:       "/home",
			sourceName: "file with spaces.txt",
			destName:   "file-with-dashes.txt",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			err := fs.RenameFileUtil(ctx, tt.path, tt.sourceName, tt.destName)

			if tt.wantErr {
				if err == nil {
					t.Error("RenameFileUtil() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("RenameFileUtil() unexpected error = %v", err)
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

// TestRenameFileUtilAuthentication tests authentication scenarios for RenameFileUtil
func TestRenameFileUtilAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		err := fs.RenameFileUtil(ctx, "/home", "old.txt", "new.txt")

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestCopyFilesUtil tests the CopyFilesUtil function (func=copy via utilRequest.cgi)
func TestCopyFilesUtil(t *testing.T) {
	tests := []struct {
		name          string
		mockResponse  testutil.MockResponse
		sourcePath    string
		destPath      string
		sourceFiles   []string
		wantErr       bool
		expectedErr   api.ErrorCode
		assertRequest func(*testing.T, *http.Request)
	}{
		{
			name: "copy single file successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			sourcePath:  "/home",
			destPath:    "/backup",
			sourceFiles: []string{"file1.txt"},
			wantErr:     false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if src := r.URL.Query().Get("source_path"); src != "/home" {
					t.Errorf("Expected source_path=/home, got %s", src)
				}
				if dst := r.URL.Query().Get("dest_path"); dst != "/backup" {
					t.Errorf("Expected dest_path=/backup, got %s", dst)
				}
				if total := r.URL.Query().Get("source_total"); total != "1" {
					t.Errorf("Expected source_total=1, got %s", total)
				}
				files := r.URL.Query()["source_file"]
				if len(files) != 1 || files[0] != "file1.txt" {
					t.Errorf("Expected source_file=[file1.txt], got %v", files)
				}
			},
		},
		{
			name: "copy multiple files successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			sourcePath:  "/home",
			destPath:    "/backup",
			sourceFiles: []string{"file1.txt", "file2.txt", "file3.txt"},
			wantErr:     false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if total := r.URL.Query().Get("source_total"); total != "3" {
					t.Errorf("Expected source_total=3, got %s", total)
				}
				files := r.URL.Query()["source_file"]
				if len(files) != 3 {
					t.Errorf("Expected 3 source_file params, got %d", len(files))
				}
				expected := []string{"file1.txt", "file2.txt", "file3.txt"}
				for i, f := range expected {
					if i < len(files) && files[i] != f {
						t.Errorf("Expected source_file[%d]=%s, got %s", i, f, files[i])
					}
				}
			},
		},
		{
			name: "copy to same directory",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			sourcePath:  "/home",
			destPath:    "/home",
			sourceFiles: []string{"file1.txt"},
			wantErr:     false,
		},
		{
			name: "copy across different shares",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			sourcePath:  "/share1/folder",
			destPath:    "/share2/folder",
			sourceFiles: []string{"file.txt"},
			wantErr:     false,
		},
		{
			name:        "error when source files is empty",
			sourcePath:  "/home",
			destPath:    "/backup",
			sourceFiles: []string{},
			wantErr:     true,
			expectedErr: api.ErrInvalidPath,
		},
		{
			name:        "error when source files is nil",
			sourcePath:  "/home",
			destPath:    "/backup",
			sourceFiles: nil,
			wantErr:     true,
			expectedErr: api.ErrInvalidPath,
		},
		{
			name: "copy fails with status 0",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  0,
					"success": "true",
				},
			},
			sourcePath:  "/home",
			destPath:    "/backup",
			sourceFiles: []string{"file1.txt"},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "copy fails with success=false",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "false",
				},
			},
			sourcePath:  "/home",
			destPath:    "/backup",
			sourceFiles: []string{"file1.txt"},
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
			destPath:    "/backup",
			sourceFiles: []string{"file1.txt"},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			sourcePath:  "/home",
			destPath:    "/backup",
			sourceFiles: []string{"file1.txt"},
			wantErr:     true,
		},
		{
			name: "copy with many files",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			sourcePath:  "/home",
			destPath:    "/backup",
			sourceFiles: []string{"file1.txt", "file2.txt", "file3.txt", "file4.txt", "file5.txt"},
			wantErr:     false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if total := r.URL.Query().Get("source_total"); total != "5" {
					t.Errorf("Expected source_total=5, got %s", total)
				}
				files := r.URL.Query()["source_file"]
				if len(files) != 5 {
					t.Errorf("Expected 5 source_file params, got %d", len(files))
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

			err := fs.CopyFilesUtil(ctx, tt.sourcePath, tt.destPath, tt.sourceFiles)

			if tt.wantErr {
				if err == nil {
					t.Error("CopyFilesUtil() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("CopyFilesUtil() unexpected error = %v", err)
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

// TestCopyFilesUtilAuthentication tests authentication scenarios for CopyFilesUtil
func TestCopyFilesUtilAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		err := fs.CopyFilesUtil(ctx, "/home", "/backup", []string{"file.txt"})

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestMoveFilesUtil tests the MoveFilesUtil function (func=move via utilRequest.cgi)
func TestMoveFilesUtil(t *testing.T) {
	tests := []struct {
		name          string
		mockResponse  testutil.MockResponse
		sourcePath    string
		destPath      string
		sourceFiles   []string
		wantErr       bool
		expectedErr   api.ErrorCode
		assertRequest func(*testing.T, *http.Request)
	}{
		{
			name: "move single file successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			sourcePath:  "/home",
			destPath:    "/archive",
			sourceFiles: []string{"file1.txt"},
			wantErr:     false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if src := r.URL.Query().Get("source_path"); src != "/home" {
					t.Errorf("Expected source_path=/home, got %s", src)
				}
				if dst := r.URL.Query().Get("dest_path"); dst != "/archive" {
					t.Errorf("Expected dest_path=/archive, got %s", dst)
				}
				if total := r.URL.Query().Get("source_total"); total != "1" {
					t.Errorf("Expected source_total=1, got %s", total)
				}
				files := r.URL.Query()["source_file"]
				if len(files) != 1 || files[0] != "file1.txt" {
					t.Errorf("Expected source_file=[file1.txt], got %v", files)
				}
				if fn := r.URL.Query().Get("func"); fn != "move" {
					t.Errorf("Expected func=move, got %s", fn)
				}
			},
		},
		{
			name: "move multiple files successfully",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			sourcePath:  "/home",
			destPath:    "/archive",
			sourceFiles: []string{"file1.txt", "file2.txt", "file3.txt"},
			wantErr:     false,
			assertRequest: func(t *testing.T, r *http.Request) {
				t.Helper()
				if total := r.URL.Query().Get("source_total"); total != "3" {
					t.Errorf("Expected source_total=3, got %s", total)
				}
				files := r.URL.Query()["source_file"]
				if len(files) != 3 {
					t.Errorf("Expected 3 source_file params, got %d", len(files))
				}
				expected := []string{"file1.txt", "file2.txt", "file3.txt"}
				for i, f := range expected {
					if i < len(files) && files[i] != f {
						t.Errorf("Expected source_file[%d]=%s, got %s", i, f, files[i])
					}
				}
			},
		},
		{
			name: "move across directories",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			sourcePath:  "/downloads",
			destPath:    "/documents",
			sourceFiles: []string{"report.pdf"},
			wantErr:     false,
		},
		{
			name: "move to nested directory",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			sourcePath:  "/home",
			destPath:    "/archive/2024/january",
			sourceFiles: []string{"file.txt"},
			wantErr:     false,
		},
		{
			name: "move folder",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			sourcePath:  "/home",
			destPath:    "/backup",
			sourceFiles: []string{"myfolder"},
			wantErr:     false,
		},
		{
			name:        "error when source files is empty",
			sourcePath:  "/home",
			destPath:    "/archive",
			sourceFiles: []string{},
			wantErr:     true,
			expectedErr: api.ErrInvalidPath,
		},
		{
			name:        "error when source files is nil",
			sourcePath:  "/home",
			destPath:    "/archive",
			sourceFiles: nil,
			wantErr:     true,
			expectedErr: api.ErrInvalidPath,
		},
		{
			name: "move fails with status 0",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  0,
					"success": "true",
				},
			},
			sourcePath:  "/home",
			destPath:    "/archive",
			sourceFiles: []string{"file1.txt"},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "move fails with success=false",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "false",
				},
			},
			sourcePath:  "/home",
			destPath:    "/archive",
			sourceFiles: []string{"file1.txt"},
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
			destPath:    "/archive",
			sourceFiles: []string{"file1.txt"},
			wantErr:     true,
			expectedErr: api.ErrUnknown,
		},
		{
			name: "network error",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusInternalServerError,
				Error:      "internal server error",
			},
			sourcePath:  "/home",
			destPath:    "/archive",
			sourceFiles: []string{"file1.txt"},
			wantErr:     true,
		},
		{
			name: "move with special characters",
			mockResponse: testutil.MockResponse{
				StatusCode: http.StatusOK,
				Body: map[string]interface{}{
					"status":  1,
					"success": "true",
				},
			},
			sourcePath:  "/home",
			destPath:    "/archive",
			sourceFiles: []string{"file with spaces.txt", "file(1).txt"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", tt.mockResponse)

			ctx := context.Background()
			fs := NewFileStationService(client)

			err := fs.MoveFilesUtil(ctx, tt.sourcePath, tt.destPath, tt.sourceFiles)

			if tt.wantErr {
				if err == nil {
					t.Error("MoveFilesUtil() expected error, got nil")
				}
				if tt.expectedErr != 0 {
					assertAPIError(t, err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("MoveFilesUtil() unexpected error = %v", err)
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

// TestMoveFilesUtilAuthentication tests authentication scenarios for MoveFilesUtil
func TestMoveFilesUtilAuthentication(t *testing.T) {
	t.Run("returns error when not authenticated", func(t *testing.T) {
		client := setupUnauthenticatedClient(t)
		fs := NewFileStationService(client)

		ctx := context.Background()
		err := fs.MoveFilesUtil(ctx, "/home", "/archive", []string{"file.txt"})

		assertAPIError(t, err, api.ErrAuthFailed)
	})
}

// TestFileMethodsIntegration tests integration between different file methods
func TestFileMethodsIntegration(t *testing.T) {
	t.Run("full workflow: list, copy, move, delete", func(t *testing.T) {
		client, mockServer := setupTestClient(t)
		defer mockServer.Close()

		ctx := context.Background()
		fs := NewFileStationService(client)

		// Step 1: List files
		mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"total": 1,
				"datas": []File{
					{FileName: "original.txt", Path: "/home/original.txt", IsFolder: 0},
				},
			},
		})

		files, err := fs.ListFiles(ctx, "/home", nil)
		if err != nil {
			t.Fatalf("ListFiles() error = %v", err)
		}
		if len(files) != 1 {
			t.Fatalf("ListFiles() returned %d files, want 1", len(files))
		}

		// Step 2: Copy file
		mockServer.SetResponse("POST", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
			StatusCode: http.StatusOK,
			Body: map[string]interface{}{
				"status":  1,
				"success": "true",
			},
		})

		err = fs.CopyFilesUtil(ctx, "/home", "/backup", []string{"original.txt"})
		if err != nil {
			t.Fatalf("CopyFilesUtil() error = %v", err)
		}

		// Step 3: Rename in backup
		err = fs.RenameFileUtil(ctx, "/backup", "original.txt", "renamed.txt")
		if err != nil {
			t.Fatalf("RenameFileUtil() error = %v", err)
		}

		// Step 4: Move to archive
		err = fs.MoveFilesUtil(ctx, "/backup", "/archive", []string{"renamed.txt"})
		if err != nil {
			t.Fatalf("MoveFilesUtil() error = %v", err)
		}

		// Step 5: Delete from source
		err = fs.DeleteFiles(ctx, "/home", []string{"original.txt"})
		if err != nil {
			t.Fatalf("DeleteFiles() error = %v", err)
		}
	})
}

// TestContextCancellation tests context cancellation handling
func TestContextCancellation(t *testing.T) {
	tests := []struct {
		name   string
		testFn func(*testing.T, *FileStationService, context.Context)
	}{
		{
			name: "ListFiles respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				_, err := fs.ListFiles(ctx, "/home", nil)
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("ListFiles with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "DeleteFiles respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				err := fs.DeleteFiles(ctx, "/home", []string{"file.txt"})
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("DeleteFiles with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "RenameFileUtil respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				err := fs.RenameFileUtil(ctx, "/home", "old.txt", "new.txt")
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("RenameFileUtil with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "CopyFilesUtil respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				err := fs.CopyFilesUtil(ctx, "/home", "/backup", []string{"file.txt"})
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("CopyFilesUtil with canceled context returned error: %v", err)
				}
			},
		},
		{
			name: "MoveFilesUtil respects context cancellation",
			testFn: func(t *testing.T, fs *FileStationService, ctx context.Context) {
				err := fs.MoveFilesUtil(ctx, "/home", "/archive", []string{"file.txt"})
				if !errors.Is(err, context.Canceled) && err != nil {
					t.Logf("MoveFilesUtil with canceled context returned error: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mockServer := setupTestClient(t)
			defer mockServer.Close()

			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			fs := NewFileStationService(client)
			tt.testFn(t, fs, ctx)
		})
	}
}

// BenchmarkListFiles benchmarks the ListFiles function
func BenchmarkListFiles(b *testing.B) {
	client, mockServer := setupTestClient(&testing.T{})
	defer mockServer.Close()

	mockServer.SetResponse("GET", "/cgi-bin/filemanager/utilRequest.cgi", testutil.MockResponse{
		StatusCode: http.StatusOK,
		Body: map[string]interface{}{
			"total": 100,
			"datas": make([]File, 100),
		},
	})

	ctx := context.Background()
	fs := NewFileStationService(client)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = fs.ListFiles(ctx, "/home", nil)
	}
}
