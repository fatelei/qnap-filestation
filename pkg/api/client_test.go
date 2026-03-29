package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestNewClient validates client creation with various configurations
func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		wantErr     bool
		errContains string
		validate    func(t *testing.T, c *Client)
	}{
		{
			name: "valid config with all fields",
			config: &Config{
				Host:     "qnap.example.com",
				Port:     8080,
				Username: "admin",
				Password: "secret123",
				Insecure: false,
				Timeout:  60 * time.Second,
			},
			wantErr: false,
			validate: func(t *testing.T, c *Client) {
				if c == nil {
					t.Fatal("client is nil")
				}
				if c.baseURL == nil {
					t.Fatal("baseURL is nil")
				}
				if got := c.baseURL.String(); got != "https://qnap.example.com:8080" {
					t.Errorf("baseURL = %s, want https://qnap.example.com:8080", got)
				}
				if c.httpClient.Timeout != 60*time.Second {
					t.Errorf("timeout = %v, want 60s", c.httpClient.Timeout)
				}
				tlsConfig := c.httpClient.Transport.(*http.Transport).TLSClientConfig
				if tlsConfig.InsecureSkipVerify {
					t.Error("InsecureSkipVerify should be false")
				}
			},
		},
		{
			name: "valid config with insecure flag",
			config: &Config{
				Host:     "qnap.example.com",
				Port:     8080,
				Username: "admin",
				Password: "secret123",
				Insecure: true,
			},
			wantErr: false,
			validate: func(t *testing.T, c *Client) {
				if c.baseURL.Scheme != "http" {
					t.Errorf("scheme = %s, want http", c.baseURL.Scheme)
				}
				tlsConfig := c.httpClient.Transport.(*http.Transport).TLSClientConfig
				if !tlsConfig.InsecureSkipVerify {
					t.Error("InsecureSkipVerify should be true")
				}
			},
		},
		{
			name: "valid config with port 80 uses http",
			config: &Config{
				Host:     "qnap.example.com",
				Port:     80,
				Username: "admin",
				Password: "secret123",
			},
			wantErr: false,
			validate: func(t *testing.T, c *Client) {
				if c.baseURL.Scheme != "http" {
					t.Errorf("scheme = %s, want http for port 80", c.baseURL.Scheme)
				}
			},
		},
		{
			name: "valid config with port 0 (host contains port)",
			config: &Config{
				Host:     "qnap.example.com:443",
				Port:     0,
				Username: "admin",
				Password: "secret123",
			},
			wantErr: false,
			validate: func(t *testing.T, c *Client) {
				if got := c.baseURL.String(); got != "https://qnap.example.com:443" {
					t.Errorf("baseURL = %s, want https://qnap.example.com:443", got)
				}
			},
		},
		{
			name: "valid config with default values from nil config",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.Host = "test.local"
				cfg.Username = "user"
				cfg.Password = "pass"
				return cfg
			}(),
			wantErr: false,
			validate: func(t *testing.T, c *Client) {
				if c.httpClient.Timeout != 30*time.Second {
					t.Errorf("default timeout = %v, want 30s", c.httpClient.Timeout)
				}
				if c.logger == nil {
					t.Error("logger should be set")
				}
			},
		},
		{
			name:        "nil config uses defaults",
			config:      nil,
			wantErr:     true,
			errContains: "host is required",
		},
		{
			name: "missing host",
			config: &Config{
				Port:     8080,
				Username: "admin",
				Password: "secret123",
			},
			wantErr:     true,
			errContains: "host is required",
		},
		{
			name: "empty host",
			config: &Config{
				Host:     "",
				Port:     8080,
				Username: "admin",
				Password: "secret123",
			},
			wantErr:     true,
			errContains: "host is required",
		},
		{
			name: "missing username",
			config: &Config{
				Host:     "qnap.example.com",
				Port:     8080,
				Password: "secret123",
			},
			wantErr:     true,
			errContains: "username is required",
		},
		{
			name: "empty username",
			config: &Config{
				Host:     "qnap.example.com",
				Port:     8080,
				Username: "",
				Password: "secret123",
			},
			wantErr:     true,
			errContains: "username is required",
		},
		{
			name: "missing password",
			config: &Config{
				Host:     "qnap.example.com",
				Port:     8080,
				Username: "admin",
			},
			wantErr:     true,
			errContains: "password is required",
		},
		{
			name: "empty password",
			config: &Config{
				Host:     "qnap.example.com",
				Port:     8080,
				Username: "admin",
				Password: "",
			},
			wantErr:     true,
			errContains: "password is required",
		},
		{
			name: "invalid hostname with control characters",
			config: &Config{
				Host:     "test\x00.local",
				Port:     8080,
				Username: "admin",
				Password: "secret123",
			},
			wantErr: true,
		},
		{
			name: "custom logger",
			config: &Config{
				Host:     "qnap.example.com",
				Port:     8080,
				Username: "admin",
				Password: "secret123",
				Logger:   slog.Default(),
			},
			wantErr: false,
			validate: func(t *testing.T, c *Client) {
				if c.logger == nil {
					t.Error("logger should be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewClient() expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewClient() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("NewClient() unexpected error = %v", err)
				return
			}

			if client == nil {
				t.Fatal("NewClient() returned nil client")
			}

			if tt.validate != nil {
				tt.validate(t, client)
			}
		})
	}
}

