build-client:
	go build -o bin/client ./client

build-server:
	go build -o bin/server ./server

run-client:
	go run ./client

run-server:
	go run ./server

run-test:
	go test -v ./...
