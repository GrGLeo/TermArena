# Networking Protocol

The networking protocol defines how the different components of the CTF game communicate with each other. It is designed to be simple, efficient, and extensible.

## Communication Channels

There are three main communication channels in the system:

- **Client to Go Server:** The client communicates with the Go server to authenticate and find a game room.
- **Go Server to Auth Service:** The Go server communicates with the Auth service to verify user credentials.
- **Client to Rust Game Server:** The client communicates with the Rust game server to play the game.

## Packet Structure

All communication between the client and the servers is done using custom network packets. The `shared` package defines the structure of these packets and provides functions for serializing and deserializing them.

Each packet has a common header that includes the packet type and length. The rest of the packet contains the data that is specific to that packet type.

## gRPC

The Go server communicates with the Rust auth service using gRPC. The `auth.proto` file defines the gRPC services and messages that are used for this communication.

gRPC provides a number of benefits, including:

- **Strongly typed contracts:** The service contract is defined in a `.proto` file, which ensures that the client and server are always in sync.
- **High performance:** gRPC is designed to be fast and efficient, making it ideal for inter-service communication.
- **Language-agnostic:** gRPC supports a wide range of programming languages, making it easy to integrate services that are written in different languages.
