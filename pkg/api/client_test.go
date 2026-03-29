package api

import (
	"context"
	"log/slog"
	"testing"

	"github.com/fatelei/qnap-filestation/internal/testutil"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Host:     "test.local",
				Port:     8080,
				Username: "admin",
				Password: "password",
			},
			wantErr: false,
		},
		{
			name:    "default config",
			config:  DefaultConfig(),
			wantErr: true,
		},
		{
			name: "missing host",
			config: &Config{
				Port:     8080,
				Username: "admin",
				Password: "password",
			},
			wantErr: true,
		},
		{
			name: "missing username",
			config: &Config{
				Host:     "test.local",
				Port:     8080,
				Password: "password",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			config: &Config{
				Host:     "test.local",
				Port:     8080,
				Username: "admin",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client")
			}
		})
	}
}

func TestClientLogin(t *testing.T) {
	mockServer := testutil.NewMockServer()
	defer mockServer.Close()

	url := mockServer.URL()
	config := &Config{
		Host:     url[7:], // Remove "http://"
		Port:     0,      // Don't set port, use the one from URL
		Username: "admin",
		Password: "password",
		Insecure: true,
		Logger:   slog.Default(),
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()

	// Test login
	err = client.Login(ctx)
	if err != nil {
		t.Errorf("Login() error = %v", err)
	}

	// Verify SID was set
	if client.getSID() == "" {
		t.Error("Login() did not set SID")
	}

	// Test logout
	err = client.Logout(ctx)
	if err != nil {
		t.Errorf("Logout() error = %v", err)
	}
}

func TestClientEnsureAuthenticated(t *testing.T) {
	tests := []struct {
		name    string
		sid     string
		wantErr bool
	}{
		{
			name:    "authenticated",
			sid:     "test-sid",
			wantErr: false,
		},
		{
			name:    "not authenticated",
			sid:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.Host = "test.local"
			config.Username = "admin"
			config.Password = "password"

			client, err := NewClient(config)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			client.setSID(tt.sid)

			err = client.ensureAuthenticated(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureAuthenticated() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
