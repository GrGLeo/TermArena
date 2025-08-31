# E2E Test Suite

This directory contains end-to-end tests for the CTF game's messaging system.

## Overview

The E2E test suite validates the complete messaging functionality from client connections through server processing to message delivery. It includes comprehensive test scenarios for different message types and error conditions.

## Test Scenarios

### 1. Basic Two Client Test
- **Clients**: `user1`, `user2`
- **Purpose**: Validates fundamental messaging functionality
- **Tests**: Basic message sending, client authentication, timing

### 2. Multi Client Room Test
- **Clients**: `user1`, `user2`, `user3`, `user4`
- **Purpose**: Tests group chat functionality
- **Tests**: Room-based broadcasting, multiple participants, message ordering

### 3. Whisper Message Test
- **Clients**: `user1`, `user2`
- **Purpose**: Validates private messaging
- **Tests**: Direct messaging syntax (`/username`), message filtering

### 4. Broadcast Message Test
- **Clients**: `user1`, `user2`, `user3`, `user4`
- **Purpose**: Tests system-wide broadcasting
- **Tests**: `/all` command, universal message delivery

### 5. Error Handling Test
- **Clients**: `user1`, `user2`
- **Purpose**: Validates error scenarios
- **Tests**: Empty messages, invalid targets, error recovery

### 6. Concurrent Messaging Test
- **Clients**: `user1`, `user2`, `user3`, `user4`
- **Purpose**: Tests system under concurrent load
- **Tests**: Race conditions, concurrent processing, performance

## Usage

### Running All Tests
```bash
cd test/e2e
go run main.go
```

### Running Unit Tests
```bash
cd test/e2e
go test -v
```

### Building the Test Client
```bash
cd test/e2e
go build -o e2e_test
```

## Requirements

- Go 1.23.2+
- Server running on `localhost:8082`
- Authentication service running
- Message service running
- User keys available in `~/.config/term_arena/keys/`

## User Management

The test suite reuses users `user1` through `user4` for all scenarios to ensure:
- Consistent test environment
- Proper connection cleanup between tests
- Resource management

## Monitoring

The test client includes comprehensive logging for:
- Connection establishment/disconnection
- Message sending/receiving
- Authentication flow
- Error conditions
- Performance metrics

## Integration

This test suite integrates with:
- **Server**: `../server/` - Main game server
- **Shared**: `../shared/` - Common utilities and packet handling
- **Client**: `../client/` - Client application (for dependencies)

## Architecture

```
test/e2e/
├── main.go           # Test scenarios and orchestration
├── main_test.go      # Unit tests for test utilities
├── go.mod           # Module dependencies
├── go.sum           # Dependency checksums
├── e2e_test         # Compiled test binary
└── README.md        # This documentation
```

## Best Practices

1. **User Reuse**: Tests reuse the same users to ensure proper cleanup
2. **Sequential Execution**: Tests run sequentially to avoid conflicts
3. **Connection Cleanup**: Each test properly closes connections
4. **Error Isolation**: Failures in one test don't affect others
5. **Comprehensive Logging**: Detailed logs for debugging

## Troubleshooting

### Common Issues

1. **Connection Refused**: Ensure server is running on port 8082
2. **Authentication Failed**: Check user keys exist in the expected location
3. **Test Timeouts**: Verify all services are responding within expected timeframes

### Debug Mode

Enable verbose logging by setting the environment variable:
```bash
export LOG_LEVEL=debug
```

## Contributing

When adding new test scenarios:
1. Follow the existing pattern in `main.go`
2. Add appropriate user management
3. Include proper error handling
4. Update this README with the new scenario details