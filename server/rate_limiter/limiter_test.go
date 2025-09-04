package ratelimiter_test

import (
	"fmt"
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
			requestType: "register-request",
			isIPBound:   true,
			wantErr:     false,
		},
		{
			name:        "IP-bound login request",
			identifier:  "192.168.1.1",
			requestType: "login-request-challenge",
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
		bucket1, err := ipLimiter.GetBucket(ip, "register-request")
		if err != nil {
			t.Fatalf("GetBucket() failed: %v", err)
		}
		if bucket1 == nil {
			t.Fatal("bucket should not be nil")
		}

		// Second call should return same bucket
		bucket2, err := ipLimiter.GetBucket(ip, "register-request")
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

		bucket1, err := ipLimiter.GetBucket(ip1, "register-request")
		if err != nil {
			t.Fatalf("GetBucket() failed: %v", err)
		}

		bucket2, err := ipLimiter.GetBucket(ip2, "register-request")
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
		for i := 0; i < 2; i++ {
			allowed, err := grl.Allow(ip, "register-request", true)
			if err != nil {
				t.Fatalf("Allow() failed: %v", err)
			}
			if !allowed {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}

		// Third request should be denied
		allowed, err := grl.Allow(ip, "register-request", true)
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
		for i := 0; i < 35; i++ {
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
	_, _ = ipLimiter.GetBucket(ip, "register-request")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = ipLimiter.GetBucket(ip, "register-request")
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
	_, _ = grl.Allow(ip, "register-request", true)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = grl.Allow(ip, "register-request", true)
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
			_, _ = ipLimiter.GetBucket(ip, "register-request")
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
				_, _ = grl.Allow(ip, "register-request", true)
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

		_, _ = ipLimiter.GetBucket(ip, "register-request")
		_, _ = userLimiter.GetBucket(user, "find-room")
	}
}
