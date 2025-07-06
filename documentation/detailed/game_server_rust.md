# Rust Game Server

The Rust game server is responsible for running the core game logic in real-time. It is designed to be highly performant and scalable, ensuring a smooth and responsive gaming experience.

## Key Responsibilities

- **Real-time Game Logic:** Manages the game state, including player positions, actions, and game events.
- **Client Communication:** Communicates directly with clients to send game state updates and receive player input.
- **Game Loop:** Runs a high-frequency game loop to ensure that the game state is updated consistently.

## Architecture

The game server is built using the Tokio asynchronous runtime, which allows it to handle a large number of concurrent connections efficiently.

### GameManager

The `GameManager` is the central component of the game server. It is responsible for:

- Managing the game state.
- Tracking player connections.
- Processing player actions.
- Broadcasting game state updates to clients.

### Asynchronous Networking

The server uses asynchronous networking to handle client connections. Each client is assigned its own task, which is responsible for reading data from the client and sending game state updates.

This approach allows the server to handle a large number of clients without blocking the main game loop.

## Code Structure

The Rust game server's code is organized as follows:

- **`main.rs`:** The entry point of the server, responsible for initializing the server and starting the game loop.
- **`game/`:** Contains the core game logic, including the `GameManager` and game state definitions.
- **`packet/`:** Defines the network packets that are used to communicate with clients.
- **`config.rs`:** Handles the loading of game configuration from a TOML file.
