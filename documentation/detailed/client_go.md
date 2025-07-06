# Go Client

The Go client provides a terminal-based user interface (TUI) for the CTF game. It is built using the Bubble Tea framework, which allows for the creation of rich and interactive TUI applications.

## Key Features

- **TUI Interface:** Provides a clean and intuitive interface for players to interact with the game.
- **Real-time Updates:** Receives real-time game state updates from the server and renders them in the TUI.
- **State Management:** Manages the client-side game state, including the player's current status and game progress.

## Architecture

The client is built using the Model-View-Update (MVU) architecture, which is a popular pattern for building interactive applications.

### Model

The `MetaModel` struct represents the application's state. It contains all the information that is needed to render the UI and handle user input.

### View

The `View` function is responsible for rendering the UI based on the current state of the model. It returns a string that is then displayed in the terminal.

### Update

The `Update` function is responsible for handling events, such as user input and messages from the server. It updates the model accordingly and returns a new model.

## Code Structure

The Go client's code is organized as follows:

- **`main.go`:** The entry point of the client, responsible for initializing the application and starting the event loop.
- **`model/`:** Contains the different models that are used to represent the application's state.
- **`communication/`:** Handles communication with the server, including sending and receiving network packets.