// TestDefaultConfig validates the default configuration values
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.Insecure != false {
		t.Errorf("Insecure = %v, want false", cfg.Insecure)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
	}
	if cfg.Logger == nil {
		t.Error("Logger should be set to default")
	}
}

// TestClientLogin_SuccessfulLogin tests successful authentication
func TestClientLogin_SuccessfulLogin(t *testing.T) {
	var receivedUser, receivedPwd string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cgi-bin/authLogin.cgi" {
			t.Errorf("path = %s, want /cgi-bin/authLogin.cgi", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		receivedUser = r.FormValue("user")
		receivedPwd = r.FormValue("pwd")
		dontVerify := r.FormValue("dont_verify_2sv")
		if dontVerify != "1" {
			t.Errorf("dont_verify_2sv = %s, want 1", dontVerify)
		}

		// Return successful XML response
		w.Header().Set("Content-Type", "application/xml")
		xmlResponse := `<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid-abc123xyz</authSid>
</QDocRoot>`
		w.Write([]byte(xmlResponse))
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(u.Port())
	client, err := NewClient(&Config{
		Host:     u.Hostname(),
		Port:     port,
		Username: "testuser",
		Password: "testpass",
		Insecure: true,
		Logger:   slog.Default(),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Errorf("Login() error = %v", err)
	}

	// Verify credentials were sent correctly
	if receivedUser != "testuser" {
		t.Errorf("received user = %s, want testuser", receivedUser)
	}

	// Verify password was base64 encoded
	expectedPwd := base64.StdEncoding.EncodeToString([]byte("testpass"))
	if receivedPwd != expectedPwd {
		t.Errorf("received pwd = %s, want %s (base64 encoded)", receivedPwd, expectedPwd)
	}

	// Verify SID was stored
	if got := client.GetSID(); got != "test-sid-abc123xyz" {
		t.Errorf("SID = %s, want test-sid-abc123xyz", got)
	}
}

// TestClientLogin_FailedLogin tests authentication failure scenarios
func TestClientLogin_FailedLogin(t *testing.T) {
	tests := []struct {
		name           string
		xmlResponse    string
		expectedErr    *APIError
		expectedCode   ErrorCode
		expectedErrMsg string
	}{
		{
			name: "wrong password - authPassed is 0",
			xmlResponse: `<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>0</authPassed>
	<authSid></authSid>
</QDocRoot>`,
			expectedCode:   ErrAuthFailed,
			expectedErrMsg: "login failed: invalid credentials",
		},
		{
			name: "empty SID returned",
			xmlResponse: `<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid></authSid>
</QDocRoot>`,
			expectedCode:   ErrAuthFailed,
			expectedErrMsg: "no SID returned",
		},
		{
			name: "missing authPassed field",
			xmlResponse: `<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authSid>test-sid</authSid>
</QDocRoot>`,
			expectedCode:   ErrAuthFailed,
			expectedErrMsg: "login failed: invalid credentials",
		},
		{
			name: "missing authSid field",
			xmlResponse: `<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
</QDocRoot>`,
			expectedCode:   ErrAuthFailed,
			expectedErrMsg: "no SID returned",
		},
		{
			name: "malformed XML",
			xmlResponse: `<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid
</QDocRoot>`,
			expectedCode: ErrUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				w.Write([]byte(tt.xmlResponse))
			}))
			defer server.Close()

			u, _ := url.Parse(server.URL)
			port, _ := strconv.Atoi(u.Port())
			client, err := NewClient(&Config{
				Host:     u.Hostname(),
				Port:     port,
				Username: "testuser",
				Password: "testpass",
				Insecure: true,
				Logger:   slog.Default(),
			})
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			ctx := context.Background()
			err = client.Login(ctx)

			if err == nil {
				t.Fatal("Login() expected error, got nil")
			}

			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("error type = %T, want *APIError", err)
			}

			if apiErr.Code != tt.expectedCode {
				t.Errorf("error code = %d, want %d", apiErr.Code, tt.expectedCode)
			}

			if tt.expectedErrMsg != "" && !strings.Contains(apiErr.Message, tt.expectedErrMsg) {
				t.Errorf("error message = %q, want containing %q", apiErr.Message, tt.expectedErrMsg)
			}

			// Verify SID was not set on failed login
			if got := client.GetSID(); got != "" {
				t.Errorf("SID = %s, want empty on failed login", got)
			}
		})
	}
}

