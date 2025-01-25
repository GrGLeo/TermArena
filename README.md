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
  
- **Server**: A TCP server.
  - Manages player states.
  - Processes player actions and updates the board.
  - Uses <b>double-buffered grids</b> for smooth state transitions.
  - Compresses board updates using <b>Run-Length Encoding (RLE)</b> for efficient packet handling.
  - Sends either <b>full board RLE</b> or <b>delta updates</b> to clients, depending on the situation.

## Installation
1. Ensure <a href="https://golang.org/dl/">Go</a> is installed.
2. Clone the repository:
   <code>
   git clone https://github.com/GrGLeo/ctf.git
   cd ctf
   </code>

## Usage
To run the game locally, open two terminals:

1. <b>Run the Server</b>:
   <code>
   make run-server
   </code>
   
2. <b>Run the Client</b>:
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

### Double-Buffered Grid System
- The server uses a <b>double-buffered grid system</b> to manage game state:
  - <b>CurrentGrid</b>: Represents the latest state of the board.
  - <b>PreviousGrid</b>: Stores the previous state of the board.
- Deltas (changes between <code>CurrentGrid</code> and <code>PreviousGrid</code>) are computed and sent to clients for efficient updates.
- This ensures smooth transitions and minimizes network overhead.

### Board Updates
- The server sends updates to clients in one of two ways:
  1. <b>Full Board RLE</b>: The entire board is compressed using Run-Length Encoding (RLE) and sent to the client. This is typically used when a client first joins the game or when the board state changes significantly.
  2. <b>Delta Updates</b>: Only the changes (deltas) between the previous and current board states are sent. This is more efficient for frequent updates during gameplay.

## Server Details
- The server handles multiple player connections.
- Player actions are processed, and the board state is updated using the double-buffered grid system.
- The board is compressed using RLE before sending to clients to optimize network usage.
- The server intelligently decides whether to send a full board RLE or delta updates based on the situation.

## Future Improvements
- Add player abilities (dash & freeze spell).
- Multiplayer room management.
- Improve client-side rendering for smoother animations.
