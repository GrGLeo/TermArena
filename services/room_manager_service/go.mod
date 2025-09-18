module github.com/GrGLeo/ctf/services/room_manager_service

go 1.23.2

require (
	github.com/GrGLeo/ctf_game/pkg/shared v0.0.0
	github.com/joho/godotenv v1.5.1
	google.golang.org/grpc v1.71.1
)

replace github.com/GrGLeo/ctf/pkg/shared => ../../pkg/shared
