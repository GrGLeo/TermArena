package main

import "fmt"

type ConnectionError struct {
	Code    int
	Message string
}

func (c *ConnectionError) Error() string {
	return fmt.Sprintf("Error %d: %s", c.Code, c.Message)
}

func NewConnectionError(code int, message string) *ConnectionError {
	return &ConnectionError{
		Code:    code,
		Message: message,
	}
}
