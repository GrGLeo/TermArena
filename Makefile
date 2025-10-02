.PHONY: all build build-auth build-game build-server build-message build-room-manager build-client clean run-auth run-message run-game run-server run-room-manager run-client run-simulation test package deploy

all: build

build: build-auth build-game build-server build-message build-room-manager build-client

build-auth:
	@echo "Building auth service..."
	@mkdir -p bin
	@rm -f bin/auth
	cd services/auth && cargo build --release && mv target/release/auth ../../bin/auth

build-game:
	@echo "Building game engine..."
	@mkdir -p bin
	@rm -f bin/game
	cd services/game && cargo build --release && mv target/release/game ../../bin/game

build-server:
	@echo "Building server..."
	@mkdir -p bin
	cd server && go build -o ../bin/server .

build-message:
	@echo "Building message service..."
	@mkdir -p bin
	cd services/message_service && go build -o ../../bin/message_service .

build-room-manager:
	@echo "Building room manager service..."
	@mkdir -p bin
	cd services/room_manager_service/cmd && go build -o ../../../bin/room_service .

build-client:
	@echo "Building client..."
	@mkdir -p bin
	cd client && go build -o ../bin/client .

build-all-client:
	@echo "Building client statically..."
	@mkdir -p bin
	cd client && CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o ../bin/client .

package: build
	@echo "Packaging application..."
	@mkdir -p bin

run-auth:
	@echo "Running auth service..."
	./bin/auth

run-message:
	@echo "Running message service..."
	./bin/message_service

run-room-manager:
	@echo "Running room manager service..."
	./bin/room_service

run-game:
	@echo "Running game engine..."
	./bin/game

run-server:
	@echo "Running server..."
	./bin/server

run-client:
	@echo "Running client..."
	./bin/client

run-simulation:
	@echo "Running simulation..."
	cd test/e2e && go run game_simulation.go

run-e2e:
	@echo "Starting services..."
	./bin/auth >> auth.log 2>&1 &
	./bin/server >> server.log 2>&1 &
	./bin/room_service >> room.log 2>&1 &
	./bin/message_service >> message.log 2>&1 &
	@sleep 2
	cd test/e2e && go run main.go
	@echo "Stopping services..."
	@pkill auth || true
	@pkill server || true
	@pkill room_service || true
	@pkill message_service || true

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning up build artifacts..."
	rm -rf bin
	cd services/auth && cargo clean
	cd services/game && cargo clean
