package reliability

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// FallbackMode represents the operating mode when primary service is unavailable.
type FallbackMode int

const (
	// ModeNormal indicates normal operation with primary service.
	ModeNormal FallbackMode = iota
	// ModeDegraded indicates degraded operation with limited functionality.
	ModeDegraded
	// ModeOffline indicates offline operation with cached/local data only.
	ModeOffline
)

// String returns a string representation of the fallback mode.
func (m FallbackMode) String() string {
	switch m {
	case ModeNormal:
		return "normal"
	case ModeDegraded:
		return "degraded"
	case ModeOffline:
		return "offline"
	default:
		return "unknown"
	}
}

// FallbackHandler provides graceful degradation when services are unavailable.
type FallbackHandler struct {
	mode           FallbackMode
	primaryAvail   bool
	lastCheck      time.Time
	checkInterval  time.Duration
	onModeChange   func(from, to FallbackMode)
	fallbacks      map[string]FallbackFunc
	cache          *ResponseCache
	mu             sync.RWMutex
}

// FallbackFunc is a function that provides alternative behavior.
type FallbackFunc func(ctx context.Context, request interface{}) (interface{}, error)

// FallbackConfig contains configuration for fallback behavior.
type FallbackConfig struct {
	CheckInterval  time.Duration
	CacheEnabled   bool
	CacheTTL       time.Duration
	CacheMaxSize   int
}

// DefaultFallbackConfig returns sensible defaults.
func DefaultFallbackConfig() FallbackConfig {
	return FallbackConfig{
		CheckInterval: 30 * time.Second,
		CacheEnabled:  true,
		CacheTTL:      5 * time.Minute,
		CacheMaxSize:  100,
	}
}

// NewFallbackHandler creates a new fallback handler.
func NewFallbackHandler(config FallbackConfig) *FallbackHandler {
	fh := &FallbackHandler{
		mode:          ModeNormal,
		primaryAvail:  true,
		checkInterval: config.CheckInterval,
		fallbacks:     make(map[string]FallbackFunc),
	}

	if config.CacheEnabled {
		fh.cache = NewResponseCache(config.CacheTTL, config.CacheMaxSize)
	}

	return fh
}

// OnModeChange sets a callback for mode changes.
func (fh *FallbackHandler) OnModeChange(fn func(from, to FallbackMode)) {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	fh.onModeChange = fn
}

// RegisterFallback registers a fallback function for an operation.
func (fh *FallbackHandler) RegisterFallback(operation string, fn FallbackFunc) {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	fh.fallbacks[operation] = fn
}

// SetPrimaryAvailable updates the primary service availability.
func (fh *FallbackHandler) SetPrimaryAvailable(available bool) {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	fh.primaryAvail = available
	fh.lastCheck = time.Now()

	oldMode := fh.mode
	if available {
		fh.mode = ModeNormal
	} else if fh.cache != nil {
		fh.mode = ModeDegraded
	} else {
		fh.mode = ModeOffline
	}

	if oldMode != fh.mode && fh.onModeChange != nil {
		fh.onModeChange(oldMode, fh.mode)
	}
}

// Mode returns the current operating mode.
func (fh *FallbackHandler) Mode() FallbackMode {
	fh.mu.RLock()
	defer fh.mu.RUnlock()
	return fh.mode
}

// IsPrimaryAvailable returns whether the primary service is available.
func (fh *FallbackHandler) IsPrimaryAvailable() bool {
	fh.mu.RLock()
	defer fh.mu.RUnlock()
	return fh.primaryAvail
}

// Execute runs an operation with fallback support.
func (fh *FallbackHandler) Execute(ctx context.Context, operation string, primaryFn func() (interface{}, error), cacheKey string) (interface{}, error) {
	fh.mu.RLock()
	mode := fh.mode
	fallbackFn := fh.fallbacks[operation]
	fh.mu.RUnlock()

	// Try primary first if available
	if mode == ModeNormal {
		result, err := primaryFn()
		if err == nil {
			// Cache successful result
			if fh.cache != nil && cacheKey != "" {
				fh.cache.Set(cacheKey, result)
			}
			return result, nil
		}

		// Primary failed, update availability
		fh.SetPrimaryAvailable(false)
	}

	// Try cache in degraded mode
	if fh.cache != nil && cacheKey != "" {
		if cached, ok := fh.cache.Get(cacheKey); ok {
			return cached, nil
		}
	}

	// Try fallback function
	if fallbackFn != nil {
		return fallbackFn(ctx, nil)
	}

	return nil, fmt.Errorf("service unavailable and no fallback available for operation: %s", operation)
}

// ResponseCache provides caching for responses.
type ResponseCache struct {
	entries  map[string]*cacheEntry
	ttl      time.Duration
	maxSize  int
	mu       sync.RWMutex
}

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

