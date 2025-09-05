# Rate Limiter

The rate limiter is a critical security component that protects the TermArena game server from abuse and resource exhaustion attacks. It implements a token bucket algorithm to control request rates across different types of operations.

## Overview

The rate limiter uses a **token bucket algorithm** where:
- Each request type has its own bucket with a defined capacity
- Tokens are consumed when requests are made
- Tokens are automatically refilled over time at a configurable rate
- When a bucket is empty, requests are rejected until more tokens are available

## Architecture

The rate limiter supports two types of rate limiting:

### IP-Based Rate Limiting
- Used for authentication-related requests (registration, login challenges)
- Limits are applied per IP address to prevent abuse from single sources
- Protects against brute force attacks and spam registration

### User-Based Rate Limiting
- Used for authenticated user actions (messaging, room finding)
- Limits are applied per user account
- Prevents individual users from overwhelming the system

## Configuration

Rate limits are configured via the `server/rate_limiter/rate_limiter.yaml` file:

```yaml
register_request:
  capacity: 2      # Maximum tokens in bucket
  refill: 33       # Tokens added per second (scaled by 1000)
login_challenge_request:
  capacity: 2
  refill: 33
find-room:
  capacity: 30
  refill: 500
message_request:
  capacity: 30
  refill: 500
```

### Rate Calculations

- **Registration/Login**: ~2 requests per 30 seconds (33ms token refill rate)
- **Messages/Room Finding**: ~30 requests per 3.6 seconds (500ms token refill rate)

## Implementation Details

### Token Bucket Algorithm

```go
type TokenBucket struct {
    capacity    int64  // Maximum tokens
    tokens      int64  // Current tokens
    refillRate  int64  // Tokens per second (scaled)
    lastRefill  time.Time
}
```

The bucket automatically refills tokens based on elapsed time:
```go
tokensToAdd = (refillRate * elapsedMs) / 1000000
```

### Lazy Initialization

Rate limiters are created only when first accessed:
- **IPRateLimiter**: Creates buckets per IP address on first request
- **UserRateLimiter**: Creates buckets per user on first request
- Reduces memory usage for unused identifiers

### Automatic Cleanup

Unused rate limiters are automatically cleaned up:
- Runs every hour in a background goroutine
- Removes limiters inactive for 24+ hours
- Prevents memory leaks from temporary users/IPs

### Thread Safety

All operations are thread-safe using mutexes:
- **GlobalRateLimiter**: Protects configuration access
- **IPRateLimiter/UserRateLimiter**: Protects per-identifier access
- **TokenBucket**: Protects token count modifications

## Integration

### Authentication Handler

IP-based rate limiting for registration and login:

```go
func (ac *AuthClient) HandleRegistration(msg event.Message) event.Message {
    req := msg.(event.RegisterRequestMessage)
    ip, _ := shared.ExtractIP(req.Conn)

    allowed, err := ac.rateLimiter.Allow(ip, req.Type(), true) // IP-based
    if !allowed {
        return event.RateLimitResponse{ResponseCh: req.ResponseCh}
    }
    // Process registration...
}
```

### Message Handler

User-based rate limiting for messaging:

```go
func (ms *MessagesServiceClient) HandleRouteMessage(msg event.Message) event.Message {
    req := msg.(event.MessageRequestMessage)

    allowed, err := ms.rateLimiter.Allow(req.User, req.Type(), false) // User-based
    if !allowed {
        return event.MessageErrorResponse{
            Error: "Rate limit exceeded",
            User: req.User,
        }
    }
    // Process message...
}
```

## Security Benefits

### Protection Against Common Attacks

1. **Brute Force Registration**: Limits registration attempts per IP
2. **Authentication Spam**: Prevents rapid login challenge requests
3. **Message Flooding**: Controls messaging rate per user
4. **Resource Exhaustion**: Prevents overwhelming server resources

### Rate Limit Response

When rate limits are exceeded, clients receive:
- **RateLimitResponse**: For auth-related requests
- **MessageErrorResponse**: For messaging requests
- Proper logging for monitoring and debugging

## Performance Considerations

### Memory Efficiency
- Lazy initialization prevents memory waste
- Automatic cleanup removes unused limiters
- Minimal memory footprint for active users

### CPU Efficiency
- Lock contention minimized through fine-grained locking
- Background cleanup runs infrequently (every hour)
- Token refill calculations are lightweight

### Scalability
- Supports thousands of concurrent users
- Efficient hash map lookups for IP/user identification
- Configurable limits allow tuning for different deployment sizes

## Testing

The rate limiter is thoroughly tested with:

### Unit Tests (`bucket_test.go`, `limiter_test.go`)
- Token bucket refill calculations
- Rate limit enforcement
- Thread safety validation
- Configuration loading

### Integration Tests (`handlers/auth_unit_test.go`)
- End-to-end rate limiting in auth flow
- Proper error responses
- Configuration validation

### E2E Tests (`test/e2e/main.go`)
- Registration rate limit testing
- Authentication rate limit testing
- Message rate limit testing
- Proper timing between test scenarios
