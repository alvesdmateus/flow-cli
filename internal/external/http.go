package external

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HTTPClient provides HTTP request capabilities.
type HTTPClient struct {
	client  *http.Client
	baseURL string
	headers map[string]string
	timeout time.Duration
}

// HTTPConfig configures the HTTP client.
type HTTPConfig struct {
	BaseURL           string
	Timeout           time.Duration
	Headers           map[string]string
	InsecureSkipVerify bool
	FollowRedirects   bool
	MaxRedirects      int
}

// HTTPRequest represents an HTTP request.
type HTTPRequest struct {
	Method      string
	URL         string
	Headers     map[string]string
	QueryParams map[string]string
	Body        interface{} // Can be string, []byte, map, or struct
	FormData    map[string]string
	Files       map[string]string // field name -> file path
}

// HTTPResponse represents an HTTP response.
type HTTPResponse struct {
	StatusCode    int               `json:"status_code"`
	Status        string            `json:"status"`
	Headers       map[string]string `json:"headers"`
	Body          string            `json:"body"`
	ContentType   string            `json:"content_type"`
	ContentLength int64             `json:"content_length"`
	Duration      time.Duration     `json:"duration_ms"`
}

// NewHTTPClient creates a new HTTP client.
func NewHTTPClient(config *HTTPConfig) *HTTPClient {
	timeout := 30 * time.Second
	if config != nil && config.Timeout > 0 {
		timeout = config.Timeout
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config != nil && config.InsecureSkipVerify,
		},
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	// Configure redirects
	if config != nil && !config.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else if config != nil && config.MaxRedirects > 0 {
		maxRedirects := config.MaxRedirects
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil
		}
	}

	hc := &HTTPClient{
		client:  client,
		timeout: timeout,
		headers: make(map[string]string),
	}

	if config != nil {
		hc.baseURL = config.BaseURL
		for k, v := range config.Headers {
			hc.headers[k] = v
		}
	}

	return hc
}

// SetHeader sets a default header for all requests.
func (hc *HTTPClient) SetHeader(key, value string) {
	hc.headers[key] = value
}

// SetAuth sets authentication header.
func (hc *HTTPClient) SetAuth(authType, credentials string) {
	switch strings.ToLower(authType) {
	case "bearer":
		hc.headers["Authorization"] = "Bearer " + credentials
	case "basic":
		hc.headers["Authorization"] = "Basic " + credentials
	case "apikey":
		hc.headers["X-API-Key"] = credentials
	default:
		hc.headers["Authorization"] = credentials
	}
}

// Request executes an HTTP request.
func (hc *HTTPClient) Request(ctx context.Context, req HTTPRequest) (*HTTPResponse, error) {
	start := time.Now()

	// Build URL
	requestURL := req.URL
	if hc.baseURL != "" && !strings.HasPrefix(req.URL, "http") {
		requestURL = strings.TrimSuffix(hc.baseURL, "/") + "/" + strings.TrimPrefix(req.URL, "/")
	}

	// Add query parameters
	if len(req.QueryParams) > 0 {
		u, err := url.Parse(requestURL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}
		q := u.Query()
		for k, v := range req.QueryParams {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		requestURL = u.String()
	}

	// Prepare body
	var bodyReader io.Reader
	var contentType string

	if len(req.Files) > 0 {
		// Multipart form with files
		body, ct, err := hc.createMultipartForm(req.FormData, req.Files)
		if err != nil {
			return nil, err
		}
		bodyReader = body
		contentType = ct
	} else if len(req.FormData) > 0 {
		// URL-encoded form
		form := url.Values{}
		for k, v := range req.FormData {
			form.Set(k, v)
		}
		bodyReader = strings.NewReader(form.Encode())
		contentType = "application/x-www-form-urlencoded"
	} else if req.Body != nil {
		// JSON or raw body
		switch v := req.Body.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		default:
			jsonBody, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonBody)
			contentType = "application/json"
		}
	}

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, requestURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	for k, v := range hc.headers {
		httpReq.Header.Set(k, v)
	}

	// Set request-specific headers
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Set content type if not already set
	if contentType != "" && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	// Execute request
	resp, err := hc.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Build response
	response := &HTTPResponse{
		StatusCode:    resp.StatusCode,
		Status:        resp.Status,
		Headers:       make(map[string]string),
		Body:          string(body),
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: resp.ContentLength,
		Duration:      time.Since(start),
	}

	for k, v := range resp.Header {
		if len(v) > 0 {
			response.Headers[k] = v[0]
		}
	}

	return response, nil
}

// GET performs a GET request.
func (hc *HTTPClient) GET(ctx context.Context, url string, params map[string]string) (*HTTPResponse, error) {
	return hc.Request(ctx, HTTPRequest{
		Method:      "GET",
		URL:         url,
		QueryParams: params,
	})
}

// POST performs a POST request.
func (hc *HTTPClient) POST(ctx context.Context, url string, body interface{}) (*HTTPResponse, error) {
	return hc.Request(ctx, HTTPRequest{
		Method: "POST",
		URL:    url,
		Body:   body,
	})
}

// PUT performs a PUT request.
func (hc *HTTPClient) PUT(ctx context.Context, url string, body interface{}) (*HTTPResponse, error) {
	return hc.Request(ctx, HTTPRequest{
		Method: "PUT",
		URL:    url,
		Body:   body,
	})
}

