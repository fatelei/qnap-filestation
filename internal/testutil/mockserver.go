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
// nolint:govet,fieldalignment // field order chosen for clarity in tests
type MockServer struct {
	// Order fields to reduce padding per govet fieldalignment
	requests     []*http.Request
	sid          string
	responses    map[string]MockResponse
	server       *httptest.Server
	requestsMu   sync.Mutex
	authCalledMu sync.Mutex
	authCalled   bool
}

// MockResponse represents a mock response
// nolint:govet,fieldalignment // small test struct; packing not critical
type MockResponse struct {
	Body       interface{}
	Error      string
	StatusCode int
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
	if strings.Contains(path, "/auth.cgi") || strings.Contains(path, "/authLogin.cgi") {
		ms.authCalledMu.Lock()
		ms.authCalled = true
		ms.authCalledMu.Unlock()

		sid := "test-sid-12345"
		ms.sid = sid

		// Return XML response for QNAP authLogin.cgi
		if strings.Contains(path, "/authLogin.cgi") {
			w.Header().Set("Content-Type", "application/xml")
			xmlResponse := `<?xml version="1.0" encoding="UTF-8"?>
<QDocRoot>
	<authPassed>1</authPassed>
	<authSid>` + sid + `</authSid>
</QDocRoot>`
			if _, err := w.Write([]byte(xmlResponse)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// Return JSON response for standard auth.cgi
		response := map[string]interface{}{
			"success": 1,
			"data": map[string]string{
				"sid": sid,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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
			if err := json.NewEncoder(w).Encode(resp.Body); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
		return
	}

	// Default response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": 1,
		"data":    map[string]interface{}{},
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
