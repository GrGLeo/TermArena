#!/bin/bash

# Load environment variables
if [ -f .env ]; then
  export $(cat .env | sed 's/#.*//g' | xargs)
fi

# Clean up previous logs and processes
echo "Cleaning up previous game logs..."
rm -f rust_game_*.log
rm -f client/debug.log
rm -f auth.log
rm -f server.log
rm -f message_service.log

echo "Attempting to clear ports 50051, 8082, 8083..."
sudo fuser -k 50051/tcp >/dev/null 2>&1 || true
sudo fuser -k 8082/tcp >/dev/null 2>&1 || true
sudo fuser -k 8083/tcp >/dev/null 2>&1 || true

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

# Build the message service
echo "Building message service..."
(cd services/message_service && go build -o message_service_bin) || { echo "Failed to build message service"; exit 1; }

# Build the server binary
echo "Building server service..."
(cd server && go build -o server_bin main.go) || { echo "Failed to build server service"; exit 1; }

# Kill existing auth, message service, and server processes
kill_process "./auth/target/debug/auth"
kill_process "./services/message_service/message_service_bin"
kill_process "./server/server_bin"

# Start the auth, message service, and server services in the background, redirecting output to files
echo "Starting auth service. Output redirected to auth.log"
./auth/target/debug/auth > auth.log 2>&1 &
AUTH_PID=$!

echo "Starting message service. Output redirected to message_service.log"
./services/message_service/message_service_bin > message_service.log 2>&1 &
MESSAGE_PID=$!

echo "Starting server service. Output redirected to server.log"
./server/server_bin > server.log 2>&1 &
SERVER_PID=$!

# Wait a moment for services to start up
sleep 5

# Check if services are running
echo "Checking service status..."
if kill -0 $AUTH_PID 2>/dev/null; then
    echo "✅ Auth service is running (PID: $AUTH_PID)"
else
    echo "❌ Auth service failed to start"
fi

if kill -0 $MESSAGE_PID 2>/dev/null; then
    echo "✅ Message service is running (PID: $MESSAGE_PID)"
else
    echo "❌ Message service failed to start"
fi

if kill -0 $SERVER_PID 2>/dev/null; then
    echo "✅ Server service is running (PID: $SERVER_PID)"
else
    echo "❌ Server service failed to start"
fi

# Start the client
echo "Starting client..."
(cd client && go run main.go)

# Clean up background processes when client exits (or script is manually terminated)
echo "Client exited. Killing background services..."
kill -TERM $AUTH_PID $MESSAGE_PID $SERVER_PID 2>/dev/null
