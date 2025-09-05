package ratelimiter

import (
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type RateLimitConfig struct {
	RegisterRequest       BucketConfig `yaml:"register_request"`
	LoginChallengeRequest BucketConfig `yaml:"login_request_challenge"`
	FindRoom              BucketConfig `yaml:"find-room"`
	MessageRequest        BucketConfig `yaml:"message_request"`
}

type BucketConfig struct {
	Capacity int `yaml:"capacity"`
	Refill   int `yaml:"refill"`
}

type RateLimiter struct {
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
}

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

func NewIPRateLimiter(config *RateLimitConfig) *IPRateLimiter {
	irl := &IPRateLimiter{
		limiters:   make(map[string]*RateLimiter),
		lastAccess: make(map[string]time.Time),
		config:     config,
	}
	go irl.cleanupRoutine()
	return irl
}

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

func NewUserRateLimiter(config *RateLimitConfig) *UserRateLimiter {
	url := &UserRateLimiter{
		limiters:   make(map[string]*RateLimiter),
		lastAccess: make(map[string]time.Time),
		config:     config,
	}
	go url.cleanupRoutine()
	return url
}

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

func (grl *GlobalRateLimiter) GetConfig() RateLimitConfig {
	return grl.config
}
