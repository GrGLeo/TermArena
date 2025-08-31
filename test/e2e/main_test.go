package main

import (
	"testing"
)

// TestCleanupClients tests the cleanup functionality
func TestCleanupClients(t *testing.T) {
	// This is a basic test to ensure the cleanup function doesn't panic
	clients := []*TestClient{
		{username: "test1"},
		{username: "test2"},
	}

	// Should not panic even with nil connections
	cleanupClients(clients)

	// Test with empty slice
	emptyClients := []*TestClient{}
	cleanupClients(emptyClients)
}

// TestBasicFunctionality tests that basic functions work
func TestBasicFunctionality(t *testing.T) {
	// Test that we can create a test client struct
	client := &TestClient{
		username: "testuser",
	}

	if client.username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", client.username)
	}
}
