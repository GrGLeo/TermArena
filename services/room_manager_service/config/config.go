package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Host                   string
	Port                   int
	MaxRoom                int
	ShutdownTimeoutSeconds int
}

func NewConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Warning: .env file not found. Using environment variables. %v\n", err)
	}

	host := os.Getenv("MESSAGE_SERVICE_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("MESSAGE_SERVICE_PORT")
	intPort, err := strconv.Atoi(port)
	if err != nil {
		intPort = 8083
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	maxRoom := os.Getenv("ROOM_MANAGER_MAX_ROOM")
	intMaxRoom, err := strconv.Atoi(maxRoom)
	if err != nil {
    intMaxRoom = 25
	}

	shutdownTimeout := os.Getenv("SHUTDOWN_TIMEOUT_SECONDS")
	intShutdownTimeout, err := strconv.Atoi(shutdownTimeout)
	if err != nil {
		intShutdownTimeout = 30
	}

	return &Config{
		Host:                   host,
		Port:                   intPort,
		MaxRoom:                intMaxRoom,
		ShutdownTimeoutSeconds: intShutdownTimeout,
	}
}
