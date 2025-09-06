package ratelimiter

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenBucket_BasicAllow(t *testing.T) {
	tb := NewTokenBucket(10, 1000) // 1 token per second

	// Should allow 10 requests initially
	for i := range 10 {
		if !tb.Allow() {
			t.Errorf("Expected allow on request %d, but got deny", i+1)
		}
	}

	// Should deny the 11th request
	if tb.Allow() {
		t.Error("Expected deny on 11th request, but got allow")
	}
}

func TestTokenBucket_RefillOverTime(t *testing.T) {
	tb := NewTokenBucket(10, 1000) // 1 token per second

	// Empty the bucket
	for range 10 {
		tb.Allow()
	}

	// Should deny immediately
	if tb.Allow() {
		t.Error("Expected deny when bucket is empty")
	}

	// Wait 2 seconds for refill
	time.Sleep(2 * time.Second)

	// Should allow 2 requests now
	allowed := 0
	for range 3 {
		if tb.Allow() {
			allowed++
		}
	}

	if allowed != 2 {
		t.Errorf("Expected 2 allowed requests after 2 seconds, got %d", allowed)
	}
}

func TestTokenBucket_CapacityLimit(t *testing.T) {
	tb := NewTokenBucket(5, 1000) // 1 token per second

	// Empty bucket
	for range 5 {
		tb.Allow()
	}

	// Wait for refill that would exceed capacity
	time.Sleep(10 * time.Second)

	// Should not exceed capacity
	if tb.tokens > 5 {
		t.Errorf("Bucket exceeded capacity: got %d, expected <= 5", tb.tokens)
	}
}

func TestTokenBucket_FractionalRate(t *testing.T) {
	// Test with 1.67 tokens per second (1670 in current scaling)
	tb := NewTokenBucket(10, 1670)

	// Empty bucket
	for range 10 {
		tb.Allow()
	}

	// Wait 1 second
	time.Sleep(1 * time.Second)

	// Check refill amount - should be approximately 1.67 tokens
	initialTokens := tb.tokens

	// Force refill by calling Allow
	tb.Allow()

	// The refill calculation should add approximately 1-2 tokens
	// (exact amount depends on scaling implementation)
	refilledTokens := tb.tokens - initialTokens

	if refilledTokens < 0 || refilledTokens > 3 {
		t.Errorf("Unexpected refill amount: got %d tokens", refilledTokens)
	}
}

func TestTokenBucket_NoRefillWhenFull(t *testing.T) {
	tb := NewTokenBucket(10, 1000)

	// Bucket should start full
	if tb.tokens != 10 {
		t.Errorf("Expected bucket to start full with 10 tokens, got %d", tb.tokens)
	}

	// Wait some time
	time.Sleep(1 * time.Second)

	// Tokens should not exceed capacity
	if tb.Allow() {
		if tb.tokens > 10 {
			t.Errorf("Tokens exceeded capacity after refill: got %d", tb.tokens)
		}
	}
}

func TestTokenBucket_HighFrequency(t *testing.T) {
	tb := NewTokenBucket(100, 1000) // 1 token per second

	// Test rapid successive calls
	allowed := 0
	denied := 0

	for range 200 {
		if tb.Allow() {
			allowed++
		} else {
			denied++
		}
		// Small delay to simulate high frequency but allow some refill
		time.Sleep(10 * time.Millisecond)
	}

	// Should have allowed approximately 100 requests initially
	if allowed < 95 || allowed > 105 {
		t.Errorf("Expected ~100 allowed requests, got %d allowed, %d denied", allowed, denied)
	}
}

func TestTokenBucket_ConcurrentAccess(t *testing.T) {
	tb := NewTokenBucket(1000, 10000) // High rate to allow many requests

	var allowed int64
	var denied int64

	// Run multiple goroutines to test thread safety
	numGoroutines := 10
	numRequests := 100

	var wg sync.WaitGroup
	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localAllowed := 0
			localDenied := 0
			for range numRequests {
				if tb.Allow() {
					localAllowed++
				} else {
					localDenied++
				}
			}
			// Use atomic operations for thread-safe counting
			atomic.AddInt64(&allowed, int64(localAllowed))
			atomic.AddInt64(&denied, int64(localDenied))
		}()
	}

	wg.Wait()

	totalRequests := allowed + denied
	expectedTotal := int64(numGoroutines * numRequests)

	if totalRequests != expectedTotal {
		t.Errorf("Expected %d total requests, got %d (allowed: %d, denied: %d)",
			expectedTotal, totalRequests, allowed, denied)
	}

	// Should have allowed most requests due to high refill rate
	if allowed < expectedTotal/2 {
		t.Errorf("Too many requests denied: allowed %d, denied %d", allowed, denied)
	}
}

func BenchmarkTokenBucket_Allow(b *testing.B) {
	tb := NewTokenBucket(1000, 1000) // High capacity to avoid refill overhead

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tb.Allow()
	}
}
