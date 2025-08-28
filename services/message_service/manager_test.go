package main

import (
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
)

// Test constants
const (
	testClient1 = "client1"
	testClient2 = "client2"
	testClient3 = "client3"
	testRoom1   = 1
	testRoom2   = 2
	testMaxSize = 1000
)

func createTestManager() *MessageManager {
	// Create a logger that discards output for testing
	logger := slog.New(slog.NewTextHandler(&nullWriter{}, nil))
	return NewMessageManager(testMaxSize, logger)
}

// nullWriter discards all writes - used for testing
type nullWriter struct{}

func (w *nullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

var _ io.Writer = (*nullWriter)(nil)

func TestNewMessageManager(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, nil))
	manager := NewMessageManager(testMaxSize, logger)

	if manager == nil {
		t.Fatal("NewMessageManager returned nil")
	}
	if manager.userToRoom == nil {
		t.Error("userToRoom map not initialized")
	}
	if manager.roomToClient == nil {
		t.Error("roomToClient map not initialized")
	}
	if manager.logger == nil {
		t.Error("logger not set")
	}
	if manager.maxMessageSize != testMaxSize {
		t.Errorf("maxMessageSize = %d, want %d", manager.maxMessageSize, testMaxSize)
	}
}

func TestRegisterClient(t *testing.T) {
	tests := []struct {
		name         string
		client       string
		roomID       int
		setupFunc    func(*MessageManager)
		wantErr      bool
		errContains  string
		validateFunc func(*testing.T, *MessageManager)
	}{
		{
			name:    "successful registration",
			client:  testClient1,
			roomID:  testRoom1,
			wantErr: false,
			validateFunc: func(t *testing.T, m *MessageManager) {
				if roomID, exists := m.userToRoom[testClient1]; !exists || roomID != testRoom1 {
					t.Errorf("client %s not registered in room %d", testClient1, testRoom1)
				}
				if clients, exists := m.roomToClient[testRoom1]; !exists || clients == nil {
					t.Error("room not created")
				} else if _, exists := clients[testClient1]; !exists {
					t.Errorf("client %s not added to room %d", testClient1, testRoom1)
				}
			},
		},
		{
			name:   "duplicate registration same room",
			client: testClient1,
			roomID: testRoom1,
			setupFunc: func(m *MessageManager) {
				m.RegisterClient(testClient1, testRoom1)
			},
			wantErr:     true,
			errContains: "already in the room",
		},
		{
			name:   "room switching",
			client: testClient1,
			roomID: testRoom2,
			setupFunc: func(m *MessageManager) {
				m.RegisterClient(testClient1, testRoom1)
			},
			wantErr: false,
			validateFunc: func(t *testing.T, m *MessageManager) {
				// Should be in new room
				if roomID, exists := m.userToRoom[testClient1]; !exists || roomID != testRoom2 {
					t.Errorf("client %s not in room %d", testClient1, testRoom2)
				}
				// Should not be in old room
				if clients, exists := m.roomToClient[testRoom1]; exists {
					if _, exists := clients[testClient1]; exists {
						t.Errorf("client %s still in old room %d", testClient1, testRoom1)
					}
				}
				// Should be in new room
				if clients, exists := m.roomToClient[testRoom2]; !exists || clients == nil {
					t.Error("new room not created")
				} else if _, exists := clients[testClient1]; !exists {
					t.Errorf("client %s not in new room %d", testClient1, testRoom2)
				}
			},
		},
		{
			name:        "empty client ID",
			client:      "",
			roomID:      testRoom1,
			wantErr:     true,
			errContains: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := createTestManager()

			if tt.setupFunc != nil {
				tt.setupFunc(manager)
			}

			err := manager.RegisterClient(tt.client, tt.roomID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, manager)
			}
		})
	}
}

func TestUnregisterClient(t *testing.T) {
	tests := []struct {
		name         string
		client       string
		setupFunc    func(*MessageManager)
		wantErr      bool
		errContains  string
		validateFunc func(*testing.T, *MessageManager)
	}{
		{
			name:   "successful unregistration",
			client: testClient1,
			setupFunc: func(m *MessageManager) {
				m.RegisterClient(testClient1, testRoom1)
			},
			wantErr: false,
			validateFunc: func(t *testing.T, m *MessageManager) {
				if _, exists := m.userToRoom[testClient1]; exists {
					t.Errorf("client %s still in userToRoom map", testClient1)
				}
				if clients, exists := m.roomToClient[testRoom1]; exists {
					if _, exists := clients[testClient1]; exists {
						t.Errorf("client %s still in room %d", testClient1, testRoom1)
					}
				}
			},
		},
		{
			name:        "unregister non-existent client",
			client:      testClient1,
			wantErr:     true,
			errContains: "Failed to find client",
		},
		{
			name:   "room cleanup after last client leaves",
			client: testClient1,
			setupFunc: func(m *MessageManager) {
				m.RegisterClient(testClient1, testRoom1)
			},
			wantErr: false,
			validateFunc: func(t *testing.T, m *MessageManager) {
				if _, exists := m.roomToClient[testRoom1]; exists {
					t.Errorf("empty room %d not cleaned up", testRoom1)
				}
			},
		},
		{
			name:   "room not cleaned up with remaining clients",
			client: testClient1,
			setupFunc: func(m *MessageManager) {
				m.RegisterClient(testClient1, testRoom1)
				m.RegisterClient(testClient2, testRoom1)
			},
			wantErr: false,
			validateFunc: func(t *testing.T, m *MessageManager) {
				if _, exists := m.roomToClient[testRoom1]; !exists {
					t.Errorf("room %d cleaned up with remaining clients", testRoom1)
				}
				if clients := m.roomToClient[testRoom1]; clients != nil {
					if _, exists := clients[testClient2]; !exists {
						t.Errorf("remaining client %s not in room", testClient2)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := createTestManager()

			if tt.setupFunc != nil {
				tt.setupFunc(manager)
			}

			err := manager.UnregisterClient(tt.client)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, manager)
			}
		})
	}
}

