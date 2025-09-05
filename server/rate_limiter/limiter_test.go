package ratelimiter_test

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/GrGLeo/ctf/server/rate_limiter"
)

func TestNewGlobalRateLimiter(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		wantErr    bool
	}{
		{
			name:       "valid config file",
			configPath: "rate_limiter.yaml",
			wantErr:    false,
		},
		{
			name:       "invalid config path",
			configPath: "nonexistent.yaml",
			wantErr:    true,
		},
		{
			name:       "empty config path",
			configPath: "",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := ratelimiter.NewGlobalRateLimiter(tt.configPath)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("NewGlobalRateLimiter() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("NewGlobalRateLimiter() succeeded unexpectedly")
			}
			if got == nil {
				t.Fatal("NewGlobalRateLimiter() returned nil")
			}
		})
	}
}

func TestNewGlobalRateLimiter_InvalidYAMLValues(t *testing.T) {
	// Create a temporary YAML file with invalid values
	invalidYAML := `
register_request:
  capacity: -1
  refill: 0
login_challenge_request:
  capacity: 0
  refill: -5
find-room:
  capacity: -10
  refill: 0
message_request:
  capacity: 0
  refill: -100
`
	tmpFile, err := os.CreateTemp("", "invalid_config_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(invalidYAML); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Test that defaults are applied for invalid values
	grl, err := ratelimiter.NewGlobalRateLimiter(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewGlobalRateLimiter() failed: %v", err)
	}

	// Verify defaults were applied
	expectedDefaults := ratelimiter.RateLimitConfig{
		RegisterRequest: ratelimiter.BucketConfig{
			Capacity: 2,
			Refill:   33,
		},
		LoginChallengeRequest: ratelimiter.BucketConfig{
			Capacity: 2,
			Refill:   33,
		},
		FindRoom: ratelimiter.BucketConfig{
			Capacity: 30,
			Refill:   500,
		},
		MessageRequest: ratelimiter.BucketConfig{
			Capacity: 100,
			Refill:   1670,
		},
	}

	actualConfig := grl.GetConfig()
	if actualConfig != expectedDefaults {
		t.Errorf("Defaults not applied correctly.\nGot: %+v\nExpected: %+v", actualConfig, expectedDefaults)
	}
}

func TestGlobalRateLimiter_Allow(t *testing.T) {
	grl, err := ratelimiter.NewGlobalRateLimiter("rate_limiter.yaml")
	if err != nil {
		t.Fatalf("Failed to create GlobalRateLimiter: %v", err)
	}

	tests := []struct {
		name        string
		identifier  string
		requestType string
		isIPBound   bool
		wantErr     bool
	}{
		{
			name:        "IP-bound register request",
			identifier:  "192.168.1.1",
			requestType: "register_request",
			isIPBound:   true,
			wantErr:     false,
		},
		{
			name:        "IP-bound login request",
			identifier:  "192.168.1.1",
			requestType: "login_challenge_request",
			isIPBound:   true,
			wantErr:     false,
		},
		{
			name:        "user-bound find room request",
			identifier:  "user123",
			requestType: "find-room",
			isIPBound:   false,
			wantErr:     false,
		},
		{
			name:        "user-bound message request",
			identifier:  "user123",
			requestType: "message_request",
			isIPBound:   false,
			wantErr:     false,
		},
		{
			name:        "invalid IP-bound request type",
			identifier:  "192.168.1.1",
			requestType: "invalid-request",
			isIPBound:   true,
			wantErr:     true,
		},
		{
			name:        "invalid user-bound request type",
			identifier:  "user123",
			requestType: "invalid-request",
			isIPBound:   false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := grl.Allow(tt.identifier, tt.requestType, tt.isIPBound)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Allow() should have failed for %s", tt.name)
				}
				return
			}
			if err != nil {
				t.Errorf("Allow() failed unexpectedly: %v", err)
				return
			}
			// For valid requests, we expect true (allowed) initially
			if !allowed {
				t.Logf("Request was rate limited (this is expected after capacity is reached)")
			}
		})
	}
}