// TestClientLogin_NetworkError tests network error handling
func TestClientLogin_NetworkError(t *testing.T) {
	// Create a client pointing to a non-existent server
	client, err := NewClient(&Config{
		Host:     "localhost:59999", // Non-existent port
		Username: "testuser",
		Password: "testpass",
		Insecure: true,
		Timeout:  100 * time.Millisecond,
		Logger:   slog.Default(),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	err = client.Login(ctx)

	if err == nil {
		t.Fatal("Login() expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}

	if apiErr.Code != ErrNetwork {
		t.Errorf("error code = %d, want %d (ErrNetwork)", apiErr.Code, ErrNetwork)
	}
	if !strings.Contains(apiErr.Message, "network error") {
		t.Errorf("error message = %q, want containing 'network error'", apiErr.Message)
	}
}

// TestClientLogin_ContextCancellation tests context cancellation
func TestClientLogin_ContextCancellation(t *testing.T) {
	blockCh := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blockCh // Block until test completes
	}))
	defer server.Close()
	defer close(blockCh)

	u, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(u.Port())
	client, err := NewClient(&Config{
		Host:     u.Hostname(),
		Port:     port,
		Username: "testuser",
		Password: "testpass",
		Insecure: true,
		Logger:   slog.Default(),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = client.Login(ctx)
	if err == nil {
		t.Fatal("Login() expected error for canceled context, got nil")
	}
}

// TestClientLogin_XMLParsing tests various XML response formats
func TestClientLogin_XMLParsing(t *testing.T) {
	tests := []struct {
		name        string
		xmlResponse string
		wantSID     string
		wantErr     bool
	}{
		{
			name: "standard valid XML",
			xmlResponse: `<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>mysid123</authSid>
</QDocRoot>`,
			wantSID: "mysid123",
			wantErr: false,
		},
		{
			name: "XML with extra whitespace - should fail due to whitespace not being trimmed",
			xmlResponse: `<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
  <authPassed>  1  </authPassed>
  <authSid>  mysid456  </authSid>
</QDocRoot>`,
			wantSID: "",
			wantErr: true, // Go XML decoder doesn't trim whitespace, so "  1  " != "1"
		},
		{
			name: "XML with extra fields",
			xmlResponse: `<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>mysid789</authSid>
	<username>admin</username>
	<role>2</role>
</QDocRoot>`,
			wantSID: "mysid789",
			wantErr: false,
		},
		{
			name: "XML with CDATA",
			xmlResponse: `<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed><![CDATA[1]]></authPassed>
	<authSid><![CDATA[sid-cdata]]></authSid>
</QDocRoot>`,
			wantSID: "sid-cdata",
			wantErr: false,
		},
		{
			name:        "empty XML response",
			xmlResponse: ``,
			wantErr:     true,
		},
		{
			name: "XML without QDocRoot",
			xmlResponse: `<?xml version="1.0" encoding="UTF-8"?>
<OtherRoot>
	<authPassed>1</authPassed>
	<authSid>test</authSid>
</OtherRoot>`,
			wantSID: "",
			wantErr: true, // authPassed will be empty, not "1"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.xmlResponse != "" {
					w.Header().Set("Content-Type", "application/xml")
					w.Write([]byte(tt.xmlResponse))
				}
			}))
			defer server.Close()

			u, _ := url.Parse(server.URL)
			port, _ := strconv.Atoi(u.Port())
			client, err := NewClient(&Config{
				Host:     u.Hostname(),
				Port:     port,
				Username: "testuser",
				Password: "testpass",
				Insecure: true,
				Logger:   slog.Default(),
			})
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			ctx := context.Background()
			err = client.Login(ctx)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Login() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Login() unexpected error = %v", err)
				return
			}

			if got := client.GetSID(); got != tt.wantSID {
				t.Errorf("SID = %q, want %q", got, tt.wantSID)
			}
		})
	}
}

