module github.com/GrGLeo/ctf/services/message_service

go 1.23.2

require (
	github.com/GrGLeo/ctf_game/pkg/shared v0.0.0
	github.com/joho/godotenv v1.5.1
	google.golang.org/grpc v1.71.1
)

require (
	github.com/GrGLeo/ctf v0.0.0-20250825162123-973788692dfd // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
	google.golang.org/protobuf v1.36.4 // indirect
)

replace github.com/GrGLeo/ctf_game/pkg/shared => ../../pkg/shared
