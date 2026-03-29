package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// MockServer is a mock HTTP server for testing
type MockServer struct {
	server      *httptest.Server
	requests    []*http.Request
	requestsMu  sync.Mutex
	responses   map[string]MockResponse
	sid         string
	authCalled  bool
	authCalledMu sync.Mutex
}

// MockResponse represents a mock response
type MockResponse struct {
	StatusCode int
	Body       interface{}
	Error      string
}

// NewMockServer creates a new mock server
func NewMockServer() *MockServer {
	ms := &MockServer{
		responses: make(map[string]MockResponse),
	}

	ms.server = httptest.NewServer(http.HandlerFunc(ms.handler))

	return ms
}

// handler handles HTTP requests
func (ms *MockServer) handler(w http.ResponseWriter, r *http.Request) {
	// Record request
	ms.requestsMu.Lock()
	ms.requests = append(ms.requests, r)
	ms.requestsMu.Unlock()

	path := r.URL.Path
	method := r.Method

	// Handle authentication
	if strings.Contains(path, "/auth.cgi") {
		ms.authCalledMu.Lock()
		ms.authCalled = true
		ms.authCalledMu.Unlock()

		sid := "test-sid-12345"
		ms.sid = sid

		response := map[string]interface{}{
			"success": 1,
			"data": map[string]string{
				"sid": sid,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check for custom response
	key := method + ":" + path
	if resp, ok := ms.responses[key]; ok {
		if resp.Error != "" {
			http.Error(w, resp.Error, resp.StatusCode)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)

		if resp.Body != nil {
			json.NewEncoder(w).Encode(resp.Body)
		}
		return
	}

	// Default response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": 1,
		"data":    map[string]interface{}{},
	})
}

// URL returns the server's URL
func (ms *MockServer) URL() string {
	return ms.server.URL
}

// Close closes the server
func (ms *MockServer) Close() {
	ms.server.Close()
}

// SetResponse sets a custom response for a path
func (ms *MockServer) SetResponse(method, path string, response MockResponse) {
	key := method + ":" + path
	ms.responses[key] = response
}

// GetSID returns the current SID
func (ms *MockServer) GetSID() string {
	return ms.sid
}

// WasAuthCalled returns true if auth was called
func (ms *MockServer) WasAuthCalled() bool {
	ms.authCalledMu.Lock()
	defer ms.authCalledMu.Unlock()
	return ms.authCalled
}

// GetRequests returns all recorded requests
func (ms *MockServer) GetRequests() []*http.Request {
	ms.requestsMu.Lock()
	defer ms.requestsMu.Unlock()
	return ms.requests
}

// GetLastRequest returns the last request
func (ms *MockServer) GetLastRequest() *http.Request {
	reqs := ms.GetRequests()
	if len(reqs) == 0 {
		return nil
	}
	return reqs[len(reqs)-1]
}

// ReadBody reads the request body
func ReadBody(r *http.Request) string {
	body, _ := io.ReadAll(r.Body)
	return string(body)
}

// JSONBody creates a JSON response body
func JSONBody(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// SuccessResponse creates a success response
func SuccessResponse(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"success": 1,
		"data":    data,
	}
}

// ErrorResponse creates an error response
func ErrorResponse(code int, message string) map[string]interface{} {
	return map[string]interface{}{
		"success":    0,
		"error_code": code,
		"error_msg":  message,
	}
}
