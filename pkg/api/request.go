package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// RequestBuilder builds HTTP requests for the QNAP API
type RequestBuilder struct {
	baseURL    *url.URL
	httpClient *http.Client
	sid        string
}

// NewRequestBuilder creates a new RequestBuilder
func NewRequestBuilder(baseURL string, client *http.Client) (*RequestBuilder, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &RequestBuilder{
		baseURL:    u,
		httpClient: client,
	}, nil
}

// SetSID sets the session ID for authentication
func (rb *RequestBuilder) SetSID(sid string) {
	rb.sid = sid
}

// BuildRequest creates an HTTP request with the given parameters
func (rb *RequestBuilder) BuildRequest(ctx context.Context, method, endpoint string, queryParams map[string]string, body interface{}) (*http.Request, error) {
	// Build URL
	u := rb.baseURL.ResolveReference(&url.URL{Path: endpoint})

	// Add query parameters
	if len(queryParams) > 0 {
		q := u.Query()
		for k, v := range queryParams {
			q.Set(k, v)
		}
		// Add SID if available
		if rb.sid != "" {
			q.Set("sid", rb.sid)
		}
		u.RawQuery = q.Encode()
	}

	// Build body
	var bodyReader io.Reader
	var contentType string

	if body != nil {
		switch v := body.(type) {
		case io.Reader:
			bodyReader = v
			contentType = "application/octet-stream"
		case []byte:
			bodyReader = bytes.NewReader(v)
			contentType = "application/octet-stream"
		case *multipart.Writer:
			bodyReader = bytes.NewReader([]byte{})
			contentType = v.FormDataContentType()
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, err
			}
			bodyReader = bytes.NewReader(jsonData)
			contentType = "application/json"
		}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return req, nil
}

// BuildMultipartRequest creates a multipart request for file uploads
func (rb *RequestBuilder) BuildMultipartRequest(ctx context.Context, endpoint string, queryParams map[string]string, fields map[string]io.Reader) (*http.Request, error) {
	// Prepare body buffer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add form fields
	for key, r := range fields {
		if r == nil {
			continue
		}

		// Handle file uploads
		if strings.HasPrefix(key, "@") {
			filename := key[1:]
			part, err := writer.CreateFormFile("file", filename)
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(part, r); err != nil {
				return nil, err
			}
		} else {
			// Handle text fields
			part, err := writer.CreateFormField(key)
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(part, r); err != nil {
				return nil, err
			}
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	// Build URL
	u := rb.baseURL.ResolveReference(&url.URL{Path: endpoint})
	if len(queryParams) > 0 {
		q := u.Query()
		for k, v := range queryParams {
			q.Set(k, v)
		}
		if rb.sid != "" {
			q.Set("sid", rb.sid)
		}
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, nil
}

// JoinPath joins path elements safely
func JoinPath(base string, elems ...string) string {
	return path.Join(append([]string{base}, elems...)...)
}
