#!/bin/bash

# Load environment variables
if [ -f .env ]; then
  export $(cat .env | sed 's/#.*//g' | xargs)
fi

# Set server IP based on environment
if [ "$APP_ENV" = "prd" ]; then
  SERVER_IP="endurace.cloud"
else
  SERVER_IP="localhost"
fi

# Function to kill processes by name
kill_process() {
  local name=$1
  echo "Checking for existing $name processes..."
  # Use pgrep -f to search for the full command line
  # Exclude the current script's PID from the search to prevent self-termination
  PIDS=$(pgrep -f "$name" | grep -v "^$$") # $$ is current script PID
  if [ -n "$PIDS" ]; then
    echo "Found existing $name processes (PIDs: $PIDS). Killing them..."
    kill -TERM $PIDS
    sleep 2 # Give processes a moment to terminate
    # Check if they are still running
    PIDS_AFTER_KILL=$(pgrep -f "$name" | grep -v "^$$")
    if [ -n "$PIDS" ]; then
      echo "$name processes still running (PIDs: $PIDS_AFTER_KILL). Forcibly killing them..."
      kill -KILL $PIDS_AFTER_KILL
    fi
  else
    echo "No existing $name processes found."
  fi
}

# Build the game and auth services
echo "Building game service..."
(cd game && cargo build) || { echo "Failed to build game service"; exit 1; }
echo "Building auth service..."
(cd auth && cargo build) || { echo "Failed to build auth service"; exit 1; }

# Build the server binary
echo "Building server service..."
(cd server && go build -o server_bin main.go) || { echo "Failed to build server service"; exit 1; }

# Kill existing auth and server processes
kill_process "./auth/target/debug/auth"
kill_process "./server/server_bin"

# Start the auth and server services in the background, redirecting output to files
echo "Starting auth service. Output redirected to auth.log"
./auth/target/debug/auth > auth.log 2>&1 &
AUTH_PID=$!

echo "Starting server service. Output redirected to server.log"
./server/server_bin > server.log 2>&1 &
SERVER_PID=$!

# Wait a moment for services to start up
sleep 3

# Start the client
echo "Starting client..."
(cd client && go run main.go)

# Clean up background processes when client exits (or script is manually terminated)
echo "Client exited. Killing background services..."
kill -TERM $AUTH_PID $SERVER_PID 2>/dev/null