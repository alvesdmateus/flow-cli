package reliability

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HealthStatus represents the health status of a service.
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusUnknown   HealthStatus = "unknown"
)

// String returns the string representation of the health status.
func (s HealthStatus) String() string {
	return string(s)
}

// ProviderHealth contains health information for an LLM provider.
type ProviderHealth struct {
	Name         string            `json:"name"`
	Status       HealthStatus      `json:"status"`
	Latency      time.Duration     `json:"latency_ms"`
	LastCheck    time.Time         `json:"last_check"`
	LastSuccess  time.Time         `json:"last_success,omitempty"`
	LastError    string            `json:"last_error,omitempty"`
	Consecutive  int               `json:"consecutive_failures"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Available    bool              `json:"available"`
	Models       []string          `json:"models,omitempty"`
}

// HealthChecker performs health checks on LLM providers.
type HealthChecker struct {
	providers    map[string]*providerConfig
	results      map[string]*ProviderHealth
	checkInterval time.Duration
	timeout      time.Duration
	onStatusChange func(name string, oldStatus, newStatus HealthStatus)
	httpClient   *http.Client
	mu           sync.RWMutex
	stopChan     chan struct{}
	running      bool
}

type providerConfig struct {
	name        string
	endpoint    string
	healthPath  string
	modelsPath  string
	headers     map[string]string
	checkFn     func(ctx context.Context) error
}

// HealthCheckConfig contains configuration for health checking.
type HealthCheckConfig struct {
	CheckInterval time.Duration
	Timeout       time.Duration
}

// DefaultHealthCheckConfig returns sensible defaults.
func DefaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{
		CheckInterval: 30 * time.Second,
		Timeout:       5 * time.Second,
	}
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(config HealthCheckConfig) *HealthChecker {
	return &HealthChecker{
		providers:     make(map[string]*providerConfig),
		results:       make(map[string]*ProviderHealth),
		checkInterval: config.CheckInterval,
		timeout:       config.Timeout,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		stopChan: make(chan struct{}),
	}
}

// RegisterProvider registers an LLM provider for health checking.
func (hc *HealthChecker) RegisterProvider(name, endpoint string, options ...ProviderOption) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	config := &providerConfig{
		name:       name,
		endpoint:   endpoint,
		healthPath: "/health",
		headers:    make(map[string]string),
	}

	for _, opt := range options {
		opt(config)
	}

	hc.providers[name] = config
	hc.results[name] = &ProviderHealth{
		Name:      name,
		Status:    StatusUnknown,
		Available: false,
		Metadata:  make(map[string]string),
	}
}

// ProviderOption configures a provider.
type ProviderOption func(*providerConfig)

// WithHealthPath sets the health check endpoint path.
func WithHealthPath(path string) ProviderOption {
	return func(c *providerConfig) {
		c.healthPath = path
	}
}

// WithModelsPath sets the models list endpoint path.
func WithModelsPath(path string) ProviderOption {
	return func(c *providerConfig) {
		c.modelsPath = path
	}
}

// WithHeader adds a header to health check requests.
func WithHeader(key, value string) ProviderOption {
	return func(c *providerConfig) {
		c.headers[key] = value
	}
}

// WithCustomCheck sets a custom health check function.
func WithCustomCheck(fn func(ctx context.Context) error) ProviderOption {
	return func(c *providerConfig) {
		c.checkFn = fn
	}
}

// OnStatusChange sets a callback for status changes.
func (hc *HealthChecker) OnStatusChange(fn func(name string, oldStatus, newStatus HealthStatus)) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.onStatusChange = fn
}

// Start begins periodic health checking.
func (hc *HealthChecker) Start() {
	hc.mu.Lock()
	if hc.running {
		hc.mu.Unlock()
		return
	}
	hc.running = true
	hc.stopChan = make(chan struct{})
	hc.mu.Unlock()

	// Initial check
	hc.CheckAll()

	go func() {
		ticker := time.NewTicker(hc.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				hc.CheckAll()
			case <-hc.stopChan:
				return
			}
		}
	}()
}

// Stop stops periodic health checking.
func (hc *HealthChecker) Stop() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if hc.running {
		close(hc.stopChan)
		hc.running = false
	}
}

// CheckAll performs health checks on all registered providers.
func (hc *HealthChecker) CheckAll() {
	hc.mu.RLock()
	providers := make([]*providerConfig, 0, len(hc.providers))
	for _, p := range hc.providers {
		providers = append(providers, p)
	}
	hc.mu.RUnlock()

	var wg sync.WaitGroup
	for _, provider := range providers {
		wg.Add(1)
		go func(p *providerConfig) {
			defer wg.Done()
			hc.Check(p.name)
		}(provider)
	}
	wg.Wait()
}

// Check performs a health check on a specific provider.
func (hc *HealthChecker) Check(name string) *ProviderHealth {
	hc.mu.RLock()
	provider, ok := hc.providers[name]
	if !ok {
		hc.mu.RUnlock()
		return nil
	}
	hc.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	startTime := time.Now()
	var checkErr error

	if provider.checkFn != nil {
		checkErr = provider.checkFn(ctx)
	} else {
		checkErr = hc.defaultCheck(ctx, provider)
	}

	latency := time.Since(startTime)

	hc.mu.Lock()
	defer hc.mu.Unlock()

	result := hc.results[name]
	oldStatus := result.Status
	result.LastCheck = time.Now()
	result.Latency = latency

	if checkErr == nil {
		result.Status = StatusHealthy
		result.Available = true
		result.LastSuccess = time.Now()
		result.LastError = ""
		result.Consecutive = 0

		// Try to fetch models if path is configured
		if provider.modelsPath != "" {
			models, _ := hc.fetchModels(ctx, provider)
			if len(models) > 0 {
				result.Models = models
			}
		}
	} else {
		result.LastError = checkErr.Error()
		result.Consecutive++

		if result.Consecutive >= 3 {
			result.Status = StatusUnhealthy
			result.Available = false
		} else {
			result.Status = StatusDegraded
			result.Available = true
		}
	}

	// Notify status change
	if oldStatus != result.Status && hc.onStatusChange != nil {
		go hc.onStatusChange(name, oldStatus, result.Status)
	}

	return result
}

func (hc *HealthChecker) defaultCheck(ctx context.Context, provider *providerConfig) error {
	url := provider.endpoint + provider.healthPath

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range provider.headers {
		req.Header.Set(key, value)
	}

	resp, err := hc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("unhealthy status code: %d", resp.StatusCode)
	}

	return nil
}

func (hc *HealthChecker) fetchModels(ctx context.Context, provider *providerConfig) ([]string, error) {
	url := provider.endpoint + provider.modelsPath

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range provider.headers {
		req.Header.Set(key, value)
	}

	resp, err := hc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch models: %d", resp.StatusCode)
	}

	// Try to parse as Ollama format
	var ollamaResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err == nil && len(ollamaResp.Models) > 0 {
		models := make([]string, len(ollamaResp.Models))
		for i, m := range ollamaResp.Models {
			models[i] = m.Name
		}
		return models, nil
	}

	// Try OpenAI format
	var openaiResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err == nil && len(openaiResp.Data) > 0 {
		models := make([]string, len(openaiResp.Data))
		for i, m := range openaiResp.Data {
			models[i] = m.ID
		}
		return models, nil
	}

	return nil, nil
}

// GetStatus returns the health status for a provider.
func (hc *HealthChecker) GetStatus(name string) *ProviderHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if result, ok := hc.results[name]; ok {
		// Return a copy
		copy := *result
		return &copy
	}
	return nil
}

// GetAllStatus returns health status for all providers.
func (hc *HealthChecker) GetAllStatus() map[string]*ProviderHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	results := make(map[string]*ProviderHealth, len(hc.results))
	for name, result := range hc.results {
		copy := *result
		results[name] = &copy
	}
	return results
}

// GetHealthyProviders returns names of healthy providers.
func (hc *HealthChecker) GetHealthyProviders() []string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	var healthy []string
	for name, result := range hc.results {
		if result.Status == StatusHealthy {
			healthy = append(healthy, name)
		}
	}
	return healthy
}

// GetAvailableProviders returns names of available providers (healthy or degraded).
func (hc *HealthChecker) GetAvailableProviders() []string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	var available []string
	for name, result := range hc.results {
		if result.Available {
			available = append(available, name)
		}
	}
	return available
}

// IsHealthy returns whether a specific provider is healthy.
func (hc *HealthChecker) IsHealthy(name string) bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if result, ok := hc.results[name]; ok {
		return result.Status == StatusHealthy
	}
	return false
}

// IsAvailable returns whether a specific provider is available.
func (hc *HealthChecker) IsAvailable(name string) bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if result, ok := hc.results[name]; ok {
		return result.Available
	}
	return false
}

// OverallHealth returns the overall health status across all providers.
func (hc *HealthChecker) OverallHealth() HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if len(hc.results) == 0 {
		return StatusUnknown
	}

	healthyCount := 0
	availableCount := 0

	for _, result := range hc.results {
		if result.Status == StatusHealthy {
			healthyCount++
		}
		if result.Available {
			availableCount++
		}
	}

	if healthyCount == len(hc.results) {
		return StatusHealthy
	}
	if availableCount > 0 {
		return StatusDegraded
	}
	return StatusUnhealthy
}

// WaitForHealthy blocks until at least one provider is healthy or timeout.
func (hc *HealthChecker) WaitForHealthy(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		if len(hc.GetHealthyProviders()) > 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			hc.CheckAll()
		}
	}
}

// SelectBestProvider returns the healthiest provider with lowest latency.
func (hc *HealthChecker) SelectBestProvider() string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	var bestName string
	var bestLatency time.Duration

	for name, result := range hc.results {
		if !result.Available {
			continue
		}

		// Prefer healthy over degraded
		if bestName == "" {
			bestName = name
			bestLatency = result.Latency
			continue
		}

		currentBest := hc.results[bestName]
		if result.Status == StatusHealthy && currentBest.Status != StatusHealthy {
			bestName = name
			bestLatency = result.Latency
		} else if result.Status == currentBest.Status && result.Latency < bestLatency {
			bestName = name
			bestLatency = result.Latency
		}
	}

	return bestName
}
