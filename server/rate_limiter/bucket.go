// Package ratelimiter provides a thread-safe rate limiting system using token bucket algorithm.
// It supports both IP-based and user-based rate limiting with lazy initialization
// and automatic cleanup of unused limiters.
package ratelimiter

import (
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter.
// It maintains a bucket of tokens that refill over time at a configurable rate.
// Thread-safe for concurrent use.
type TokenBucket struct {
	mu             sync.Mutex
	capacity       int64
	tokens         int64
	refillRate     int64 // tokens per second scaled by 1000 (e.g., 1670 = 1.67 tokens/sec)
	incrementToken int64
	lastRefill     time.Time
}

// NewTokenBucket creates a new token bucket with the specified capacity and refill rate.
// The bucket starts full and refills tokens at the given rate (tokens per second).
// The refillRate is scaled by 1000 internally (e.g., 1670 = 1.67 tokens/second).
func NewTokenBucket(capacity, refillRate int64) *TokenBucket {
	return &TokenBucket{
		capacity:       capacity,
		tokens:         capacity, // Start with full bucket
		refillRate:     refillRate,
		incrementToken: 0,
		lastRefill:     time.Now(),
	}
}

// NewTokenBucketFromConfig creates a new token bucket from a BucketConfig struct.
// This is a convenience function that extracts capacity and refill values from config.
func NewTokenBucketFromConfig(config BucketConfig) *TokenBucket {
	return NewTokenBucket(int64(config.Capacity), int64(config.Refill))
}

// Allow attempts to consume one token from the bucket.
// Returns true if a token was available and consumed, false if the bucket is empty.
// The bucket automatically refills tokens based on elapsed time since last refill.
// Thread-safe for concurrent use.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill the bucket based on elapsed time
	if tb.tokens < tb.capacity {
		durationMs := time.Now().Sub(tb.lastRefill).Milliseconds()
		tokensRefill := (tb.refillRate * durationMs) + (tb.incrementToken * 1000)
		tokensToAdd := tokensRefill / 1000000
		tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
		tb.incrementToken = (tokensRefill % 1000000) / 1000
		tb.lastRefill = time.Now()
	}

	// Check if we have tokens available
	if tb.tokens < 1 {
		return false
	}
	tb.tokens--
	return true
}
