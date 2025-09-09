package event

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestWorkerMonitoring(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError + 1}))
	broker := NewEventBroker(logger, 3)

	// Test initial state
	stats := broker.GetWorkerStats()
	if stats["total_workers"] != 3 {
		t.Errorf("Expected 3 total workers, got %v", stats["total_workers"])
	}
	if stats["active_workers"] != 0 {
		t.Errorf("Expected 0 active workers initially, got %v", stats["active_workers"])
	}
}

func TestWorkerHealthTracking(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError + 1}))
	broker := NewEventBroker(logger, 2)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start monitoring
	go broker.monitorWorkers(ctx)

	// Let monitoring run briefly
	time.Sleep(100 * time.Millisecond)

	// Check that monitoring doesn't panic and stats are accessible
	stats := broker.GetWorkerStats()
	if stats == nil {
		t.Error("Expected stats to be non-nil")
	}

	// Verify expected fields exist
	expectedFields := []string{"active_workers", "total_workers", "available_capacity", "queue_size", "worker_health"}
	for _, field := range expectedFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("Expected field %s to exist in stats", field)
		}
	}
}

func TestStuckWorkerDetection(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError + 1}))
	broker := NewEventBroker(logger, 1)

	// Manually set a worker as having old activity
	broker.healthMu.Lock()
	broker.workerHealth[0] = time.Now().Add(-3 * time.Minute) // 3 minutes ago
	broker.healthMu.Unlock()

	// Run stuck worker check
	broker.checkForStuckWorkers()

	// Verify the stuck worker is detected (this will log warnings/errors)
	stats := broker.GetWorkerStats()
	if workerHealth, ok := stats["worker_health"].(map[int]time.Time); ok {
		if lastActivity, exists := workerHealth[0]; exists {
			if time.Since(lastActivity) < 3*time.Minute {
				t.Error("Expected worker 0 to be marked as having old activity")
			}
		}
	}
}

func TestWorkerTimeout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError + 1}))
	broker := NewEventBroker(logger, 1)

	// Create a slow message handler that takes more than 10 seconds
	slowHandler := func(msg Message) Message {
		time.Sleep(12 * time.Second) // Sleep longer than timeout
		return nil
	}

	// Subscribe to test messages
	broker.Subscribe("slow_test", slowHandler)

	// Create a test message
	testMsg := &testMessage{
		msgType:    "slow_test",
		responseCh: make(chan Message, 1),
	}

	// Start the broker
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	go broker.StartWithMonitoring(ctx)

	// Give workers time to start
	time.Sleep(100 * time.Millisecond)

	// Publish the slow message
	broker.Publish(testMsg)

	// Wait for response with timeout
	select {
	case response := <-testMsg.responseCh:
		// Should receive a timeout error
		if errorResp, ok := response.(MessageErrorResponse); ok {
			if errorResp.Error != "Message processing timeout after 10 seconds" {
				t.Errorf("Expected timeout error, got: %s", errorResp.Error)
			}
		} else {
			t.Errorf("Expected MessageErrorResponse, got: %T", response)
		}
	case <-time.After(15 * time.Second):
		t.Error("Test timed out waiting for response")
	}
}

// testMessage is a simple test message implementation
type testMessage struct {
	msgType    string
	responseCh chan Message
}

func (m *testMessage) Type() string {
	return m.msgType
}

func (m *testMessage) Validate() error {
	return nil
}

func (m *testMessage) ResponseChan() chan Message {
	return m.responseCh
}