func TestRouteMessage(t *testing.T) {
	tests := []struct {
		name              string
		sender            string
		content           string
		setupFunc         func(*MessageManager)
		wantErr           bool
		errContains       string
		expectedReceivers []string
		expectedContent   string
	}{
		{
			name:    "regular room message",
			sender:  testClient1,
			content: "Hello everyone",
			setupFunc: func(m *MessageManager) {
				m.RegisterClient(testClient1, testRoom1)
				m.RegisterClient(testClient2, testRoom1)
			},
			wantErr:           false,
			expectedReceivers: []string{testClient2},
			expectedContent:   "(room) client1: Hello everyone",
		},
		{
			name:    "all message",
			sender:  testClient1,
			content: "/all Hello everyone",
			setupFunc: func(m *MessageManager) {
				m.RegisterClient(testClient1, testRoom1)
				m.RegisterClient(testClient2, testRoom1)
				m.RegisterClient(testClient3, testRoom1)
			},
			wantErr:           false,
			expectedReceivers: []string{testClient2, testClient3},
			expectedContent:   "(all) client1: Hello everyone",
		},
		{
			name:    "whisper message",
			sender:  testClient1,
			content: "/client2 Secret message",
			setupFunc: func(m *MessageManager) {
				m.RegisterClient(testClient1, testRoom1)
				m.RegisterClient(testClient2, testRoom1)
			},
			wantErr:           false,
			expectedReceivers: []string{testClient2},
			expectedContent:   "(whisper) client1: Secret message",
		},
		{
			name:        "unregistered sender",
			sender:      testClient1,
			content:     "Hello",
			wantErr:     true,
			errContains: "not registered",
		},
		{
			name:    "whisper to non-existent user",
			sender:  testClient1,
			content: "/nonexistent Hello",
			setupFunc: func(m *MessageManager) {
				m.RegisterClient(testClient1, testRoom1)
			},
			wantErr:     true,
			errContains: "not in room",
		},
		{
			name:    "empty message",
			sender:  testClient1,
			content: "",
			setupFunc: func(m *MessageManager) {
				m.RegisterClient(testClient1, testRoom1)
			},
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:    "message too long",
			sender:  testClient1,
			content: strings.Repeat("a", testMaxSize+1),
			setupFunc: func(m *MessageManager) {
				m.RegisterClient(testClient1, testRoom1)
			},
			wantErr:     true,
			errContains: "too long",
		},
		{
			name:        "empty sender",
			sender:      "",
			content:     "Hello",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:    "single client in room",
			sender:  testClient1,
			content: "Hello",
			setupFunc: func(m *MessageManager) {
				m.RegisterClient(testClient1, testRoom1)
			},
			wantErr:           false,
			expectedReceivers: []string{}, // No one else to receive
			expectedContent:   "(room) client1: Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := createTestManager()

			if tt.setupFunc != nil {
				tt.setupFunc(manager)
			}

			receivers, content, err := manager.RouteMessage(tt.sender, tt.content)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Check receivers
				if len(receivers) != len(tt.expectedReceivers) {
					t.Errorf("receivers = %v, want %v", receivers, tt.expectedReceivers)
				} else {
					for _, expected := range tt.expectedReceivers {
						found := slices.Contains(receivers, expected)
						if !found {
							t.Errorf("expected receiver %s not found in %v", expected, receivers)
						}
					}
				}

				// Check content
				if content != tt.expectedContent {
					t.Errorf("content = %q, want %q", content, tt.expectedContent)
				}
			}
		})
	}
}

func TestConcurrentOperations(t *testing.T) {
	manager := createTestManager()

	// Test concurrent registrations
	var wg sync.WaitGroup
	numGoroutines := 10
	clientsPerGoroutine := 5

	// Concurrent registrations
	for i := range numGoroutines {
		wg.Add(1)
		go func(baseID int) {
			defer wg.Done()
			for j := range clientsPerGoroutine {
				clientID := baseID*clientsPerGoroutine + j
				roomID := clientID % 3 // Distribute across 3 rooms
				err := manager.RegisterClient(fmt.Sprintf("client%d", clientID), roomID)
				if err != nil {
					t.Errorf("concurrent registration failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all registrations succeeded
	totalClients := numGoroutines * clientsPerGoroutine
	if len(manager.userToRoom) != totalClients {
		t.Errorf("expected %d clients, got %d", totalClients, len(manager.userToRoom))
	}

	// Test concurrent message routing
	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(baseID int) {
			defer wg.Done()
			for j := range clientsPerGoroutine {
				clientID := baseID*clientsPerGoroutine + j
				clientName := fmt.Sprintf("client%d", clientID)
				_, _, err := manager.RouteMessage(clientName, "concurrent message")
				if err != nil {
					t.Errorf("concurrent message routing failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()
}

func BenchmarkRegisterClient(b *testing.B) {
	manager := createTestManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clientID := i % 1000 // Reuse clients to avoid unbounded growth
		roomID := i % 10
		_ = manager.RegisterClient(fmt.Sprintf("client%d", clientID), roomID)
	}
}

func BenchmarkRouteMessage(b *testing.B) {
	manager := createTestManager()

	// Setup: register clients
	for i := range 100 {
		manager.RegisterClient(fmt.Sprintf("client%d", i), i%5)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clientID := i % 100
		_, _, _ = manager.RouteMessage(fmt.Sprintf("client%d", clientID), "benchmark message")
	}
}
