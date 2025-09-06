# CTF Game: System Architecture

This document provides a high-level overview of the CTF game's architecture. The system is composed of several key components that work together to provide a real-time multiplayer experience.

## Core Components

The system is divided into the following main components:

- **[Go Server](./server_go.md):** The central hub that manages client connections, authentication, game room orchestration, and messaging.
- **[Rust Game Server](./game_server_rust.md):** A dedicated server that runs the core game logic in real-time.
- **[Go Client](./client_go.md):** A terminal-based user interface (TUI) that allows players to interact with the game and provides real-time messaging.
- **[Rust Auth Service](./auth_rust.md):** A gRPC service for user authentication and account management.
- **[Message Service](./message_service.md):** A gRPC service for real-time messaging between players.
- **[Rate Limiter](./rate_limiter.md):** Security component that protects against abuse using token bucket algorithm.
- **[Networking Protocol](./networking.md):** Defines the communication protocol between the various components.

## System Diagram

The following diagram illustrates the interaction between the different components:

```mermaid
graph TD
    A[Client] -->|TCP| B(Go Server)
    B <-->|Check| C(Rate Limit)
    B -->|gRPC| D{Auth Service}
    B -->|gRPC| E{Message Service}
    B -->|Spawns| F[Rust Game Server]
    A <-->|TCP| F
```

### Data Flow

1.  **Client to Go Server:** The client initiates a connection to the Go server for authentication, room management, and messaging.
2.  **Rate Limiting Check:** The Go server applies rate limiting before processing requests to prevent abuse.
3.  **Go Server to Auth Service:** The Go server communicates with the Auth service via gRPC to verify user credentials.
4.  **Go Server to Message Service:** The Go server communicates with the Message service via gRPC to handle real-time messaging between clients.
5.  **Go Server Spawns Rust Game Server:** When a game room is created, the Go server spawns a new Rust game server process.
6.  **Client to Rust Game Server:** The client connects directly to the Rust game server to play the game.
7.  **Client to Message Service:** The client connects directly to the Message service for real-time messaging.
8.  **Rust Game Server to Client:** The Rust game server sends real-time game state updates to the client.
9.  **Message Service to Client:** The Message service delivers real-time messages to connected clients.