// NewResponseCache creates a new response cache.
func NewResponseCache(ttl time.Duration, maxSize int) *ResponseCache {
	return &ResponseCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Set adds or updates a cache entry.
func (rc *ResponseCache) Set(key string, value interface{}) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Evict if at capacity
	if len(rc.entries) >= rc.maxSize {
		rc.evictOldest()
	}

	rc.entries[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(rc.ttl),
	}
}

// Get retrieves a cache entry if it exists and is not expired.
func (rc *ResponseCache) Get(key string) (interface{}, bool) {
	rc.mu.RLock()
	entry, ok := rc.entries[key]
	rc.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		rc.mu.Lock()
		delete(rc.entries, key)
		rc.mu.Unlock()
		return nil, false
	}

	return entry.value, true
}

// Delete removes a cache entry.
func (rc *ResponseCache) Delete(key string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	delete(rc.entries, key)
}

// Clear removes all cache entries.
func (rc *ResponseCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.entries = make(map[string]*cacheEntry)
}

func (rc *ResponseCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range rc.entries {
		if oldestKey == "" || entry.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiresAt
		}
	}

	if oldestKey != "" {
		delete(rc.entries, oldestKey)
	}
}

// Size returns the number of entries in the cache.
func (rc *ResponseCache) Size() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return len(rc.entries)
}

// ServiceDegrader manages graceful degradation for multiple services.
type ServiceDegrader struct {
	services map[string]*serviceState
	mu       sync.RWMutex
}

type serviceState struct {
	name        string
	available   bool
	lastCheck   time.Time
	failCount   int
	features    map[string]bool
}

// NewServiceDegrader creates a new service degrader.
func NewServiceDegrader() *ServiceDegrader {
	return &ServiceDegrader{
		services: make(map[string]*serviceState),
	}
}

// RegisterService registers a service with its features.
func (sd *ServiceDegrader) RegisterService(name string, features []string) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	state := &serviceState{
		name:      name,
		available: true,
		features:  make(map[string]bool),
	}
	for _, f := range features {
		state.features[f] = true
	}
	sd.services[name] = state
}

// MarkServiceDown marks a service as unavailable.
func (sd *ServiceDegrader) MarkServiceDown(name string) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	if state, ok := sd.services[name]; ok {
		state.available = false
		state.failCount++
		state.lastCheck = time.Now()
	}
}

// MarkServiceUp marks a service as available.
func (sd *ServiceDegrader) MarkServiceUp(name string) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	if state, ok := sd.services[name]; ok {
		state.available = true
		state.failCount = 0
		state.lastCheck = time.Now()
	}
}

// IsFeatureAvailable checks if a feature is available.
func (sd *ServiceDegrader) IsFeatureAvailable(feature string) bool {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	for _, state := range sd.services {
		if state.available && state.features[feature] {
			return true
		}
	}
	return false
}

// GetAvailableFeatures returns all available features.
func (sd *ServiceDegrader) GetAvailableFeatures() []string {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	features := make(map[string]bool)
	for _, state := range sd.services {
		if state.available {
			for f := range state.features {
				features[f] = true
			}
		}
	}

	result := make([]string, 0, len(features))
	for f := range features {
		result = append(result, f)
	}
	return result
}

// GetDegradedFeatures returns features that are unavailable.
func (sd *ServiceDegrader) GetDegradedFeatures() []string {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	allFeatures := make(map[string]bool)
	availableFeatures := make(map[string]bool)

	for _, state := range sd.services {
		for f := range state.features {
			allFeatures[f] = true
			if state.available {
				availableFeatures[f] = true
			}
		}
	}

	var degraded []string
	for f := range allFeatures {
		if !availableFeatures[f] {
			degraded = append(degraded, f)
		}
	}
	return degraded
}

// OfflineCapability represents functionality available offline.
type OfflineCapability struct {
	Name        string
	Description string
	Handler     func(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

// OfflineManager manages offline capabilities.
type OfflineManager struct {
	capabilities map[string]*OfflineCapability
	mu           sync.RWMutex
}

// NewOfflineManager creates a new offline manager.
func NewOfflineManager() *OfflineManager {
	return &OfflineManager{
		capabilities: make(map[string]*OfflineCapability),
	}
}

// Register adds an offline capability.
func (om *OfflineManager) Register(cap *OfflineCapability) {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.capabilities[cap.Name] = cap
}

// Execute runs an offline capability.
func (om *OfflineManager) Execute(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	om.mu.RLock()
	cap, ok := om.capabilities[name]
	om.mu.RUnlock()

	if !ok {
		return nil, errors.New("offline capability not available: " + name)
	}

	return cap.Handler(ctx, args)
}

// List returns all available offline capabilities.
func (om *OfflineManager) List() []string {
	om.mu.RLock()
	defer om.mu.RUnlock()

	names := make([]string, 0, len(om.capabilities))
	for name := range om.capabilities {
		names = append(names, name)
	}
	return names
}

// Has checks if an offline capability exists.
func (om *OfflineManager) Has(name string) bool {
	om.mu.RLock()
	defer om.mu.RUnlock()
	_, ok := om.capabilities[name]
	return ok
}