// TestClientLogout tests logout functionality
func TestClientLogout(t *testing.T) {
	tests := []struct {
		name           string
		initialSID     string
		wantErr        bool
		expectRequest  bool
		serverResponse int
	}{
		{
			name:           "successful logout with SID",
			initialSID:     "test-sid-123",
			wantErr:        false,
			expectRequest:  true,
			serverResponse: 200,
		},
		{
			name:           "no SID - should return nil without request",
			initialSID:     "",
			wantErr:        false,
			expectRequest:  false,
			serverResponse: 200,
		},
		{
			name:           "server error during logout - logout still succeeds (no HTTP status check)",
			initialSID:     "test-sid-123",
			wantErr:        false,
			expectRequest:  true,
			serverResponse: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestReceived := false

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestReceived = true

				// Verify endpoint
				if r.URL.Path != "/auth.cgi" {
					t.Errorf("path = %s, want /auth.cgi", r.URL.Path)
				}

				// Verify query params
				if r.URL.Query().Get("api") != "SYNO.API.Auth" {
					t.Errorf("api param = %s, want SYNO.API.Auth", r.URL.Query().Get("api"))
				}
				if r.URL.Query().Get("method") != "logout" {
					t.Errorf("method param = %s, want logout", r.URL.Query().Get("method"))
				}
				if r.URL.Query().Get("version") != "2" {
					t.Errorf("version param = %s, want 2", r.URL.Query().Get("version"))
				}
				if r.URL.Query().Get("session") != "FileStation" {
					t.Errorf("session param = %s, want FileStation", r.URL.Query().Get("session"))
				}

				w.WriteHeader(tt.serverResponse)
				if tt.serverResponse == 200 {
					w.Write([]byte(`{"success":true}`))
				}
			}))
			defer server.Close()

			u, _ := url.Parse(server.URL)
			port, _ := strconv.Atoi(u.Port())
			client, err := NewClient(&Config{
				Host:     u.Hostname(),
				Port:     port,
				Username: "testuser",
				Password: "testpass",
				Insecure: true,
				Logger:   slog.Default(),
			})
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			client.SetSID(tt.initialSID)

			ctx := context.Background()
			err = client.Logout(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("Logout() error = %v, wantErr %v", err, tt.wantErr)
			}

			if requestReceived != tt.expectRequest {
				t.Errorf("request received = %v, want %v", requestReceived, tt.expectRequest)
			}

			// Verify SID is cleared regardless of error
			if got := client.GetSID(); got != "" {
				t.Errorf("SID after logout = %q, want empty", got)
			}
		})
	}
}