func TestIPRateLimiter_GetBucket(t *testing.T) {
	config := &ratelimiter.RateLimitConfig{
		RegisterRequest: ratelimiter.BucketConfig{
			Capacity: 2,
			Refill:   33,
		},
		LoginChallengeRequest: ratelimiter.BucketConfig{
			Capacity: 2,
			Refill:   33,
		},
	}

	ipLimiter := ratelimiter.NewIPRateLimiter(config)

	// Test lazy initialization
	t.Run("lazy initialization", func(t *testing.T) {
		ip := "192.168.1.1"
		bucket1, err := ipLimiter.GetBucket(ip, "register_request")
		if err != nil {
			t.Fatalf("GetBucket() failed: %v", err)
		}
		if bucket1 == nil {
			t.Fatal("bucket should not be nil")
		}

		// Second call should return same bucket
		bucket2, err := ipLimiter.GetBucket(ip, "register_request")
		if err != nil {
			t.Fatalf("GetBucket() failed: %v", err)
		}
		if bucket1 != bucket2 {
			t.Error("Second call should return same bucket instance")
		}
	})

	// Test invalid request type
	t.Run("invalid request type", func(t *testing.T) {
		_, err := ipLimiter.GetBucket("192.168.1.1", "invalid-request")
		if err == nil {
			t.Error("GetBucket() should fail for invalid request type")
		}
	})

	// Test different IPs get different limiters
	t.Run("different IPs", func(t *testing.T) {
		ip1 := "192.168.1.1"
		ip2 := "192.168.1.2"

		bucket1, err := ipLimiter.GetBucket(ip1, "register_request")
		if err != nil {
			t.Fatalf("GetBucket() failed: %v", err)
		}

		bucket2, err := ipLimiter.GetBucket(ip2, "register_request")
		if err != nil {
			t.Fatalf("GetBucket() failed: %v", err)
		}

		if bucket1 == bucket2 {
			t.Error("Different IPs should get different bucket instances")
		}
	})
}

func TestUserRateLimiter_GetBucket(t *testing.T) {
	config := &ratelimiter.RateLimitConfig{
		FindRoom: ratelimiter.BucketConfig{
			Capacity: 30,
			Refill:   500,
		},
		MessageRequest: ratelimiter.BucketConfig{
			Capacity: 100,
			Refill:   1670,
		},
	}

	userLimiter := ratelimiter.NewUserRateLimiter(config)

	// Test lazy initialization
	t.Run("lazy initialization", func(t *testing.T) {
		user := "user123"
		bucket1, err := userLimiter.GetBucket(user, "find-room")
		if err != nil {
			t.Fatalf("GetBucket() failed: %v", err)
		}
		if bucket1 == nil {
			t.Fatal("bucket should not be nil")
		}

		// Second call should return same bucket
		bucket2, err := userLimiter.GetBucket(user, "find-room")
		if err != nil {
			t.Fatalf("GetBucket() failed: %v", err)
		}
		if bucket1 != bucket2 {
			t.Error("Second call should return same bucket instance")
		}
	})

	// Test invalid request type
	t.Run("invalid request type", func(t *testing.T) {
		_, err := userLimiter.GetBucket("user123", "invalid-request")
		if err == nil {
			t.Error("GetBucket() should fail for invalid request type")
		}
	})

	// Test different users get different limiters
	t.Run("different users", func(t *testing.T) {
		user1 := "user123"
		user2 := "user456"

		bucket1, err := userLimiter.GetBucket(user1, "find-room")
		if err != nil {
			t.Fatalf("GetBucket() failed: %v", err)
		}

		bucket2, err := userLimiter.GetBucket(user2, "find-room")
		if err != nil {
			t.Fatalf("GetBucket() failed: %v", err)
		}

		if bucket1 == bucket2 {
			t.Error("Different users should get different bucket instances")
		}
	})
}

