package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Config represents the client configuration
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	Insecure bool
	Timeout  time.Duration
	Logger   *slog.Logger
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Port:     8080,
		Insecure: false,
		Timeout:  30 * time.Second,
		Logger:   slog.Default(),
	}
}

// Client is the QNAP API client
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	config     *Config
	logger     *slog.Logger

	sid       string
	sidMu     sync.RWMutex
}

// NewClient creates a new QNAP API client
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if cfg.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if cfg.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if cfg.Password == "" {
		return nil, fmt.Errorf("password is required")
	}

	// Build base URL
	scheme := "https"
	if cfg.Port == 80 || cfg.Insecure {
		scheme = "http"
	}

	var baseURL *url.URL
	var err error

	// If Port is 0, assume Host already contains port
	if cfg.Port == 0 {
		baseURL, err = url.Parse(fmt.Sprintf("%s://%s", scheme, cfg.Host))
	} else {
		baseURL, err = url.Parse(fmt.Sprintf("%s://%s:%d", scheme, cfg.Host, cfg.Port))
	}

	if err != nil {
		return nil, err
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: cfg.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.Insecure,
			},
		},
	}

	client := &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		config:     cfg,
		logger:     cfg.Logger,
	}

	return client, nil
}

// getSID returns the current session ID
func (c *Client) getSID() string {
	c.sidMu.RLock()
	defer c.sidMu.RUnlock()
	return c.sid
}

// setSID sets the session ID
func (c *Client) setSID(sid string) {
	c.sidMu.Lock()
	defer c.sidMu.Unlock()
	c.sid = sid
}

// Login authenticates with the QNAP device
func (c *Client) Login(ctx context.Context) error {
	endpoint := "/auth.cgi"
	params := map[string]string{
		"api":     "SYNO.API.Auth",
		"method":  "login",
		"version": "2",
		"account": c.config.Username,
		"passwd":  c.config.Password,
		"session": "FileStation",
		"format":  "sid",
	}

	resp, err := c.doRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Success int `json:"success"`
		Data    struct {
			SID string `json:"sid"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return WrapAPIError(ErrUnknown, "failed to parse login response", err)
	}

	if result.Success != 1 {
		return NewAPIError(ErrAuthFailed, "login failed")
	}

	if result.Data.SID == "" {
		return NewAPIError(ErrAuthFailed, "no SID returned")
	}

	c.setSID(result.Data.SID)
	c.logger.Info("Successfully authenticated", "sid", result.Data.SID)

	return nil
}

// Logout ends the session
func (c *Client) Logout(ctx context.Context) error {
	endpoint := "/auth.cgi"
	params := map[string]string{
		"api":     "SYNO.API.Auth",
		"method":  "logout",
		"version": "2",
		"session": "FileStation",
	}

	sid := c.getSID()
	if sid == "" {
		return nil // Already logged out
	}

	resp, err := c.doRequest(ctx, "GET", endpoint, params, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.setSID("")
	c.logger.Info("Successfully logged out")

	return nil
}

// doRequest executes an HTTP request
func (c *Client) doRequest(ctx context.Context, method, endpoint string, queryParams map[string]string, body interface{}) (*http.Response, error) {
	builder, err := NewRequestBuilder(c.baseURL.String(), c.httpClient)
	if err != nil {
		return nil, err
	}

	builder.SetSID(c.getSID())

	req, err := builder.BuildRequest(ctx, method, endpoint, queryParams, body)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("API request", "method", method, "endpoint", endpoint, "url", req.URL.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, WrapAPIError(ErrNetwork, "network error", err)
	}

	return resp, nil
}

// DoRequest is an exported version of doRequest for use by sub-packages
func (c *Client) DoRequest(ctx context.Context, method, endpoint string, queryParams map[string]string, body interface{}) (*http.Response, error) {
	return c.doRequest(ctx, method, endpoint, queryParams, body)
}

// GetSID returns the current session ID
func (c *Client) GetSID() string {
	return c.getSID()
}

// SetSID sets the session ID (for testing purposes)
func (c *Client) SetSID(sid string) {
	c.setSID(sid)
}

// GetBaseURL returns the base URL
func (c *Client) GetBaseURL() *url.URL {
	return c.baseURL
}

// GetHTTPClient returns the HTTP client
func (c *Client) GetHTTPClient() *http.Client {
	return c.httpClient
}

// ensureAuthenticated ensures the client has a valid session
func (c *Client) ensureAuthenticated(ctx context.Context) error {
	sid := c.getSID()
	if sid == "" {
		return WrapAPIError(ErrAuthFailed, "not authenticated", nil)
	}
	return nil
}

// GetLogger returns the client's logger
func (c *Client) GetLogger() *slog.Logger {
	return c.logger
}
