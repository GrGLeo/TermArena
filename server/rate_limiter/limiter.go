// Package ratelimiter provides a thread-safe rate limiting system using token bucket algorithm.
// It supports both IP-based and user-based rate limiting with lazy initialization
// and automatic cleanup of unused limiters.
package ratelimiter

import (
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// RateLimitConfig holds the configuration for all rate limiting buckets.
// Each field corresponds to a different type of request with its own capacity and refill rate.
type RateLimitConfig struct {
	RegisterRequest       BucketConfig `yaml:"register_request"`
	LoginChallengeRequest BucketConfig `yaml:"login_request_challenge"`
	FindRoom              BucketConfig `yaml:"find-room"`
	MessageRequest        BucketConfig `yaml:"message_request"`
}

// BucketConfig defines the capacity and refill rate for a single token bucket.
type BucketConfig struct {
	Capacity int `yaml:"capacity"`
	Refill   int `yaml:"refill"`
}

type RateLimiter struct {
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
}

// NewRateLimiter creates a new rate limiter with token buckets for different request types.
// If isIPBound is true, creates buckets for IP-bound requests (register, login).
// If isIPBound is false, creates buckets for user-bound requests (find-room, message).
func NewRateLimiter(config *RateLimitConfig, isIPBound bool) *RateLimiter {
	var buckets map[string]*TokenBucket
	if isIPBound {
		registerBucket := NewTokenBucket(int64(config.RegisterRequest.Capacity), int64(config.RegisterRequest.Refill))
		loginBucket := NewTokenBucket(int64(config.LoginChallengeRequest.Capacity), int64(config.LoginChallengeRequest.Refill))
		buckets = map[string]*TokenBucket{
			"register-request":        registerBucket,
			"login-request-challenge": loginBucket,
		}
	} else {
		findBucket := NewTokenBucket(int64(config.FindRoom.Capacity), int64(config.FindRoom.Refill))
		messageRequestBucket := NewTokenBucket(int64(config.MessageRequest.Capacity), int64(config.MessageRequest.Refill))
		buckets = map[string]*TokenBucket{
			"find-room":       findBucket,
			"message_request": messageRequestBucket,
		}
	}

	return &RateLimiter{
		buckets: buckets,
	}
}

type IPRateLimiter struct {
	limiters   map[string]*RateLimiter // ip -> RateLimiter
	lastAccess map[string]time.Time
	config     *RateLimitConfig
	mu         sync.RWMutex
}

// NewIPRateLimiter creates a new IP-based rate limiter that manages rate limits per IP address.
// It uses lazy initialization to create rate limiters only when first accessed.
// Automatically cleans up unused limiters after 24 hours of inactivity.
func NewIPRateLimiter(config *RateLimitConfig) *IPRateLimiter {
	irl := &IPRateLimiter{
		limiters:   make(map[string]*RateLimiter),
		lastAccess: make(map[string]time.Time),
		config:     config,
	}
	go irl.cleanupRoutine()
	return irl
}

// GetBucket returns the token bucket for the specified IP address and request type.
// Creates a new rate limiter for the IP on first access (lazy initialization).
// Supported request types: "register-request", "login-request-challenge".
// Returns an error for invalid request types or if limiter creation fails.
// Thread-safe for concurrent access.
func (irl *IPRateLimiter) GetBucket(ip, requestType string) (*TokenBucket, error) {
	irl.mu.RLock()
	limiter, exists := irl.limiters[ip]
	irl.mu.RUnlock()
	// Lazy initialization
	if !exists {
		irl.mu.Lock()
		if limiter, exists := irl.limiters[ip]; !exists {
			limiter = NewRateLimiter(irl.config, true)
			irl.limiters[ip] = limiter
		}
		irl.mu.Unlock()
	}

	// Re-fetch limiter after potential lazy initialization
	irl.mu.RLock()
	limiter, exists = irl.limiters[ip]
	irl.mu.RUnlock()

	if !exists || limiter == nil {
		return nil, fmt.Errorf("failed to create limiter for IP: %s", ip)
	}

	irl.mu.Lock()
	irl.lastAccess[ip] = time.Now()
	irl.mu.Unlock()

	bucket, exists := limiter.buckets[requestType]
	if !exists {
		return nil, fmt.Errorf("invalid request type for IP limiter: %s", requestType)
	}
	return bucket, nil
}

func (irl *IPRateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		irl.mu.Lock()
		now := time.Now()
		for ip, lastAccess := range irl.lastAccess {
			if now.Sub(lastAccess) > 24*time.Hour {
				delete(irl.limiters, ip)
				delete(irl.lastAccess, ip)
			}
		}
		irl.mu.Unlock()
	}
}

// GetLimiterCount returns the current number of active IP limiters.
// This is primarily used for testing and monitoring purposes.
func (irl *IPRateLimiter) GetLimiterCount() int {
	irl.mu.RLock()
	defer irl.mu.RUnlock()
	return len(irl.limiters)
}