func TestRateLimiting_Enforcement(t *testing.T) {
	grl, err := ratelimiter.NewGlobalRateLimiter("rate_limiter.yaml")
	if err != nil {
		t.Fatalf("Failed to create GlobalRateLimiter: %v", err)
	}

	// Test IP-based rate limiting (register request has capacity 2)
	t.Run("IP rate limiting", func(t *testing.T) {
		ip := "192.168.1.1"

		// First 2 requests should be allowed
		for i := range 2 {
			allowed, err := grl.Allow(ip, "register_request", true)
			if err != nil {
				t.Fatalf("Allow() failed: %v", err)
			}
			if !allowed {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}

		// Third request should be denied
		allowed, err := grl.Allow(ip, "register_request", true)
		if err != nil {
			t.Fatalf("Allow() failed: %v", err)
		}
		if allowed {
			t.Error("Third request should be rate limited")
		}
	})

	// Test user-based rate limiting (find-room has capacity 30)
	t.Run("user rate limiting", func(t *testing.T) {
		user := "user123"

		// First 30 requests should be allowed
		allowedCount := 0
		for range 35 {
			allowed, err := grl.Allow(user, "find-room", false)
			if err != nil {
				t.Fatalf("Allow() failed: %v", err)
			}
			if allowed {
				allowedCount++
			}
		}

		if allowedCount != 30 {
			t.Errorf("Expected 30 requests to be allowed, got %d", allowedCount)
		}
	})
}

func TestIPRateLimiter_ConcurrentSameIP(t *testing.T) {
	config := &ratelimiter.RateLimitConfig{
		RegisterRequest: ratelimiter.BucketConfig{
			Capacity: 100,
			Refill:   1000,
		},
		LoginChallengeRequest: ratelimiter.BucketConfig{
			Capacity: 100,
			Refill:   1000,
		},
	}

	ipLimiter := ratelimiter.NewIPRateLimiter(config)

	ip := "192.168.1.1"
	numGoroutines := 50
	numCalls := 100

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range numCalls {
				_, err := ipLimiter.GetBucket(ip, "register_request")
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	// Verify all calls succeeded (no race conditions)
	totalCalls := int64(numGoroutines * numCalls)
	if successCount != totalCalls {
		t.Errorf("Expected %d successful calls, got %d (errors: %d)",
			totalCalls, successCount, errorCount)
	}

	// Verify only one limiter instance was created
	limiterCount := ipLimiter.GetLimiterCount()
	if limiterCount != 1 {
		t.Errorf("Expected 1 limiter instance, got %d", limiterCount)
	}
}

func TestGlobalRateLimiter_ConcurrentSameIPAllow(t *testing.T) {
	grl, err := ratelimiter.NewGlobalRateLimiter("rate_limiter.yaml")
	if err != nil {
		t.Fatalf("Failed to create GlobalRateLimiter: %v", err)
	}

	ip := "192.168.1.1"
	numGoroutines := 10
	numCalls := 5 // Within capacity limit

	var wg sync.WaitGroup
	var allowedCount int64
	var deniedCount int64

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range numCalls {
				allowed, err := grl.Allow(ip, "register_request", true)
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if allowed {
					atomic.AddInt64(&allowedCount, 1)
				} else {
					atomic.AddInt64(&deniedCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	// With capacity=2, should allow exactly 2 requests
	if allowedCount != 2 {
		t.Errorf("Expected 2 allowed requests, got %d", allowedCount)
	}

	totalRequests := int64(numGoroutines * numCalls)
	if deniedCount != totalRequests-2 {
		t.Errorf("Expected %d denied requests, got %d",
			totalRequests-2, deniedCount)
	}
}

func TestGlobalRateLimiter_ConcurrentMixedAccess(t *testing.T) {
	grl, err := ratelimiter.NewGlobalRateLimiter("rate_limiter.yaml")
	if err != nil {
		t.Fatalf("Failed to create GlobalRateLimiter: %v", err)
	}

	numGoroutines := 20
	var wg sync.WaitGroup

	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Alternate between IP and user requests
			if id%2 == 0 {
				ip := fmt.Sprintf("192.168.1.%d", id%10)
				_, err := grl.Allow(ip, "register_request", true)
				if err != nil {
					t.Errorf("IP request failed: %v", err)
				}
			} else {
				user := fmt.Sprintf("user%d", id%10)
				_, err := grl.Allow(user, "find-room", false)
				if err != nil {
					t.Errorf("User request failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()
	// Test completes without race conditions or deadlocks
}

// Benchmark functions for performance testing

func BenchmarkIPRateLimiter_GetBucket(b *testing.B) {
	config := &ratelimiter.RateLimitConfig{
		RegisterRequest: ratelimiter.BucketConfig{
			Capacity: 1000,
			Refill:   1000,
		},
		LoginChallengeRequest: ratelimiter.BucketConfig{
			Capacity: 1000,
			Refill:   1000,
		},
	}

	ipLimiter := ratelimiter.NewIPRateLimiter(config)
	ip := "192.168.1.100"

	// Pre-warm the limiter
	_, _ = ipLimiter.GetBucket(ip, "register_request")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = ipLimiter.GetBucket(ip, "register_request")
		}
	})
}

func BenchmarkUserRateLimiter_GetBucket(b *testing.B) {
	config := &ratelimiter.RateLimitConfig{
		FindRoom: ratelimiter.BucketConfig{
			Capacity: 1000,
			Refill:   1000,
		},
		MessageRequest: ratelimiter.BucketConfig{
			Capacity: 1000,
			Refill:   1000,
		},
	}

	userLimiter := ratelimiter.NewUserRateLimiter(config)
	user := "user123"

	// Pre-warm the limiter
	_, _ = userLimiter.GetBucket(user, "find-room")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = userLimiter.GetBucket(user, "find-room")
		}
	})
}

func BenchmarkGlobalRateLimiter_Allow_IP(b *testing.B) {
	grl, _ := ratelimiter.NewGlobalRateLimiter("rate_limiter.yaml")
	ip := "192.168.1.100"

	// Pre-warm the limiter
	_, _ = grl.Allow(ip, "register_request", true)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = grl.Allow(ip, "register_request", true)
		}
	})
}

