package external

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Docker client tests

func TestNewDockerClient(t *testing.T) {
	dc := NewDockerClient(nil)
	if dc == nil {
		t.Fatal("NewDockerClient returned nil")
	}

	if dc.timeout != 5*time.Minute {
		t.Errorf("Expected default timeout of 5 minutes, got %v", dc.timeout)
	}
}

func TestNewDockerClient_WithConfig(t *testing.T) {
	config := &DockerConfig{
		DockerPath: "/custom/docker",
		Timeout:    10 * time.Minute,
	}

	dc := NewDockerClient(config)
	if dc == nil {
		t.Fatal("NewDockerClient returned nil")
	}

	if dc.dockerPath != "/custom/docker" {
		t.Errorf("Expected docker path '/custom/docker', got '%s'", dc.dockerPath)
	}

	if dc.timeout != 10*time.Minute {
		t.Errorf("Expected timeout of 10 minutes, got %v", dc.timeout)
	}
}

func TestBuildOptions(t *testing.T) {
	opts := BuildOptions{
		Dockerfile: "Dockerfile.prod",
		Context:    ".",
		Tags:       []string{"myapp:latest", "myapp:v1.0"},
		BuildArgs:  map[string]string{"VERSION": "1.0"},
		NoCache:    true,
		Pull:       true,
		Target:     "production",
		Platform:   "linux/amd64",
	}

	if len(opts.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(opts.Tags))
	}

	if opts.BuildArgs["VERSION"] != "1.0" {
		t.Error("BuildArgs not set correctly")
	}
}

func TestRunOptions(t *testing.T) {
	opts := RunOptions{
		Image:   "nginx:latest",
		Name:    "my-nginx",
		Detach:  true,
		Remove:  true,
		Ports:   map[string]string{"8080": "80"},
		Volumes: map[string]string{"/data": "/var/data"},
		Env:     map[string]string{"ENV": "production"},
		Network: "my-network",
	}

	if opts.Image != "nginx:latest" {
		t.Error("Image not set correctly")
	}

	if opts.Ports["8080"] != "80" {
		t.Error("Port mapping not set correctly")
	}
}

// HTTP client tests

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient(nil)
	if client == nil {
		t.Fatal("NewHTTPClient returned nil")
	}

	if client.timeout != 30*time.Second {
		t.Errorf("Expected default timeout of 30s, got %v", client.timeout)
	}
}

func TestNewHTTPClient_WithConfig(t *testing.T) {
	config := &HTTPConfig{
		BaseURL: "https://api.example.com",
		Timeout: 60 * time.Second,
		Headers: map[string]string{
			"X-Custom-Header": "value",
		},
	}

	client := NewHTTPClient(config)
	if client == nil {
		t.Fatal("NewHTTPClient returned nil")
	}

	if client.baseURL != "https://api.example.com" {
		t.Errorf("Expected baseURL 'https://api.example.com', got '%s'", client.baseURL)
	}

	if client.headers["X-Custom-Header"] != "value" {
		t.Error("Custom header not set")
	}
}

func TestHTTPClient_SetAuth(t *testing.T) {
	client := NewHTTPClient(nil)

	client.SetAuth("bearer", "my-token")
	if client.headers["Authorization"] != "Bearer my-token" {
		t.Errorf("Expected 'Bearer my-token', got '%s'", client.headers["Authorization"])
	}

	client.SetAuth("basic", "base64encoded")
	if client.headers["Authorization"] != "Basic base64encoded" {
		t.Errorf("Expected 'Basic base64encoded', got '%s'", client.headers["Authorization"])
	}

	client.SetAuth("apikey", "my-api-key")
	if client.headers["X-API-Key"] != "my-api-key" {
		t.Errorf("Expected 'my-api-key', got '%s'", client.headers["X-API-Key"])
	}
}

