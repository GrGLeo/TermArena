build-client:
	go build -o bin/client ./client

build-server:
	go build -o bin/server ./server

run-client:
	rm debug.log
	go run ./client

run-server:
	go run ./server

test:
	go test -v ./...