// TestClientLogout_NetworkError tests logout with network errors
func TestClientLogout_NetworkError(t *testing.T) {
	client, err := NewClient(&Config{
		Host:     "localhost:59999",
		Username: "testuser",
		Password: "testpass",
		Insecure: true,
		Timeout:  100 * time.Millisecond,
		Logger:   slog.Default(),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	client.SetSID("test-sid")

	ctx := context.Background()
	err = client.Logout(ctx)

	if err == nil {
		t.Fatal("Logout() expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}

	if apiErr.Code != ErrNetwork {
		t.Errorf("error code = %d, want %d (ErrNetwork)", apiErr.Code, ErrNetwork)
	}

	// Note: SID is NOT cleared on network error because doRequest fails before SID clearing
	// This is the actual behavior of the implementation
	if got := client.GetSID(); got != "test-sid" {
		t.Errorf("SID after failed logout = %q, want 'test-sid' (not cleared on error)", got)
	}
}

// TestSIDThreadSafety tests concurrent SID access
func TestSIDThreadSafety(t *testing.T) {
	client, err := NewClient(&Config{
		Host:     "test.local",
		Port:     8080,
		Username: "admin",
		Password: "password",
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	const numGoroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // readers and writers

	// Writer goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				sid := fmt.Sprintf("sid-%d-%d", id, j)
				client.SetSID(sid)
			}
		}(i)
	}

	// Reader goroutines
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = client.GetSID()
				_ = client.GetSID()
			}
		}()
	}

	wg.Wait()

	// Final SID should be one of the values set
	finalSID := client.GetSID()
	if finalSID == "" {
		t.Error("Final SID should not be empty")
	}
}

// TestGetSetSID tests basic get/set SID operations
func TestGetSetSID(t *testing.T) {
	tests := []struct {
		name    string
		setSID  string
		wantSID string
	}{
		{
			name:    "set and get empty SID",
			setSID:  "",
			wantSID: "",
		},
		{
			name:    "set and get normal SID",
			setSID:  "abc123def456",
			wantSID: "abc123def456",
		},
		{
			name:    "set and get SID with special characters",
			setSID:  "sid-with.special@characters#123",
			wantSID: "sid-with.special@characters#123",
		},
		{
			name:    "set and get very long SID",
			setSID:  strings.Repeat("a", 1000),
			wantSID: strings.Repeat("a", 1000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(&Config{
				Host:     "test.local",
				Port:     8080,
				Username: "admin",
				Password: "password",
			})
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			client.SetSID(tt.setSID)
			if got := client.GetSID(); got != tt.wantSID {
				t.Errorf("GetSID() = %q, want %q", got, tt.wantSID)
			}
		})
	}
}

// TestClient_GetBaseURL tests GetBaseURL method
func TestClient_GetBaseURL(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		wantScheme string
		wantHost   string
	}{
		{
			name: "https with port",
			config: &Config{
				Host:     "example.com",
				Port:     443,
				Username: "admin",
				Password: "pass",
			},
			wantScheme: "https",
			wantHost:   "example.com:443",
		},
		{
			name: "http with insecure flag",
			config: &Config{
				Host:     "example.com",
				Port:     8080,
				Username: "admin",
				Password: "pass",
				Insecure: true,
			},
			wantScheme: "http",
			wantHost:   "example.com:8080",
		},
		{
			name: "http with port 80",
			config: &Config{
				Host:     "example.com",
				Port:     80,
				Username: "admin",
				Password: "pass",
			},
			wantScheme: "http",
			wantHost:   "example.com:80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			baseURL := client.GetBaseURL()
			if baseURL.Scheme != tt.wantScheme {
				t.Errorf("Scheme = %s, want %s", baseURL.Scheme, tt.wantScheme)
			}
			if baseURL.Host != tt.wantHost {
				t.Errorf("Host = %s, want %s", baseURL.Host, tt.wantHost)
			}
		})
	}
}

