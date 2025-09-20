package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Host                   string
	Port                   int
	MaxMessageSize         int
	MessageRateLimit       int
	MessageTTLSecond       int
	LogLevel               string
	WorkerPoolSize         int
	MaxQueueSize           int
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

	maxMessageSize := os.Getenv("MAX_MESSAGE_SIZE")
	intMaxMessageSize, err := strconv.Atoi(maxMessageSize)
	if err != nil {
		intMaxMessageSize = 1024
	}

	messageRateLimit := os.Getenv("MESSAGE_RATE_LIMIT")
	intMessageRateLimit, err := strconv.Atoi(messageRateLimit)
	if err != nil {
		intMessageRateLimit = 100
	}

	messageTTLSeconds := os.Getenv("MESSAGE_TTL_SECONDS")
	intMessageTTLSeconds, err := strconv.Atoi(messageTTLSeconds)
	if err != nil {
		intMessageTTLSeconds = 3600
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "debug"
	}

	workerPoolSize := os.Getenv("WORKER_POOL_SIZE")
	intWorkerPoolSize, err := strconv.Atoi(workerPoolSize)
	if err != nil {
		intWorkerPoolSize = 10
	}

	messageQueueSize := os.Getenv("MESSAGE_QUEUE_SIZE")
	intMessageQueueSize, err := strconv.Atoi(messageQueueSize)
	if err != nil {
		intMessageQueueSize = 1000
	}

	shutdownTimeout := os.Getenv("SHUTDOWN_TIMEOUT_SECONDS")
	intShutdownTimeout, err := strconv.Atoi(shutdownTimeout)
	if err != nil {
		intShutdownTimeout = 30
	}

	return &Config{
		Host:                   host,
		Port:                   intPort,
		MaxMessageSize:         intMaxMessageSize,
		MessageRateLimit:       intMessageRateLimit,
		MessageTTLSecond:       intMessageTTLSeconds,
		LogLevel:               logLevel,
		WorkerPoolSize:         intWorkerPoolSize,
		MaxQueueSize:           intMessageQueueSize,
		ShutdownTimeoutSeconds: intShutdownTimeout,
	}
}