// PATCH performs a PATCH request.
func (hc *HTTPClient) PATCH(ctx context.Context, url string, body interface{}) (*HTTPResponse, error) {
	return hc.Request(ctx, HTTPRequest{
		Method: "PATCH",
		URL:    url,
		Body:   body,
	})
}

// DELETE performs a DELETE request.
func (hc *HTTPClient) DELETE(ctx context.Context, url string) (*HTTPResponse, error) {
	return hc.Request(ctx, HTTPRequest{
		Method: "DELETE",
		URL:    url,
	})
}

// PostForm performs a POST with form data.
func (hc *HTTPClient) PostForm(ctx context.Context, url string, formData map[string]string) (*HTTPResponse, error) {
	return hc.Request(ctx, HTTPRequest{
		Method:   "POST",
		URL:      url,
		FormData: formData,
	})
}

// UploadFile uploads a file.
func (hc *HTTPClient) UploadFile(ctx context.Context, url, fieldName, filePath string, extraFields map[string]string) (*HTTPResponse, error) {
	return hc.Request(ctx, HTTPRequest{
		Method:   "POST",
		URL:      url,
		FormData: extraFields,
		Files:    map[string]string{fieldName: filePath},
	})
}

// Download downloads a file.
func (hc *HTTPClient) Download(ctx context.Context, url, destPath string) error {
	resp, err := hc.Request(ctx, HTTPRequest{
		Method: "GET",
		URL:    url,
	})
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	// Create directory if needed
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(destPath, []byte(resp.Body), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// DownloadStream downloads a large file using streaming.
func (hc *HTTPClient) DownloadStream(ctx context.Context, requestURL, destPath string) error {
	// Build full URL
	fullURL := requestURL
	if hc.baseURL != "" && !strings.HasPrefix(requestURL, "http") {
		fullURL = strings.TrimSuffix(hc.baseURL, "/") + "/" + strings.TrimPrefix(requestURL, "/")
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for k, v := range hc.headers {
		req.Header.Set(k, v)
	}

	// Execute request
	resp, err := hc.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	// Create directory if needed
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Stream to file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (hc *HTTPClient) createMultipartForm(fields map[string]string, files map[string]string) (io.Reader, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add fields
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, "", fmt.Errorf("failed to write field: %w", err)
		}
	}

	// Add files
	for fieldName, filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
		if err != nil {
			return nil, "", fmt.Errorf("failed to create form file: %w", err)
		}

		if _, err := io.Copy(part, file); err != nil {
			return nil, "", fmt.Errorf("failed to copy file: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close writer: %w", err)
	}

	return &buf, writer.FormDataContentType(), nil
}

// JSON parses response body as JSON.
func (hr *HTTPResponse) JSON(v interface{}) error {
	return json.Unmarshal([]byte(hr.Body), v)
}

// IsSuccess returns true if status code is 2xx.
func (hr *HTTPResponse) IsSuccess() bool {
	return hr.StatusCode >= 200 && hr.StatusCode < 300
}

// IsRedirect returns true if status code is 3xx.
func (hr *HTTPResponse) IsRedirect() bool {
	return hr.StatusCode >= 300 && hr.StatusCode < 400
}

// IsClientError returns true if status code is 4xx.
func (hr *HTTPResponse) IsClientError() bool {
	return hr.StatusCode >= 400 && hr.StatusCode < 500
}

// IsServerError returns true if status code is 5xx.
func (hr *HTTPResponse) IsServerError() bool {
	return hr.StatusCode >= 500
}

// APIClient is a higher-level API client with common patterns.
type APIClient struct {
	*HTTPClient
}

// NewAPIClient creates a new API client.
func NewAPIClient(baseURL string, authType, credentials string) *APIClient {
	client := NewHTTPClient(&HTTPConfig{
		BaseURL: baseURL,
		Headers: map[string]string{
			"Accept":       "application/json",
			"Content-Type": "application/json",
		},
	})

	if credentials != "" {
		client.SetAuth(authType, credentials)
	}

	return &APIClient{HTTPClient: client}
}

// GetJSON performs GET and parses JSON response.
func (ac *APIClient) GetJSON(ctx context.Context, path string, params map[string]string, result interface{}) error {
	resp, err := ac.GET(ctx, path, params)
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("API error: %s - %s", resp.Status, resp.Body)
	}

	return resp.JSON(result)
}

// PostJSON performs POST with JSON body and parses JSON response.
func (ac *APIClient) PostJSON(ctx context.Context, path string, body, result interface{}) error {
	resp, err := ac.POST(ctx, path, body)
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("API error: %s - %s", resp.Status, resp.Body)
	}

	if result != nil {
		return resp.JSON(result)
	}
	return nil
}

// PutJSON performs PUT with JSON body and parses JSON response.
func (ac *APIClient) PutJSON(ctx context.Context, path string, body, result interface{}) error {
	resp, err := ac.PUT(ctx, path, body)
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("API error: %s - %s", resp.Status, resp.Body)
	}

	if result != nil {
		return resp.JSON(result)
	}
	return nil
}

// DeleteJSON performs DELETE and parses JSON response.
func (ac *APIClient) DeleteJSON(ctx context.Context, path string, result interface{}) error {
	resp, err := ac.DELETE(ctx, path)
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("API error: %s - %s", resp.Status, resp.Body)
	}

	if result != nil {
		return resp.JSON(result)
	}
	return nil
}