type UserRateLimiter struct {
	limiters   map[string]*RateLimiter // user -> RateLimiter
	lastAccess map[string]time.Time
	config     *RateLimitConfig
	mu         sync.RWMutex
}

// NewUserRateLimiter creates a new user-based rate limiter that manages rate limits per user.
// It uses lazy initialization to create rate limiters only when first accessed.
// Automatically cleans up unused limiters after 24 hours of inactivity.
func NewUserRateLimiter(config *RateLimitConfig) *UserRateLimiter {
	url := &UserRateLimiter{
		limiters:   make(map[string]*RateLimiter),
		lastAccess: make(map[string]time.Time),
		config:     config,
	}
	go url.cleanupRoutine()
	return url
}

// GetBucket returns the token bucket for the specified user and request type.
// Creates a new rate limiter for the user on first access (lazy initialization).
// Supported request types: "find-room", "message_request".
// Returns an error for invalid request types or if limiter creation fails.
// Thread-safe for concurrent access.
func (url *UserRateLimiter) GetBucket(user, requestType string) (*TokenBucket, error) {
	url.mu.RLock()
	limiter, exists := url.limiters[user]
	url.mu.RUnlock()
	// Lazy initialization
	if !exists {
		url.mu.Lock()
		if limiter, exists := url.limiters[user]; !exists {
			limiter = NewRateLimiter(url.config, false)
			url.limiters[user] = limiter
		}
		url.mu.Unlock()
	}

	// Re-fetch limiter after potential lazy initialization
	url.mu.RLock()
	limiter, exists = url.limiters[user]
	url.mu.RUnlock()

	if !exists || limiter == nil {
		return nil, fmt.Errorf("failed to create limiter for user: %s", user)
	}

	url.mu.Lock()
	url.lastAccess[user] = time.Now()
	url.mu.Unlock()

	bucket, exists := limiter.buckets[requestType]
	if !exists {
		return nil, fmt.Errorf("invalid request type for User limiter: %s", requestType)
	}
	return bucket, nil
}

func (url *UserRateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		url.mu.Lock()
		now := time.Now()
		for user, lastAccess := range url.lastAccess {
			if now.Sub(lastAccess) > 24*time.Hour {
				delete(url.limiters, user)
				delete(url.lastAccess, user)
			}
		}
		url.mu.Unlock()
	}
}

type GlobalRateLimiter struct {
	ipLimiter   *IPRateLimiter
	userLimiter *UserRateLimiter
	config      RateLimitConfig
}

// NewGlobalRateLimiter creates a new global rate limiter from a YAML configuration file.
// Loads rate limit settings from the specified config file and applies default values
// for any missing or invalid configuration. Returns an error if the file cannot be read
// or parsed, or if the YAML structure is invalid.
func NewGlobalRateLimiter(configPath string) (*GlobalRateLimiter, error) {
	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %s", err)
	}
	config := RateLimitConfig{}
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to Unmarshal yaml file: %s", err)
	}

	// YAML validation
	applyDefaults(&config)
	return &GlobalRateLimiter{
		ipLimiter:   NewIPRateLimiter(&config),
		userLimiter: NewUserRateLimiter(&config),
		config:      config,
	}, nil
}

// Allow checks if a request should be allowed based on the current rate limits.
// If isIPBound is true, applies IP-based rate limiting using the identifier as an IP address.
// If isIPBound is false, applies user-based rate limiting using the identifier as a user ID.
// Returns true if the request is allowed, false if rate limited.
// Returns an error if the identifier or request type is invalid.
// Thread-safe for concurrent access.
func (grl *GlobalRateLimiter) Allow(identifier string, requestType string, isIPBound bool) (bool, error) {
	if isIPBound {
		bucket, err := grl.ipLimiter.GetBucket(identifier, requestType)
		if err != nil {
			return false, fmt.Errorf("got an error: %s", err)
		}
		return bucket.Allow(), nil
	}
	bucket, err := grl.userLimiter.GetBucket(identifier, requestType)
	if err != nil {
		return false, fmt.Errorf("got an error: %s", err)
	}
	return bucket.Allow(), nil
}

func applyDefaults(config *RateLimitConfig) {
	defaults := []struct {
		field      *int
		defaultVal int
	}{
		{&config.RegisterRequest.Refill, 33},
		{&config.RegisterRequest.Capacity, 2},
		{&config.LoginChallengeRequest.Refill, 33},
		{&config.LoginChallengeRequest.Capacity, 2},
		{&config.FindRoom.Refill, 500},
		{&config.FindRoom.Capacity, 30},
		{&config.MessageRequest.Refill, 1670},
		{&config.MessageRequest.Capacity, 100},
	}
	for _, d := range defaults {
		if *d.field <= 0 {
			*d.field = d.defaultVal
		}
	}
}

// GetConfig returns a copy of the current rate limit configuration.
// This is primarily used for testing and debugging purposes.
func (grl *GlobalRateLimiter) GetConfig() RateLimitConfig {
	return grl.config
}
