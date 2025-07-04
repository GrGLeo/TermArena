# CTF Game (MOBA Style)

This project has evolved from a Capture The Flag game to a MOBA (Multiplayer Online Battle Arena) style game, with a significant rewrite of the core game logic in Rust. It features a client, server, authentication service, and the core game engine.

## Table of Contents
- [Project Overview](#project-overview)
- [Components](#components)
- [Installation](#installation)
- [Usage](#usage)
- [Game Mechanics](#game-mechanics)
- [Future Improvements](#future-improvements)

## Project Overview

This project is a real-time multiplayer online battle arena (MOBA) game. Players control champions on a board, engaging in strategic combat, managing minions, and destroying enemy towers and bases. The game emphasizes real-time action, strategic decision-making, and team-based objectives.

## Components

The project is composed of several distinct services:

- **`game/` (Rust)**: The core game engine, responsible for managing game state, board logic, entity management (champions, minions, towers), collision detection, pathfinding, and animations. This is where the MOBA game mechanics are primarily implemented.
- **`server/` (Go)**: The main game server, handling client connections, matchmaking, room management, and relaying game state updates between the `game/` engine and clients. It also processes player actions and forwards them to the `game/` engine.
- **`client/` (Go)**: The game client, providing the user interface, handling player input, and rendering the game state received from the `server/`.
- **`auth/` (Rust)**: The authentication service, responsible for user authentication and authorization.

## Installation

1.  **Prerequisites**:
    *   Go (for `server/` and `client/`)
    *   Rust (for `game/` and `auth/`)
    *   `make`

2.  **Clone the repository**:
    ```bash
    git clone https://github.com/GrGLeo/ctf.git
    cd ctf
    ```

3.  **Build the project**:
    ```bash
    make build
    ```
    This command will build all necessary components (server, client, game engine, auth service).

## Usage

To run the game locally, open separate terminal windows for each component:

1.  **Run the Authentication Service**:
    ```bash
    make run-auth
    ```

2.  **Run the Game Engine**:
    ```bash
    make run-game
    ```

3.  **Run the Game Server**:
    ```bash
    make run-server
    ```

4.  **Run the Client**:
    ```bash
    make run-client
    ```

## Game Mechanics

Players select champions and engage in real-time combat on a dynamic board. The core game logic, including board management, entity interactions, and animations, is handled by the Rust `game/` engine. Key mechanics include:

-   **Board System**: The game takes place on a grid-based board (`board.rs`, `cell.rs`) where entities interact.
-   **Champions**: Players control unique champions (`champion.rs`) with distinct abilities.
-   **Minions**: AI-controlled minion units (`minion.rs`) are spawned and managed by a `minion_manager.rs` to push lanes and assist champions.
-   **Towers**: Defensive tower structures (`tower.rs`) are strategically placed on the map to protect bases and attack enemy units.
-   **Base Destruction**: The primary objective is to destroy the enemy's main base.
-   **Pathfinding**: Entities on the board utilize advanced pathfinding algorithms (`pathfinding.rs`) to navigate obstacles and reach their targets.
-   **Animations**: The game includes various animations for actions such as melee attacks (`melee.rs`) and tower attacks (`tower.rs`), providing visual feedback.
-   **Packet Communication**: Communication between the game engine, server, and client is facilitated by various packet types, including `action_packet.rs` for player actions, `board_packet.rs` for board state updates, `start_packet.rs` for game initialization, and `end_game_packet.rs` for game conclusion.

## Future Improvements

-   Detailed champion abilities and balance.
-   More sophisticated AI for minions and other entities.
-   Enhanced visual effects and animations.
-   Improved networking and latency compensation.
-   Comprehensive in-game tutorial and user onboarding.
