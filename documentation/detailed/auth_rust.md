# Rust Auth Service

The Rust auth service is a gRPC service that is responsible for user authentication and account management.

## Key Features

- **User Authentication:** Verifies user credentials and provides authentication tokens.
- **Account Management:** Allows users to create new accounts.
- **Secure Password Storage:** Uses the `argon2` crate to securely hash and store user passwords.

## Architecture

The auth service is built using the `tonic` gRPC framework, which provides a high-performance and scalable foundation for building gRPC services.

### gRPC Services

The auth service exposes two gRPC services:

- **`LoginService`:** Handles user authentication.
- **`CreateService`:** Handles user account creation.

### User Data Storage

User data is stored in a JSONL file, where each line represents a user record. This approach is simple and effective for the current scale of the application.

## Code Structure

The Rust auth service's code is organized as follows:

- **`main.rs`:** The entry point of the service, responsible for initializing the gRPC server and starting the service.
- **`auth.proto`:** The protobuf file that defines the gRPC services and messages.
