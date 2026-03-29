package filestation

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/fatelei/qnap-filestation/internal/testutil"
	"github.com/fatelei/qnap-filestation/pkg/api"
)

// setupTestClient creates a new client and mock server for testing
func setupTestClient(t *testing.T) (*api.Client, *testutil.MockServer) {
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

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	return client, mockServer
}

// setupUnauthenticatedClient creates a client without authentication for error testing
func setupUnauthenticatedClient(t *testing.T) *api.Client {
	t.Helper()

	config := &api.Config{
		Host:     "localhost",
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

// assertAPIError checks if the error is an APIError with the expected code
func assertAPIError(t *testing.T, err error, expectedCode api.ErrorCode) {
	t.Helper()

	if err == nil {
		t.Errorf("Expected error with code %d, but got nil", expectedCode)
		return
	}

	apiErr, ok := err.(*api.APIError)
	if !ok {
		t.Errorf("Expected APIError, got %T", err)
		return
	}

	if apiErr.Code != expectedCode {
		t.Errorf("Expected error code %d, got %d (message: %s)", expectedCode, apiErr.Code, apiErr.Message)
	}
}
