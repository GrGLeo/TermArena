.PHONY: all build clean run-auth run-game run-server run-client run-simulation test

all: build

build: build-auth build-game build-server build-client

build-auth:
	@echo "Building auth service..."
	cd auth && cargo build --release

build-game:
	@echo "Building game engine..."
	cd game && cargo build --release

build-server:
	@echo "Building server..."
	go build -o bin/server ./server

build-client:
	@echo "Building client..."
	go build -o bin/client ./client

run-auth:
	@echo "Running auth service..."
	./auth/target/release/auth

run-game:
	@echo "Running game engine..."
	./game/target/release/game

run-server:
	@echo "Running server..."
	./bin/server

run-client:
	@echo "Running client..."
	./bin/client

run-simulation:
	@echo "Running simulation..."
	go run ./simulation

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning up build artifacts..."
	rm -rf bin/server bin/client
	cd auth && cargo clean
	cd game && cargo clean