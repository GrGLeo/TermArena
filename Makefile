.PHONY: all build build-auth build-game build-server build-message build-client clean run-auth run-message run-game run-server run-client run-simulation test package deploy

all: build

build: build-auth build-game build-server build-message build-client

build-auth:
	@echo "Building auth service..."
	@rm bin/auth
	cd services/auth && cargo build --release && mv target/release/auth ../../bin/auth

build-game:
	@echo "Building game engine..."
	@rm bin/game
	cd services/game && cargo build --release && mv target/release/game ../../bin/game

build-server:
	@echo "Building server..."
	@mkdir -p bin
	cd server && go build -o ../bin/server .

build-message:
	@echo "Building message service..."
	@mkdir -p bin
	cd services/message_service && go build -o ../../bin/message_service .

build-client:
	@echo "Building client..."
	@mkdir -p bin
	cd client && go build -o ../bin/client .

package: build
	@echo "Packaging application..."
	@mkdir -p bin
	@cp services/auth/target/release/auth bin/auth
	@cp services/game/target/release/game bin/game

run-auth:
	@echo "Running auth service..."
	./bin/auth

run-message:
	@echo "Running message service..."
	./bin/message_service

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

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning up build artifacts..."
	rm -rf bin
	cd services/auth && cargo clean
	cd services/game && cargo clean

deploy: package
	@echo "Deploying to production..."
	ssh leo@endurace.cloud "mkdir -p /home/leo/bin /home/leo/game/target/debug"
	ssh leo@endurace.cloud "pkill auth || true"
	ssh leo@endurace.cloud "pkill server || true"
	ssh leo@endurace.cloud "pkill game || true"
	ssh leo@endurace.cloud "pkill message_service || true"
	scp bin/auth leo@endurace.cloud:/home/leo/bin/
	scp bin/server leo@endurace.cloud:/home/leo/bin/
	scp bin/message_service leo@endurace.cloud:/home/leo/bin/
	scp bin/game leo@endurace.cloud:/home/leo/game/target/debug/
	scp services/game/spells.toml leo@endurace.cloud:/home/leo/game/
	scp services/game/items.toml leo@endurace.cloud:/home/leo/game/
	scp services/game/rules.toml leo@endurace.cloud:/home/leo/game/
	scp services/game/stats.toml leo@endurace.cloud:/home/leo/game/
