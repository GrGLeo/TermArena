# Full Technical Overhaul: Game Logic and Code Implementation

This document outlines a comprehensive technical review and redesign of the game logic and its implementation. It is intended for senior developers or architects planning to overhaul the system. Every major subsystem is explained in detail—from low-level data structures and control flows to high-level concurrency, networking, and state management.

---

## 1. Architectural Overview

The game is built as a Capture-The-Flag (CTF) simulator on a fixed grid, with real-time state updates, concurrent player actions, and networked communications. The architecture is modular, separating concerns into the following layers:

- **Input/Action Processing Layer:**  
  Handles player input and translates these into game actions.
  
- **Game Logic and State Management Layer:**  
  Implements the core rules, including movement, ability execution (dash, freeze), flag handling, and win conditions.  
  - **Player Module:** Contains player attributes, action processing, and cooldown management.
  - **Board Module:** Maintains the grid state, tracks changes (deltas), and performs updates using a dual-grid strategy.
  - **Sprite Module:** Manages temporary visual effects with lifecycle and update routines.

- **Networking Layer:**  
  Manages TCP connections, serialization/deserialization of packets, and broadcasting of state updates.
  
- **Concurrency & Synchronization Layer:**  
  Uses mutexes and atomic operations to protect shared data and coordinate concurrent updates.

---

## 2. Data Structures and Modules

### 2.1 Player Module

**Structure and State:**
- The player entity includes spatial coordinates (X, Y), a facing direction, and a current action indicator.
- Additional attributes include team identification, flag state, and special ability configurations (dash and freeze), each with their cooldown and last-used timestamp.

**Action Processing Flow:**
1. **Receive Action:**  
   The player’s action is set externally (via a network packet) into an enumerated action type.
2. **Execute Action:**  
   The `<code>Player.TakeAction(Board)</code>` method is invoked, which uses a switch-case to delegate:
   - **Movement (Up/Down/Left/Right):**  
     - Validate move with `<code>Board.IsValidPosition(x, y)</code>`.
     - Update player position and flag state (if carrying a flag).
     - Record changes in `<code>ChangeTracker</code>`.
   - **Special Abilities:**  
     - **Dash:**  
       - Validate cooldown and flag restrictions.
       - Compute new position using dash range and adjust for obstacles.
       - Create a series of dash sprite objects along the computed path.
     - **Freeze:**  
       - Validate cooldown.
       - Cast freeze effect into adjacent cells based on current facing.
       - Create freeze sprite objects accordingly.

**Pseudocode Outline:**
```
function TakeAction(board):
    switch action:
        case NoAction:
            return false
        case move*:
            result = Move(board)
            return result
        case spellOne:
            MakeDash(board)
            return false
        case spellTwo:
            MakeFreeze(board)
            return false
```
---

### 2.2 Board Module

**Grid Management:**
- **Dual-Grid Approach:**  
  - **CurrentGrid:** Represents the latest board state.
  - **PastGrid:** Holds the previous state for efficient delta computation.
- **ChangeTracker:**  
  A structure that collects deltas (cell updates) as `<code>Delta{X, Y, Value}</code>` for efficient network broadcasting.

**Initialization:**
- The board is initialized by reading a JSON configuration, which specifies:
  - **Walls:** Fixed obstacles, placed using coordinate ranges.
  - **Flags and Players:** Initial positions and teams.
  
**Update Cycle:**
1. **Action Application:**  
   Each game tick, iterate over players to invoke `<code>TakeAction</code>`.
2. **Sprite Update:**  
   Update all active sprites using their `<code>Update()</code>` methods.
3. **Grid Reconciliation:**  
   - Apply all deltas from the `<code>ChangeTracker</code>` to the `<code>CurrentGrid</code>`.
   - Copy `<code>CurrentGrid</code>` to `<code>PastGrid</code>` for the next tick.

**Optimized Data Transfer:**
- **Run-Length Encoding (RLE):**  
  Compress the grid state into a compact representation for full-board updates when the delta volume exceeds 50% of total cells.
  
**Pseudocode Outline:**
```
function UpdateBoard():
    lock board mutex
    for each delta in ChangeTracker:
        CurrentGrid[delta.Y][delta.X] = delta.Value
        PastGrid = CurrentGrid 
    unlock board mutex
```
<code>
Board.InitBoard(walls, flags, players)
Board.Update()
Board.UpdateSprite()
RunLengthEncode(CurrentGrid)
</code>


---

### 2.3 Sprite Module

**Purpose:**  
Sprites are transient visual effects for dash and freeze abilities. They provide feedback and simulate motion.

**Lifecycle Management:**
- **DashSprite:**  
  - Has a life cycle counter that decrements each tick.
  - Appearance (cell value) changes based on the current life cycle stage.
- **FreezeSprite:**  
  - Adjusts its position based on the casting direction.
  - Incorporates a decaying life cycle, after which it clears its effect.

**Update Flow:**
1. Each sprite’s `<code>Update()</code>` method computes its new position and cell state.
2. The sprite is then added to the board's `<code>ChangeTracker</code>`.
3. The `<code>Clear()</code>` method is checked; if it returns true, the sprite is removed from the active list.

