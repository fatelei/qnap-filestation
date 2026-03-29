package filestation

import (
	"context"
	"log/slog"
	"testing"

	"github.com/fatelei/qnap-filestation/pkg/api"
	"github.com/fatelei/qnap-filestation/internal/testutil"
)

func setupTestClient(t *testing.T) (*api.Client, *testutil.MockServer) {
	mockServer := testutil.NewMockServer()

	url := mockServer.URL()
	host := url[7:] // Remove "http://"

	config := &api.Config{
		Host:     host,
		Port:     0,     // Don't set port, use the one from URL
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

	return client, mockServer
}

func TestListFiles(t *testing.T) {
	client, mockServer := setupTestClient(t)
	defer mockServer.Close()

	ctx := context.Background()

	fs := NewFileStationService(client)

	mockServer.SetResponse("GET", "/filestation/list.cgi", testutil.MockResponse{
		StatusCode: 200,
		Body: testutil.SuccessResponse(map[string]interface{}{
			"items": []File{
				{
					ID:       "1",
					Name:     "test.txt",
					Path:     "/home/test.txt",
					Size:     1024,
					IsFile:   true,
					IsFolder: false,
				},
			},
			"total":  1,
			"offset": 0,
		}),
	})

	files, err := fs.ListFiles(ctx, "/home", nil)
	if err != nil {
		t.Errorf("ListFiles() error = %v", err)
	}

	if len(files) != 1 {
		t.Errorf("ListFiles() returned %d files, want 1", len(files))
	}

	if len(files) > 0 && files[0].Name != "test.txt" {
		t.Errorf("ListFiles() returned file name %s, want test.txt", files[0].Name)
	}
}
