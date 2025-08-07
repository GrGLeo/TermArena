.PHONY: all build clean run-auth run-game run-server run-client run-simulation test package deploy

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
	@mkdir -p bin
	go build -o bin/server ./server

build-client:
	@echo "Building client..."
	@mkdir -p bin
	go build -o bin/client ./client

package: build
	@echo "Packaging application..."
	@mkdir -p bin
	@cp auth/target/release/auth bin/auth
	@cp game/target/release/game bin/game

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
	rm -rf bin
	cd auth && cargo clean
	cd game && cargo clean

deploy: package
	@echo "Deploying to production..."
	ssh leo@endurace.cloud "mkdir -p /home/leo/bin /home/leo/game/target/debug"
	ssh leo@endurace.cloud "pkill auth || true"
	ssh leo@endurace.cloud "pkill server || true"
	ssh leo@endurace.cloud "pkill game || true"
	scp bin/auth leo@endurace.cloud:/home/leo/bin/
	scp bin/server leo@endurace.cloud:/home/leo/bin/
	scp bin/game leo@endurace.cloud:/home/leo/game/target/debug/
	scp game/spells.toml leo@endurace.cloud:/home/leo/game/
	scp game/items.toml leo@endurace.cloud:/home/leo/game/
	scp game/rules.toml leo@endurace.cloud:/home/leo/game/
	scp game/stats.toml leo@endurace.cloud:/home/leo/game/
