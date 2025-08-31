# Message Service

The Message Service is a microservice responsible for handling real-time messaging between clients in the CTF game. It provides both broadcast messaging to all players in a room and private messaging between individual players, with support for different message routing strategies.

## Key Features

- **Real-time Messaging:** Supports instant message delivery between connected clients
- **Multiple Routing Modes:** Broadcast to all players, private messaging, and room-based messaging
- **Message Validation:** Client-side and server-side validation of message content and routing
- **Persistent Connections:** Maintains TCP connections for reliable message delivery
- **Graceful Shutdown:** Proper cleanup and shutdown handling with configurable timeouts
- **Structured Logging:** JSON-formatted logging with configurable log levels

## Architecture

The message service is implemented as a standalone Go microservice that communicates with the main game server via gRPC. It maintains persistent TCP connections with clients and handles message routing logic.

### Components

1. **Message Service (services/message_service/):**
   - Main service entry point with graceful shutdown
   - Configuration management
   - gRPC server implementation
   - Client connection management

2. **Server Handler (server/handlers/messages.go):**
   - Integration layer with the main game server
   - gRPC client for communicating with message service
   - Event-driven message processing

3. **Client Model (client/model/messaging.go):**
   - TUI-based messaging interface using Bubble Tea
   - Message history and scrolling viewport
   - Input validation and command parsing
   - Real-time message display with styling

## Message Routing

The service supports several message routing patterns:

- **Broadcast (/all):** Messages sent to all players in the current room
- **Private (/userID):** Direct messages to a specific player
- **Room-based:** Messages scoped to the current game room
- **System Messages:** Automated messages from the server (errors, notifications)

## gRPC API Definition

The service's API is defined in `proto/message/message.proto`.

### `MessageService`

#### `RegisterClient`
- **Request:** `RegisterClientRequest { string client, string room_id }`
- **Response:** `RegisterClientResponse {}`
- **Description:** Registers a client with the message service for a specific room. The client ID and room ID are stored for message routing.

#### `UnregisterClient`
- **Request:** `UnregisterClientRequest { string client }`
- **Response:** `UnregisterClientResponse {}`
- **Description:** Removes a client from the message service, cleaning up any associated state.

#### `RouteMessage`
- **Request:** `RouteMessageRequest { string sender, string content }`
- **Response:** `RouteMessageResponse { repeated string receivers, string content }`
- **Description:** Routes a message from a sender to appropriate receivers based on the message content and routing rules. Returns the list of receivers and processed message content.

## Client-Side Implementation

### Messaging Interface

The client provides a terminal-based messaging interface with:

- **Input Field:** Text input for composing messages with placeholder hints
- **Message History:** Scrollable viewport showing recent messages (limited to 100 messages)
- **Command Support:** Special commands for routing (/all, /userID)
- **Real-time Updates:** Live message display with timestamps
- **Styling:** Color-coded messages for different types (system, own, others, broadcasts)

### Message Validation

Client-side validation includes:
- Non-empty message check
- Maximum length limit (256 characters)
- Command format validation for routing prefixes
- User ID validation for private messages

## Server-Side Integration

### Connection Management

The server handler integrates with the existing connection manager to:
- Register clients when they connect to a room
- Unregister clients on disconnection
- Route messages through the event system
- Handle errors and timeouts gracefully

### Event Processing

Messages are processed through the server's event system:
- `ClientRegistrationMessage`: Triggers client registration with message service
- `ClientUnregistrationMessage`: Triggers client cleanup
- `MessageRequestMessage`: Routes user messages
- Response events deliver messages back to clients

## Configuration

The message service supports configuration via environment variables:

- `MESSAGE_SERVICE_ADDR`: Service address (default: "localhost:8083")
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
- `SHUTDOWN_TIMEOUT_SECONDS`: Graceful shutdown timeout

## Code Structure

### Service Layer (services/message_service/)
- **`main.go`:** Service entry point with logging and shutdown handling
- **`server.go`:** gRPC server implementation and client management
- **`config.go`:** Configuration loading and validation
- **`handler.go`:** gRPC method implementations
- **`manager.go`:** Client and room management logic

### Server Integration (server/handlers/)
- **`messages.go`:** Message service client and event handlers

### Client Implementation (client/model/)
- **`messaging.go`:** Bubble Tea model for messaging interface

## Security Considerations

- **Input Validation:** All messages are validated for length and format
- **Connection Security:** Uses TCP connections (can be upgraded to TLS)
- **Rate Limiting:** Should be implemented at the service level for production
- **Message Sanitization:** Content is trimmed and validated before processing
