package ratelimiter

import (
	"sync"
	"time"
)

type TokenBucket struct {
	mu             sync.Mutex
	capacity       int64
	tokens         int64
	refillRate     int64 // tokens per second scaled by 1000 (e.g., 1670 = 1.67 tokens/sec)
	incrementToken int64
	lastRefill     time.Time
}

func NewTokenBucket(capacity, refillRate int64) *TokenBucket {
  return &TokenBucket{
    capacity:       capacity,
    tokens:         capacity, // Start with full bucket
    refillRate:     refillRate,
    incrementToken: 0,
    lastRefill:     time.Now(),
  }
}

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
