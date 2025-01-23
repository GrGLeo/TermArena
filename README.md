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
  - A 20x50 board updated every 100ms.
  - Player movement: up(w), down(s), left(a), right(d).
  
- **Server**: A TCP server.
  - Manages player states.
  - Processes player actions and updates the board.
  - Uses Run-Length Encoding (RLE) for efficient packet handling.

## Installation
1. Ensure [Go](https://golang.org/dl/) is installed.
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
- Walls are placed on the board.  The configuration of walls are done through a config.json file
```json
{
    "walls": [
        {
            "StartPos": [5, 10],
            "EndPos": [5, 15]
        }
    ]
}
```
- Player collision are checked against walls, and walls can't be traversed.
- The board refreshes every 50ms to reflect the latest player positions.


## Server Details
- The server handles multiple player connections.
- Player actions are processed, and the board state is updated.
- The board is compressed using RLE before sending to clients to optimize network usage.

## Future Improvements
- Add the flag, and capturing mechanism. 
- Add player abilities (dash & freeze spell).
- Multiplayer room management. 
