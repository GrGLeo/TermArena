 Go Server

The Go server acts as the central orchestrator of the TermArena game. It is responsible for managing client connections, handling authentication, and coordinating game rooms.

## Key Responsibilities

- **Client Connection Handling:** Listens for incoming TCP connections from clients and manages the lifecycle of each connection.
- **Authentication:** Communicates with the [Rust Auth Service](./auth_rust.md) to authenticate users.
- **Rate Limiting:** Implements [token bucket rate limiting](./rate_limiter.md) to prevent abuse and resource exhaustion.
- **Room Management:** Allows players to create, find, and join game rooms.
- **Real-time Messaging:** Integrates with the [Message Service](./message_service.md) to handle player-to-player messaging.
- **Game Server Orchestration:** Spawns a new Rust game server process when a game room is created.

## Architecture

The Go server is built using an event-driven architecture, which makes it highly modular and extensible.

### Event Broker

The core of the server is the event broker, which is responsible for decoupling the different components. When a message is received from a client, it is published to the event broker, which then routes it to the appropriate subscriber.

This approach allows for a clean separation of concerns and makes it easy to add new functionality without modifying existing code.

### Concurrency

The server is highly concurrent, with each client connection being handled in its own goroutine. This allows the server to handle a large number of simultaneous connections without blocking.

## Code Structure

The Go server's code is organized into the following packages:

- **`main.go`:** The entry point of the server, responsible for initializing the server and starting the event loop.
- **`authentification/`:** Handles communication with the Rust Auth Service.
- **`handlers/messages.go`:** Integrates with the Message Service for real-time messaging functionality.
- **`event/`:** Implements the event broker and defines the message types.
- **`room_manager.go/`:** Manages the creation, discovery, and joining of game rooms.
- **`rate_limiter/`:** Implements token bucket rate limiting to prevent abuse.
- **`shared/`:** Contains code that is shared with the client, such as packet definitions and serialization functions.

## Security Features

The Go server includes several security measures to protect against common attacks:

### Rate Limiting
- **IP-based limiting** for authentication requests (registration, login challenges)
- **User-based limiting** for authenticated actions (messaging, room finding)
- **Token bucket algorithm** with configurable capacity and refill rates
- **Automatic cleanup** of unused rate limiters to prevent memory leaks

### Attack Protection
- **Brute force prevention** on registration and authentication
- **Message flooding protection** per user account
- **Resource exhaustion prevention** through request throttling
- **Comprehensive logging** for security monitoring