// TestClient_GetHTTPClient tests GetHTTPClient method
func TestClient_GetHTTPClient(t *testing.T) {
	config := &Config{
		Host:     "example.com",
		Port:     8080,
		Username: "admin",
		Password: "pass",
		Timeout:  45 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	httpClient := client.GetHTTPClient()
	if httpClient == nil {
		t.Fatal("GetHTTPClient() returned nil")
	}
	if httpClient.Timeout != 45*time.Second {
		t.Errorf("Timeout = %v, want 45s", httpClient.Timeout)
	}
}

// TestClient_ensureAuthenticated tests ensureAuthenticated method
func TestClient_ensureAuthenticated(t *testing.T) {
	tests := []struct {
		name    string
		sid     string
		wantErr bool
		errCode ErrorCode
	}{
		{
			name:    "with valid SID",
			sid:     "valid-sid-123",
			wantErr: false,
		},
		{
			name:    "with empty SID",
			sid:     "",
			wantErr: true,
			errCode: ErrAuthFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(&Config{
				Host:     "test.local",
				Port:     8080,
				Username: "admin",
				Password: "password",
			})
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			client.SetSID(tt.sid)

			ctx := context.Background()
			err = client.ensureAuthenticated(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("ensureAuthenticated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				apiErr, ok := err.(*APIError)
				if !ok {
					t.Fatalf("error type = %T, want *APIError", err)
				}
				if apiErr.Code != tt.errCode {
					t.Errorf("error code = %d, want %d", apiErr.Code, tt.errCode)
				}
			}
		})
	}
}

// TestGetLogger tests GetLogger method
func TestGetLogger(t *testing.T) {
	customLogger := slog.Default()

	config := &Config{
		Host:     "test.local",
		Port:     8080,
		Username: "admin",
		Password: "password",
		Logger:   customLogger,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	logger := client.GetLogger()
	if logger != customLogger {
		t.Error("GetLogger() returned different logger")
	}
}

// TestClientLogin_RequestHeaders tests that Login sends correct headers
func TestClientLogin_RequestHeaders(t *testing.T) {
	var contentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")

		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid</authSid>
</QDocRoot>`))
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(u.Port())
	client, err := NewClient(&Config{
		Host:     u.Hostname(),
		Port:     port,
		Username: "admin",
		Password: "pass",
		Insecure: true,
		Logger:   slog.Default(),
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx := context.Background()
	if err := client.Login(ctx); err != nil {
		t.Errorf("Login() error = %v", err)
	}

	if contentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %s, want application/x-www-form-urlencoded", contentType)
	}
}

// TestNewClient_TLSConfig tests TLS configuration
func TestNewClient_TLSConfig(t *testing.T) {
	tests := []struct {
		name                   string
		insecure               bool
		wantInsecureSkipVerify bool
	}{
		{
			name:                   "secure connection",
			insecure:               false,
			wantInsecureSkipVerify: false,
		},
		{
			name:                   "insecure connection",
			insecure:               true,
			wantInsecureSkipVerify: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(&Config{
				Host:     "test.local",
				Port:     8080,
				Username: "admin",
				Password: "password",
				Insecure: tt.insecure,
			})
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			transport := client.GetHTTPClient().Transport.(*http.Transport)
			got := transport.TLSClientConfig.InsecureSkipVerify
			if got != tt.wantInsecureSkipVerify {
				t.Errorf("InsecureSkipVerify = %v, want %v", got, tt.wantInsecureSkipVerify)
			}
		})
	}
}

// TestClientLogin_PasswordEncoding verifies password is base64 encoded correctly
func TestClientLogin_PasswordEncoding(t *testing.T) {
	passwords := []struct {
		plain   string
		encoded string
	}{
		{"simple", "c2ltcGxl"},
		{"with spaces", "d2l0aCBzcGFjZXM="},
		{"special!@#$%^&*()", "c3BlY2lhbCFAIyQlXiYqKCk="},
		{"unicode中文", "dW5pY29kZeS4reaWhw=="},
		{"12345678", "MTIzNDU2Nzg="},
	}

	for _, tt := range passwords {
		t.Run(tt.plain, func(t *testing.T) {
			var receivedPwd string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.ParseForm()
				receivedPwd = r.FormValue("pwd")

				w.Header().Set("Content-Type", "application/xml")
				w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>test-sid</authSid>
</QDocRoot>`))
			}))
			defer server.Close()

			u, _ := url.Parse(server.URL)
			port, _ := strconv.Atoi(u.Port())
			client, err := NewClient(&Config{
				Host:     u.Hostname(),
				Port:     port,
				Username: "admin",
				Password: tt.plain,
				Insecure: true,
				Logger:   slog.Default(),
			})
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			ctx := context.Background()
			if err := client.Login(ctx); err != nil {
				t.Errorf("Login() error = %v", err)
			}

			if receivedPwd != tt.encoded {
				t.Errorf("password encoding = %s, want %s", receivedPwd, tt.encoded)
			}
		})
	}
}

// BenchmarkSIDConcurrentAccess benchmarks concurrent SID access
func BenchmarkSIDConcurrentAccess(b *testing.B) {
	client, _ := NewClient(&Config{
		Host:     "test.local",
		Port:     8080,
		Username: "admin",
		Password: "password",
	})

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				client.SetSID("sid")
			} else {
				_ = client.GetSID()
			}
			i++
		}
	})
}
