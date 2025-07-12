# Networking Protocol

The networking protocol defines how the different components of the CTF game communicate with each other. It is designed to be simple, efficient, and extensible.

## Communication Channels

There are three main communication channels in the system:

- **Client to Go Server:** The client communicates with the Go server to authenticate and find a game room.
- **Go Server to Auth Service:** The Go server communicates with the Auth service to verify user credentials.
- **Client to Rust Game Server:** The client communicates with the Rust game server to play the game.

## Packet Structure

All communication between the client and the servers is done using custom network packets.

### Common Packet Header

All packets share a common header for versioning and type identification.

```
Byte Offset: 0       1
             +-------+-------+
             |Version| Code  |
             +-------+-------+
Size (bytes):  1       1
```

*   **Version (u8):** Protocol version (currently `1`).
*   **Code (u8):** Packet type identifier.

### Go Server/Client Packets (`shared/packet.go`)

These packets are primarily used for communication between the Go client and the Go server (authentication, room management).

#### LoginPacket (Code 0)

Used by the client to send login credentials to the authentication server.

```
Byte Offset: 0       1       2       3       4         X       X+1     X+2    
             +-------+-------+-------+-------+---------+-------+-------+----------------
             |Version| Code  |  Username Len | Username        |  Password Len | Password ...
             +-------+-------+-------+-------+---------+-------+-------+----------------
Size (bytes):  1       1       2               (variable)        2       (variable)
```

*   **Username Len (u16):** Length of the Username string in bytes.
*   **Username (string):** The user's username.
*   **Password Len (u16):** Length of the Password string in bytes.
*   **Password (string):** The user's password.

#### SignInPacket (Code 1)

Used by the client to send new user registration credentials. Structure is identical to `LoginPacket`.

#### RespPacket (Code 2)

Used by the authentication server to respond to login/signin requests.

```
Byte Offset: 0       1       2
             +-------+-------+--------+
             |Version| Code  | Success|
             +-------+-------+--------+
Size (bytes):  1       1       1
```

*   **Success (u8):** `1` for success, `0` for failure.

#### RoomRequestPacket (Code 3)

Used by the client to request a room (e.g., find a public game).

```
Byte Offset: 0       1       2
             +-------+-------+--------+
             |Version| Code  |RoomType|
             +-------+-------+--------+
Size (bytes):  1       1       1
```

*   **RoomType (u8):** Type of room requested.

#### RoomCreatePacket (Code 4)

Used by the client to request the creation of a new room. Structure is identical to `RoomRequestPacket`.

#### RoomJoinPacket (Code 5)

Used by the client to request joining a specific room by ID.

```
Byte Offset: 0       1       2    
             +-------+-------+------
             |Version| Code  | RoomID ...
             +-------+-------+------
Size (bytes):  1       1       (variable, reads to end of packet)
```

*   **RoomID (string):** The ID of the room to join.

#### LookRoomPacket (Code 6)

Used by the server to respond to room search requests.

```
Byte Offset: 0       1       2        3       4       5       6       7       8    
             +-------+-------+--------+-------+-------+-------+-------+-----+------
             |Version| Code  | Success|       RoomID (fixed 5 bytes)        | RoomIP
             +-------+-------+--------+-------+-------+-------+-------+-----+------
Size (bytes):  1       1       1       5                                    (variable, reads to end of packet)
```

*   **Success (u8):** `1` for success, `0` for failure.
*   **RoomID (string):** The ID of the found room (fixed 5 bytes, padded with spaces if shorter).
*   **RoomIP (string):** The IP address of the game server hosting the room.

#### GameStartPacket (Code 7)

Used by the server to signal the start of a game.

```
Byte Offset: 0       1       2
             +-------+-------+-------+
             |Version| Code  | Success|
             +-------+-------+-------+
Size (bytes):  1       1       1
```

*   **Success (u8):** `1` for success, `0` for failure.

#### DeltaPacket (Code 10)

Used by the game server to send incremental updates (deltas) to the client.