func BenchmarkGlobalRateLimiter_Allow_User(b *testing.B) {
	grl, _ := ratelimiter.NewGlobalRateLimiter("rate_limiter.yaml")
	user := "user123"

	// Pre-warm the limiter
	_, _ = grl.Allow(user, "find-room", false)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = grl.Allow(user, "find-room", false)
		}
	})
}

func BenchmarkLazyInitialization_IP(b *testing.B) {
	config := &ratelimiter.RateLimitConfig{
		RegisterRequest: ratelimiter.BucketConfig{
			Capacity: 1000,
			Refill:   1000,
		},
		LoginChallengeRequest: ratelimiter.BucketConfig{
			Capacity: 1000,
			Refill:   1000,
		},
	}

	ipLimiter := ratelimiter.NewIPRateLimiter(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// Use unique IPs to force lazy initialization
		ipCounter := 0
		for pb.Next() {
			ip := fmt.Sprintf("192.168.1.%d", ipCounter%1000)
			_, _ = ipLimiter.GetBucket(ip, "register_request")
			ipCounter++
		}
	})
}

func BenchmarkLazyInitialization_User(b *testing.B) {
	config := &ratelimiter.RateLimitConfig{
		FindRoom: ratelimiter.BucketConfig{
			Capacity: 1000,
			Refill:   1000,
		},
		MessageRequest: ratelimiter.BucketConfig{
			Capacity: 1000,
			Refill:   1000,
		},
	}

	userLimiter := ratelimiter.NewUserRateLimiter(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// Use unique users to force lazy initialization
		userCounter := 0
		for pb.Next() {
			user := fmt.Sprintf("user%d", userCounter%1000)
			_, _ = userLimiter.GetBucket(user, "find-room")
			userCounter++
		}
	})
}

func BenchmarkConcurrentMixedAccess(b *testing.B) {
	grl, _ := ratelimiter.NewGlobalRateLimiter("rate_limiter.yaml")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ipCounter := 0
		userCounter := 0
		for pb.Next() {
			// Alternate between IP and user requests
			if ipCounter%2 == 0 {
				ip := fmt.Sprintf("192.168.1.%d", ipCounter%100)
				_, _ = grl.Allow(ip, "register_request", true)
				ipCounter++
			} else {
				user := fmt.Sprintf("user%d", userCounter%100)
				_, _ = grl.Allow(user, "find-room", false)
				userCounter++
			}
		}
	})
}

func BenchmarkMemoryUsage(b *testing.B) {
	config := &ratelimiter.RateLimitConfig{
		RegisterRequest: ratelimiter.BucketConfig{
			Capacity: 100,
			Refill:   10,
		},
		LoginChallengeRequest: ratelimiter.BucketConfig{
			Capacity: 100,
			Refill:   10,
		},
		FindRoom: ratelimiter.BucketConfig{
			Capacity: 100,
			Refill:   10,
		},
		MessageRequest: ratelimiter.BucketConfig{
			Capacity: 100,
			Refill:   10,
		},
	}

	ipLimiter := ratelimiter.NewIPRateLimiter(config)
	userLimiter := ratelimiter.NewUserRateLimiter(config)

	// Create many unique IPs and users
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ip := fmt.Sprintf("192.168.1.%d", i%1000)
		user := fmt.Sprintf("user%d", i%1000)

		_, _ = ipLimiter.GetBucket(ip, "register_request")
		_, _ = userLimiter.GetBucket(user, "find-room")
	}
}