func TestHTTPClient_GET(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("key") != "value" {
			t.Error("Query param not received")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	client := NewHTTPClient(&HTTPConfig{BaseURL: server.URL})

	resp, err := client.GET(context.Background(), "/test", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if !resp.IsSuccess() {
		t.Error("Expected IsSuccess() to return true")
	}
}

func TestHTTPClient_POST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Content-Type header not set")
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 123}`))
	}))
	defer server.Close()

	client := NewHTTPClient(&HTTPConfig{BaseURL: server.URL})

	body := map[string]string{"name": "test"}
	resp, err := client.POST(context.Background(), "/create", body)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}

	if resp.StatusCode != 201 {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
}

func TestHTTPClient_PUT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(&HTTPConfig{BaseURL: server.URL})

	_, err := client.PUT(context.Background(), "/update", nil)
	if err != nil {
		t.Fatalf("PUT failed: %v", err)
	}
}

func TestHTTPClient_DELETE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewHTTPClient(&HTTPConfig{BaseURL: server.URL})

	resp, err := client.DELETE(context.Background(), "/item/123")
	if err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}

	if resp.StatusCode != 204 {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}
}

func TestHTTPResponse_StatusMethods(t *testing.T) {
	tests := []struct {
		code         int
		isSuccess    bool
		isRedirect   bool
		isClientErr  bool
		isServerErr  bool
	}{
		{200, true, false, false, false},
		{201, true, false, false, false},
		{301, false, true, false, false},
		{302, false, true, false, false},
		{400, false, false, true, false},
		{404, false, false, true, false},
		{500, false, false, false, true},
		{503, false, false, false, true},
	}

	for _, tc := range tests {
		resp := &HTTPResponse{StatusCode: tc.code}

		if resp.IsSuccess() != tc.isSuccess {
			t.Errorf("Status %d: IsSuccess expected %v", tc.code, tc.isSuccess)
		}
		if resp.IsRedirect() != tc.isRedirect {
			t.Errorf("Status %d: IsRedirect expected %v", tc.code, tc.isRedirect)
		}
		if resp.IsClientError() != tc.isClientErr {
			t.Errorf("Status %d: IsClientError expected %v", tc.code, tc.isClientErr)
		}
		if resp.IsServerError() != tc.isServerErr {
			t.Errorf("Status %d: IsServerError expected %v", tc.code, tc.isServerErr)
		}
	}
}

func TestHTTPResponse_JSON(t *testing.T) {
	resp := &HTTPResponse{
		Body: `{"name": "test", "count": 42}`,
	}

	var result map[string]interface{}
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("JSON parsing failed: %v", err)
	}

	if result["name"] != "test" {
		t.Errorf("Expected name 'test', got '%v'", result["name"])
	}

	if result["count"].(float64) != 42 {
		t.Errorf("Expected count 42, got '%v'", result["count"])
	}
}

// Package manager tests

func TestNewPackageClient(t *testing.T) {
	managers := []PackageManager{
		PackageManagerNPM,
		PackageManagerYarn,
		PackageManagerPnpm,
		PackageManagerPip,
		PackageManagerCargo,
		PackageManagerGo,
	}

	for _, pm := range managers {
		client := NewPackageClient(pm)
		if client == nil {
			t.Errorf("NewPackageClient(%s) returned nil", pm)
		}
		if client.manager != pm {
			t.Errorf("Expected manager %s, got %s", pm, client.manager)
		}
	}
}

func TestPackageInfo(t *testing.T) {
	info := PackageInfo{
		Name:        "express",
		Version:     "4.18.0",
		Description: "Fast web framework",
		Author:      "TJ Holowaychuk",
		License:     "MIT",
		Homepage:    "https://expressjs.com",
		Keywords:    []string{"web", "framework"},
		Dependencies: map[string]string{
			"accepts": "~1.3.8",
		},
	}

	if info.Name != "express" {
		t.Error("Name not set correctly")
	}

	if len(info.Dependencies) != 1 {
		t.Error("Dependencies not set correctly")
	}
}

func TestInstalledPackage(t *testing.T) {
	pkg := InstalledPackage{
		Name:     "lodash",
		Version:  "4.17.0",
		Wanted:   "4.17.21",
		Latest:   "4.17.21",
		Outdated: true,
	}

	if !pkg.Outdated {
		t.Error("Outdated should be true")
	}

	if pkg.Version == pkg.Latest {
		t.Error("Version should differ from Latest for outdated package")
	}
}

// CI/CD client tests

func TestNewCICDClient(t *testing.T) {
	config := CICDConfig{
		Provider: CICDGitHubActions,
		Token:    "test-token",
		Owner:    "owner",
		Repo:     "repo",
	}

	client, err := NewCICDClient(config)
	if err != nil {
		t.Fatalf("NewCICDClient failed: %v", err)
	}

	if client.provider != CICDGitHubActions {
		t.Errorf("Expected provider GitHub, got %s", client.provider)
	}

	if client.owner != "owner" {
		t.Errorf("Expected owner 'owner', got '%s'", client.owner)
	}

	if client.repo != "repo" {
		t.Errorf("Expected repo 'repo', got '%s'", client.repo)
	}
}

func TestNewCICDClient_UnsupportedProvider(t *testing.T) {
	config := CICDConfig{
		Provider: "unsupported",
	}

	_, err := NewCICDClient(config)
	if err == nil {
		t.Error("Expected error for unsupported provider")
	}
}

func TestWorkflowRunStatus(t *testing.T) {
	tests := []struct {
		status     string
		conclusion string
		isRunning  bool
		isSuccess  bool
		isFailed   bool
	}{
		{"in_progress", "", true, false, false},
		{"queued", "", true, false, false},
		{"completed", "success", false, true, false},
		{"completed", "failure", false, false, true},
		{"completed", "cancelled", false, false, false},
	}

	for _, tc := range tests {
		run := WorkflowRun{
			Status:     tc.status,
			Conclusion: tc.conclusion,
		}
		status := run.GetStatus()

		if status.IsRunning != tc.isRunning {
			t.Errorf("Status %s/%s: IsRunning expected %v", tc.status, tc.conclusion, tc.isRunning)
		}
		if status.IsSuccess != tc.isSuccess {
			t.Errorf("Status %s/%s: IsSuccess expected %v", tc.status, tc.conclusion, tc.isSuccess)
		}
		if status.IsFailed != tc.isFailed {
			t.Errorf("Status %s/%s: IsFailed expected %v", tc.status, tc.conclusion, tc.isFailed)
		}
	}
}

// Database helper tests

func TestIsReadOnlyQuery(t *testing.T) {
	readOnlyQueries := []string{
		"SELECT * FROM users",
		"select id from users",
		"SHOW TABLES",
		"DESCRIBE users",
		"EXPLAIN SELECT * FROM users",
		"PRAGMA table_info(users)",
	}

	for _, query := range readOnlyQueries {
		if !isReadOnlyQuery(query) {
			t.Errorf("Expected '%s' to be read-only", query)
		}
	}

	writeQueries := []string{
		"INSERT INTO users VALUES (1)",
		"UPDATE users SET name = 'test'",
		"DELETE FROM users",
		"DROP TABLE users",
		"CREATE TABLE test (id INT)",
	}

	for _, query := range writeQueries {
		if isReadOnlyQuery(query) {
			t.Errorf("Expected '%s' to NOT be read-only", query)
		}
	}
}

func TestConvertToJSONSafe(t *testing.T) {
	// Test nil
	result := convertToJSONSafe(nil)
	if result != nil {
		t.Error("Expected nil for nil input")
	}

	// Test []byte
	result = convertToJSONSafe([]byte("hello"))
	if result != "hello" {
		t.Errorf("Expected 'hello', got '%v'", result)
	}

	// Test time.Time
	now := time.Now()
	result = convertToJSONSafe(now)
	if _, ok := result.(string); !ok {
		t.Error("Expected time to be converted to string")
	}

	// Test other types (pass through)
	result = convertToJSONSafe(42)
	if result != 42 {
		t.Errorf("Expected 42, got '%v'", result)
	}
}

func TestQueryResult_ToMaps(t *testing.T) {
	qr := &QueryResult{
		Columns: []string{"id", "name", "age"},
		Rows: [][]interface{}{
			{1, "Alice", 30},
			{2, "Bob", 25},
		},
	}

	maps := qr.ToMaps()
	if len(maps) != 2 {
		t.Errorf("Expected 2 maps, got %d", len(maps))
	}

	if maps[0]["name"] != "Alice" {
		t.Errorf("Expected name 'Alice', got '%v'", maps[0]["name"])
	}

	if maps[1]["age"] != 25 {
		t.Errorf("Expected age 25, got '%v'", maps[1]["age"])
	}
}

func TestQueryResult_ToJSON(t *testing.T) {
	qr := &QueryResult{
		Columns:      []string{"id"},
		Rows:         [][]interface{}{{1}},
		RowsAffected: 1,
	}

	jsonStr, err := qr.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if jsonStr == "" {
		t.Error("Expected non-empty JSON string")
	}
}

// API client tests

func TestNewAPIClient(t *testing.T) {
	client := NewAPIClient("https://api.example.com", "bearer", "token")
	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}

	if client.baseURL != "https://api.example.com" {
		t.Errorf("Expected baseURL 'https://api.example.com', got '%s'", client.baseURL)
	}
}

func TestAPIClient_GetJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": "test"}`))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "", "")

	var result map[string]string
	err := client.GetJSON(context.Background(), "/data", nil, &result)
	if err != nil {
		t.Fatalf("GetJSON failed: %v", err)
	}

	if result["data"] != "test" {
		t.Errorf("Expected data 'test', got '%s'", result["data"])
	}
}

func TestAPIClient_PostJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 1}`))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "", "")

	var result map[string]int
	err := client.PostJSON(context.Background(), "/create", map[string]string{"name": "test"}, &result)
	if err != nil {
		t.Fatalf("PostJSON failed: %v", err)
	}

	if result["id"] != 1 {
		t.Errorf("Expected id 1, got %d", result["id"])
	}
}

// Helper function tests

func TestGetString(t *testing.T) {
	m := map[string]interface{}{
		"name":   "test",
		"number": 42,
		"nested": map[string]string{"key": "value"},
	}

	// Test existing string key
	result := getString(m, "name")
	if result != "test" {
		t.Errorf("Expected 'test', got '%s'", result)
	}

	// Test non-string value
	result = getString(m, "number")
	if result != "" {
		t.Errorf("Expected empty string for non-string value, got '%s'", result)
	}

	// Test missing key
	result = getString(m, "missing")
	if result != "" {
		t.Errorf("Expected empty string for missing key, got '%s'", result)
	}
}
