# CaptureTheFlag

CaptureTheFlag is a terminal-based multiplayer game where players navigate a board to capture flags. It consists of a client application built with Bubble Tea and a TCP server handling game logic and player actions.

## Table of Contents
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [Game Mechanics](#game-mechanics)
- [Server Details](#server-details)
- [Future Improvements](#future-improvements)

## Features
- **Client**: A terminal-based interface built with Bubble Tea.
  - Login page (without validation).
  - A 20x50 board updated every 50ms.
  - Player movement: up(w), down(s), left(a), right(d).
  - Dash ability: Press <code>space</code> to dash in the facing direction.
  
- **Server**: A TCP server.
  - Manages player states.
  - Processes player actions and updates the board.
  - Uses **double-buffered grids** for smooth state transitions.
  - Compresses board updates using **Run-Length Encoding (RLE)** for efficient packet handling.
  - Sends either **full board RLE** or **delta updates** to clients, depending on the situation.

## Installation
1. Ensure <a href="https://golang.org/dl/">Go</a> is installed.
2. Clone the repository:
   <code>
   git clone https://github.com/GrGLeo/ctf.git
   cd ctf
   </code>

## Usage
To run the game locally, open two terminals:

1. **Run the Server**:
   <code>
   make run-server
   </code>
   
2. **Run the Client**:
   <code>
   make run-client
   </code>

## Game Mechanics
- Players move around a 20x50 board to capture the enemy flag.
- Walls are placed on the board. The configuration of walls is done through a <code>config.json</code> file:
   <code>
   {
       "walls": [
           {
               "StartPos": [5, 10],
               "EndPos": [5, 15]
           }
       ]
   }
   </code>
- Player collisions are checked against walls, and walls can't be traversed.
- The board refreshes every 50ms to reflect the latest player positions.
- Two flags are placed on each team's side. The player needs to capture the enemy flag and bring it back to their base.

### Dash Mechanism
- Players can use the **dash ability** by pressing the <code>space</code> key.
- Dash allows players to move quickly in the direction they are facing, bypassing walls and obstacles.
- The dash range and cooldown are configurable in the player's <code>Dash</code> struct:
   <code>
   type Dash struct {
       Range    int `json:"range"`
       Cooldown int `json:"cooldown"`
       LastUsed time.Time `json:"-"`
   }
   </code>
- During a dash, a visual trail of sprites is generated to indicate the player's movement path.

### Double-Buffered Grid System
- The server uses a **double-buffered grid system** to manage game state:
  - **CurrentGrid**: Represents the latest state of the board.
  - **PreviousGrid**: Stores the previous state of the board.
- Deltas (changes between <code>CurrentGrid</code> and <code>PreviousGrid</code>) are computed and sent to clients for efficient updates.
- This ensures smooth transitions and minimizes network overhead.

### Board Updates
- The server sends updates to clients in one of two ways:
  1. **Full Board RLE**: The entire board is compressed using Run-Length Encoding (RLE) and sent to the client. This is typically used when a client first joins the game or when the board state changes significantly.
  2. **Delta Updates**: Only the changes (deltas) between the previous and current board states are sent. This is more efficient for frequent updates during gameplay.

## Server Details
- The server handles multiple player connections.
- Player actions are processed, and the board state is updated using the double-buffered grid system.
- The board is compressed using RLE before sending to clients to optimize network usage.
- The server intelligently decides whether to send a full board RLE or delta updates based on the situation.

## Future Improvements
- Add player abilities (freeze spell).
- Multiplayer room management.
- Improve client-side rendering for smoother animations.