**Pseudocode Outline:**
```
for each sprite in activeSprites:
    (x, y, cell) = sprite.Update()
    ChangeTracker.SaveDelta(x, y, cell)
    if sprite.Clear():
        remove sprite from activeSprites
```

<code>
Sprite interface { Update() (int, int, Cell); Clear() bool }
DashSprite.Update(), DashSprite.Clear()
FreezeSprite.Update(), FreezeSprite.Clear()
</code>

---

### 2.4 Game Room and Network Communication

**Game Room Responsibilities:**
- **Session Management:**  
  Create and manage a game session identified by a unique game ID.
- **Player Connection Handling:**  
  - Accept incoming TCP connections.
  - Spawn a dedicated goroutine per connection to read and deserialize packets.
  - Map network connections to player objects using a lookup table.
  
- **Game Loop Execution:**
  - Utilize a ticker (e.g., 50 ms intervals) to drive game ticks.
  - On each tick:
    - Process pending actions from the `<code>actionChan</code>`.
    - Update the board and sprites.
    - Evaluate win conditions (e.g., team score reaching a threshold).
    - Determine if a full board update or delta update is needed.
    - Broadcast the serialized game state to all connections.
    
- **Concurrency and Safety:**
  - Use mutexes (`sync.Mutex` and `sync.RWMutex`) to protect:
    - Shared board state.
    - Player action maps.
    - Tick counter (via atomic operations).
  
- **Network Protocol:**
  - Packets are serialized using a custom protocol defined in the `<code>shared</code>` package.
  - Types include:
    - **ActionPacket:** Contains a player’s intended action.
    - **BoardPacket:** Contains a full board state.
    - **DeltaPacket:** Contains a list of changes since the last update.
    - **GameStart/GameClose Packets:** Control messages for game session lifecycle.

**Pseudocode Outline:**
```
function StartGame():
    wait until playerConnections
    count == expected number send GameStartPacket to all players while game not finished:
    tickID += 1
    process actions from actionChan:
        update corresponding player
        state board.Update()
        board.UpdateSprite() 
    if win condition reached:
        broadcast GameClosePacket break loop 
    decide update type (full vs. delta) broadcast state packet to players
```
<code>
GameRoom.NewGameRoom()
GameRoom.StartGame()
GameRoom.HandleAction()
GameRoom.ListenToConnection()
</code>

---

## 3. Concurrency and Performance Engineering

### 3.1 Synchronization Strategies

- **Board Mutex:**  
  Protects the board state (both grids and the change tracker) during concurrent reads/writes. Read operations use RLock/RUnlock, while updates use Lock/Unlock.
  
- **Action Channel:**  
  A buffered channel is used to queue player actions. The game loop periodically drains this channel, applying actions in a thread-safe manner.

- **Atomic Counters:**  
  For tick ID and other counters, atomic operations ensure minimal overhead and avoid full mutex locks.

### 3.2 Data Flow Optimization

- **Delta Aggregation:**  
  By recording only changes (deltas) and using RLE for full-board updates, network bandwidth is optimized. The system dynamically selects which update to send based on the percentage of board modifications.
  
- **Sprite Management:**  
  Sprites are lightweight objects; however, their updates are optimized by only processing those with significant visual changes, reducing unnecessary computations.

### 3.3 Network Throughput and Latency

- **Serialization Efficiency:**  
  Custom serialization in the `<code>shared</code>` package is designed for minimal latency. Data structures are compact, and packet formats are fixed-length where possible.
  
- **Connection Handling:**  
  Each TCP connection is serviced in its own goroutine. Disconnection events are handled gracefully to avoid stalling the game loop.

---

## 4. Extensibility and Future-Proofing

### 4.1 Modular Design
- Each module (player, board, sprite, game room) is loosely coupled, making it easier to replace or extend components.
- New abilities can be added by extending the player action enum and implementing corresponding methods without affecting the core loop.

### 4.2 Configuration-Driven Initialization
- The game layout, including walls, flags, and player start positions, is defined via JSON configuration.
- Future enhancements may include dynamic configuration updates and runtime map editing.

### 4.3 Scalability Considerations
- Although the current design is single-instance, the architecture can be extended to multiple game rooms managed by a central orchestrator.
- Potential future work includes adopting WebSockets for improved real-time communication and load balancing across multiple VPS instances.

---

## 5. Summary and Next Steps

This overhaul targets a robust, scalable, and highly maintainable game engine:
- **Player Module:** Detailed action handling with integrated cooldown and flag management.
- **Board Module:** Dual-grid approach with delta tracking and RLE for efficient state updates.
- **Sprite Module:** Lightweight, decaying visual effects with clear lifecycle management.
- **Game Room & Network:** Concurrency-safe game loop with dedicated TCP connection handling and custom packet serialization.

Next steps involve:
- Detailed unit tests for each module.
- Performance benchmarking and network latency optimization.

This document serves as a blueprint for enhancement of the current game logic implementation.

