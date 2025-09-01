# Rust Auth Service

The Rust auth service is a gRPC service responsible for user registration and authentication using a secure, SSH-key-style, challenge-response mechanism. It replaces traditional password-based authentication.

## Key Features

- **Passwordless Authentication:** Users are identified by a public/private key pair instead of a password. The private key never leaves the user's machine.
- **Challenge-Response:** The login process involves the server sending a unique, one-time challenge to the client, which the client must sign with its private key to prove its identity.
- **Secure Storage:** The service uses a local SQLite database (`auth.db`) to store user public keys and temporary login challenges.

## Architecture

The service is built using the `tonic` gRPC framework for high-performance communication. It is designed to be a standalone microservice that the main Go server communicates with.

### User Data Storage

User data is stored in a SQLite database named `auth.db`, which contains two main tables:
- **`users`**: Stores a `username` and the corresponding `public_key` (in DER format).
- **`challenges`**: Temporarily stores a unique `challenge` (random bytes) for a specific `username`. Challenges have a short-lived expiration time (5 minutes) and are deleted after a single use to prevent replay attacks.

## gRPC API Definition

The service's API is defined in `proto/auth/auth.proto`.

### `AuthService`

#### `Register`
- **Request:** `RegisterRequest { string username, bytes public_key }`
- **Response:** `RegisterResponse { bool success, string message }`
- **Description:** Allows a new user to register. The client sends a chosen username and its generated public key. The service stores this pair in the `users` table.

#### `GetLoginChallenge`
- **Request:** `GetLoginChallengeRequest { string username }`
- **Response:** `GetLoginChallengeResponse { bytes challenge }`
- **Description:** Initiates the login process. The client requests a challenge for a given username. The service generates 32 random bytes, stores them in the `challenges` table with an expiration time, and sends them to the client.

#### `Authentificate`
- **Request:** `AuthentificateRequest { string username, bytes signed_challenge }`
- **Response:** `AuthentificateResponse { bool success, string message }`
- **Description:** Completes the login process. The client signs the challenge it received (specifically, the SHA256 hash of the challenge) with its private key and sends the resulting signature to the server. The service retrieves the user's public key and the original challenge from the database, verifies the signature, and grants access if it is valid.

## Code Structure

- **`main.rs`:** The entry point of the service. It initializes the gRPC server, sets up the SQLite database, and implements the service logic.
- **`auth.proto`:** The protobuf file that defines the gRPC services and messages.