```
Byte Offset: 0       1       2       3       4       5         6         7       8       9       10    
             +-------+-------+-------+-------+-------+---------+---------+-------+-------+-------+------
             |Version| Code  |        TickID         |Points[0]|Points[1]| Delta Count   | Deltas (3 bytes each)
             +-------+-------+-------+-------+-------+---------+---------+-------+-------+-------+------
Size (bytes):  1       1       4                       1        1         2                (variable)
```

*   **TickID (u32):** The game tick this delta corresponds to.
*   **Points[0] (u8):** Score for Team 0.
*   **Points[1] (u8):** Score for Team 1.
*   **Delta Count (u16):** Number of individual deltas in the `Deltas` field.
*   **Deltas (Vec<[u8; 3]>):** A list of 3-byte deltas, where each delta represents a change on the board.

#### GameClosePacket (Code 11)

Used by the server to signal the game is closing.

```
Byte Offset: 0       1       2
             +-------+-------+--------+
             |Version| Code  | Success|
             +-------+-------+--------+
Size (bytes):  1       1       1
```

*   **Success (u8):** `0` for won, `1` for lost, `2` for error.

#### EndGamePacket (Code 12)

Used by the server to signal the end of a game and declare the winner.

```
Byte Offset: 0       1       2
             +-------+-------+--------+
             |Version| Code  |   Win  |
             +-------+-------+--------+
Size (bytes):  1       1       1
```

*   **Win (u8):** `1` if the client's team won, `0` if lost.

### Rust Game Server/Client Packets (`game/src/packet/`)

These packets are used for communication between the Go client and the Rust game server. Note that some `Code` values are reused with different structures compared to the Go server/client packets.

#### ActionPacket (Code 8)

Used by the client to send player actions (e.g., movement, spell cast) to the game server.

```
Byte Offset: 0       1       2
             +-------+-------+-------+
             |Version| Code  | Action|
             +-------+-------+-------+
Size (bytes):  1       1       1
```

*   **Action (u8):** The specific action being performed (e.g., `1` for MoveUp, `2` for MoveDown).

#### BoardPacket (Code 9)

Used by the game server to send the player's view of the game board and their champion's status.

```
Byte Offset: 0       1       2       3       4       5       6       7       8       9       10      11      12      13      14      15      16      17      18      19    
             +-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+------
             |Version| Code  |   Points      |    Health     |  Max Health   | Level |       XP      |     XP Needed |   Length      | Encoded Board Data ...
             +-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+-------+------
Size (bytes):  1       1       2               2               2               1       4               4               2               (variable)
```

*   **Points (u16):** Player's score or points (currently `0`).
*   **Health (u16):** Current health of the player's champion.
*   **Max Health (u16):** Maximum health of the player's champion.
*   **Level (u8):** Current level of the player's champion.
*   **XP (u32):** Current experience points of the player's champion.
*   **XP Needed (u32):** Experience points needed for the next level.
*   **Length (u16):** Length of the `Encoded Board Data` in bytes.
*   **Encoded Board Data (Vec<u8>):** Run-length encoded representation of the game board visible to the player.

#### StartPacket (Code 7)

Used by the game server to confirm a successful connection and game start. Structure is identical to the Go `GameStartPacket`.

#### EndGamePacket (Code 12)

Used by the game server to signal the end of a game and declare the winner.

```
Byte Offset: 0       1       2
             +-------+-------+-------+
             |Version| Code  | Winner|
             +-------+-------+-------+
Size (bytes):  1       1       1
```

*   **Winner (u8):** `0` for Red Team, `1` for Blue Team.



## gRPC

The Go server communicates with the Rust auth service using gRPC. The `auth.proto` file defines the gRPC services and messages that are used for this communication.

gRPC provides a number of benefits, including:

- **Strongly typed contracts:** The service contract is defined in a `.proto` file, which ensures that the client and server are always in sync.
- **High performance:** gRPC is designed to be fast and efficient, making it ideal for inter-service communication.
- **Language-agnostic:** gRPC supports a wide range of programming languages, making it easy to integrate services that are written in different languages.
