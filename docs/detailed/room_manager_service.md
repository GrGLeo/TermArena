# Room Manager Service

The Room Manager Service is a microservice responsible for handling game room lifecycle management, player matchmaking, and real-time room state coordination in the TermArena MOBA game. It provides room creation, player assignment, team balancing, and spell selection management with support for different game modes.

## Key Features

- **Room Lifecycle Management:** Handles room creation, player joining, and state transitions from waiting to ready
- **Player Matchmaking:** Automatic team assignment and room capacity management
- **Real-time Notifications:** Bidirectional streaming for room state changes and player updates
- **Spell Selection:** Player spell configuration and validation during lobby phase
- **Multiple Game Modes:** Support for Sandbox (1 player), Practice (4 players), and Classic (8 players) modes
- **Graceful Shutdown:** Proper cleanup and shutdown handling with configurable timeouts
- **Structured Logging:** JSON-formatted logging with configurable log levels

## Architecture

The room manager service is implemented as a standalone Go microservice that communicates via gRPC. It maintains in-memory room state and handles concurrent player operations with proper synchronization.

### Components

1. **Room Manager Service (services/room_manager_service/):**
   - Main service entry point with graceful shutdown
   - Configuration management and environment variable loading
   - gRPC server implementation with bidirectional streaming
   - Room state management and player coordination

2. **Server Handler (server/handlers/):**
   - Integration layer with the main game server
   - gRPC client for communicating with room manager service
   - Event-driven room state processing

3. **Client Model (client/model/):**
   - TUI-based room interface using Bubble Tea
   - Real-time room status display and player information
   - Spell selection interface during lobby phase

## Room Lifecycle

The service manages rooms through several states:

- **WAITING:** Rooms with available slots that players can join
- **LOBBY:** Full rooms waiting for a 1-minute preparation timer
- **READY:** Rooms prepared for game start after timer expiration
- **PROGRESS:** Rooms currently in active gameplay

### State Transitions

1. **Room Creation:** New rooms start in WAITING state when first player joins
2. **Player Joining:** Players are assigned to teams (Blue/Red) with automatic balancing
3. **Room Filling:** When room reaches capacity, transitions to LOBBY state
4. **Preparation Timer:** 1-minute countdown begins in LOBBY state
5. **Game Ready:** Room transitions to READY state, notifying all players
6. **Game Start:** Room moves to PROGRESS when game server takes over

## gRPC API Definition

The service's API is defined in `pkg/proto/room_managing/room.proto`.

### `RoomService`

#### `LookRoom`
- **Request:** `LookRoomRequest { string username, uint32 roomType }`
- **Response:** `LookRoomResponse { uint32 roomID, uint32 team }`
- **Description:** Finds or creates a room for the player based on room type. Assigns the player to a team and returns room information.

#### `QuitRoom`
- **Request:** `QuitRoomRequest { string username, uint32 roomID }`
- **Response:** `QuitRoomResponse { bool success }`
- **Description:** Removes a player from their current room, updating team counts and room state.

#### `NotifyRoomChanges`
- **Request:** `stream Ack`
- **Response:** `stream RoomChangeNotification`
- **Description:** Bidirectional streaming RPC for real-time room state notifications. Server sends room updates, client acknowledges receipt.

#### `UpdateSpell`
- **Request:** `UpdateSpellRequest { string username, uint32 romType, uint32 roomID, uint32 spell1, uint32 spell2 }`
- **Response:** `UpdateSpellResponse { repeated string usernames, UserInfo user }`
- **Description:** Updates a player's spell selection during the lobby phase and notifies all players in the room of the change.

## Game Modes

### Sandbox Mode
- **Max Players:** 1
- **Purpose:** Single-player testing and development
- **Team Assignment:** Player assigned to Blue team
- **Transition:** Directly to LOBBY when joined

### Practice Mode
- **Max Players:** 4 (2v2)
- **Purpose:** Small-scale gameplay testing
- **Team Assignment:** Balanced between Blue and Red teams
- **Transition:** WAITING → LOBBY → READY

### Classic Mode
- **Max Players:** 8 (4v4)
- **Purpose:** Full competitive gameplay
- **Team Assignment:** Balanced between Blue and Red teams
- **Transition:** WAITING → LOBBY → READY

## Client-Side Implementation

### Room Interface

The client provides a terminal-based room management interface with:

- **Room Status Display:** Current room state, player count, and team assignments
- **Player List:** Real-time display of all players with team colors and spell selections
- **Spell Selection:** Interactive spell picker during lobby phase
- **Timer Display:** Countdown visualization for room preparation
- **Real-time Updates:** Live updates via bidirectional streaming

### Room State Management

Client-side state management includes:
- Room joining and team assignment handling
- Real-time player list updates
- Spell selection validation and submission
- Room transition notifications
- Error handling for room operations

## Server-Side Integration

### Connection Management

The server handler integrates with the existing architecture to:
- Register players with room manager when entering matchmaking
- Handle room state transitions and notifications
- Coordinate with game server for room handoff
- Manage player disconnections and cleanup

### Event Processing

Room events are processed through the server's event system:
- `RoomJoinEvent`: Triggers room lookup and player assignment
- `RoomQuitEvent`: Handles player removal from rooms
- `SpellUpdateEvent`: Processes spell selection changes
- `RoomReadyEvent`: Notifies game server of ready rooms

## Configuration

The room manager service supports configuration via environment variables:

- `ROOM_MANAGER_HOST`: Service host address (default: "localhost")
- `ROOM_MANAGER_PORT`: Service port (default: 8084)
- `ROOM_MANAGER_MAX_ROOM`: Maximum concurrent rooms (default: 100)
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
- `SHUTDOWN_TIMEOUT_SECONDS`: Graceful shutdown timeout

## Code Structure

### Service Layer (services/room_manager_service/)
- **`cmd/main.go`:** Service entry point with logging and shutdown handling
- **`config/config.go`:** Configuration loading and validation
- **`internal/server.go`:** gRPC server implementation and lifecycle management
- **`internal/handler.go`:** gRPC method implementations and streaming logic
- **`internal/manager.go`:** Core room and player management logic
- **`internal/types.go`:** Type definitions and constants

### Server Integration (server/handlers/)
- **`room.go`:** Room manager service client and event handlers

### Client Implementation (client/model/)
- **`room.go`:** Bubble Tea model for room interface

## Security Considerations

- **Input Validation:** All player names and room operations are validated
- **Rate Limiting:** Should be implemented at the service level for production
- **Connection Security:** Uses gRPC (can be upgraded to TLS)
- **State Synchronization:** Proper locking for concurrent room operations
- **Resource Limits:** Maximum room count prevents resource exhaustion

## Performance Characteristics

- **Memory Usage:** In-memory room state with efficient data structures
- **Concurrency:** Mutex-protected operations for thread safety
- **Streaming Efficiency:** Bidirectional streaming for real-time updates
- **Scalability:** Configurable maximum room limits
- **Cleanup:** Automatic room state management and player removal