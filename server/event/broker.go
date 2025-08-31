package event

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"maps"
)

// EventBroker is a struct that manages the lifecycle of events, including publishing, subscribing, and processing.
// It uses a queue to hold events and a map to store subscribers for different event types.
type EventBroker struct {
	eventQueue     *Queue
	subscribers    map[string][]func(Message) Message
	logger         *zap.SugaredLogger
	mu             sync.Mutex
	jobChannel     chan Message
	workerPoolSize int

	// Worker monitoring fields
	activeWorkers   int
	activeWorkersMu sync.RWMutex
	workerHealth    map[int]time.Time // Track last activity time per worker
	healthMu        sync.RWMutex
}

// NewEventBroker initializes and returns a new EventBroker instance.
// It sets up the event queue, subscriber map, and logger.
func NewEventBroker(logger *zap.SugaredLogger, workerPoolSize int) *EventBroker {
	return &EventBroker{
		eventQueue:     NewQueue(),
		subscribers:    make(map[string][]func(Message) Message),
		logger:         logger,
		jobChannel:     make(chan Message, workerPoolSize),
		workerPoolSize: workerPoolSize,
		workerHealth:   make(map[int]time.Time),
	}
}

// Publish adds a message to the event queue.
// This method is thread-safe and ensures that only one goroutine can access the queue at a time.
func (eb *EventBroker) Publish(msg Message) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.eventQueue.Enqueue(msg)
}

// Subscribe registers a callback function for a specific event type.
// This method allows subscribers to register interest in specific events and provide a callback to handle them.
func (eb *EventBroker) Subscribe(eventType string, callback func(Message) Message) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers[eventType] = append(eb.subscribers[eventType], callback)
}

// Start initializes the worker pool and begins processing messages.
func (eb *EventBroker) Start() {
	eb.logger.Info("[BROKER] Starting event broker")
	for i := range eb.workerPoolSize {
		go eb.worker(i)
	}
	go eb.dispatch()
}

// StartWithMonitoring initializes the worker pool, monitoring, and begins processing messages.
func (eb *EventBroker) StartWithMonitoring(ctx context.Context) {
	eb.logger.Info("[BROKER] Starting event broker with monitoring")
	for i := range eb.workerPoolSize {
		go eb.worker(i)
	}
	go eb.dispatch()
	go eb.monitorWorkers(ctx)
}

// dispatch is responsible for dequeuing messages and distributing them to workers.
func (eb *EventBroker) dispatch() {
	eb.logger.Info("[BROKER] Starting dispatcher")
	for {
		msg := eb.eventQueue.Dequeue()
		eb.jobChannel <- msg
	}
}

// worker processes messages from the job channel with timeout protection.
func (eb *EventBroker) worker(id int) {
	eb.logger.Infow("[BROKER] Starting worker", "id", id)

	// Initialize worker health tracking
	eb.healthMu.Lock()
	eb.workerHealth[id] = time.Now()
	eb.healthMu.Unlock()

	for msg := range eb.jobChannel {
		// Mark worker as active
		eb.activeWorkersMu.Lock()
		eb.activeWorkers++
		eb.activeWorkersMu.Unlock()

		// Update health timestamp
		eb.healthMu.Lock()
		eb.workerHealth[id] = time.Now()
		eb.healthMu.Unlock()

		eventType := msg.Type()
		eb.logger.Infow("[BROKER] Processing message", "worker_id", id, "message_type", eventType)

		// Process message with timeout protection
		respMsg := eb.processMessageWithTimeout(id, msg)

		// Send response back to caller
		if msg.ResponseChan() != nil {
			select {
			case msg.ResponseChan() <- respMsg:
				eb.logger.Infow("[BROKER] Response sent successfully", "worker_id", id, "message_type", eventType)
			case <-time.After(5 * time.Second):
				eb.logger.Errorw("[BROKER] Failed to send response - channel blocked", "worker_id", id, "message_type", eventType)
			}
		}

		// Mark worker as inactive
		eb.activeWorkersMu.Lock()
		eb.activeWorkers--
		eb.activeWorkersMu.Unlock()

		// Update health timestamp after processing
		eb.healthMu.Lock()
		eb.workerHealth[id] = time.Now()
		eb.healthMu.Unlock()
	}

	eb.logger.Infow("[BROKER] Worker shutting down", "id", id)
}

// processMessageWithTimeout processes a message with a 10-second timeout
func (eb *EventBroker) processMessageWithTimeout(workerID int, msg Message) Message {
	eventType := msg.Type()

	// Create a channel to receive the processing result
	resultChan := make(chan Message, 1)

	// Start processing in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				eb.logger.Errorw("[BROKER] Worker panic during message processing",
					"worker_id", workerID,
					"message_type", eventType,
					"panic", r)
				resultChan <- MessageErrorResponse{
					Error: "Worker panic during processing",
				}
			}
		}()

		var respMsg Message
		if callbacks, ok := eb.subscribers[eventType]; ok {
			for _, callback := range callbacks {
				respMsg = callback(msg)
				if respMsg != nil {
					eb.logger.Infow("[BROKER] Response message generated", "worker_id", workerID, "message_type", respMsg.Type())
				}
			}
		}
		resultChan <- respMsg
	}()

	// Wait for result with timeout
	select {
	case result := <-resultChan:
		return result

	case <-time.After(10 * time.Second):
		eb.logger.Errorw("[BROKER] Worker timeout - message processing took too long",
			"worker_id", workerID,
			"message_type", eventType,
			"timeout_seconds", 10)

		// Return timeout error response
		return MessageErrorResponse{
			Error: "Message processing timeout after 10 seconds",
		}
	}
}

// monitorWorkers monitors the health and availability of workers
func (eb *EventBroker) monitorWorkers(ctx context.Context) {
	eb.logger.Info("[BROKER] Starting worker health monitoring")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			eb.logger.Info("[BROKER] Worker monitoring shutting down")
			return

		case <-ticker.C:
			eb.performHealthCheck()
		}
	}
}

// performHealthCheck checks worker health and logs warnings if needed
func (eb *EventBroker) performHealthCheck() {
	activeWorkers := eb.getActiveWorkerCount()
	availableCapacity := eb.workerPoolSize - activeWorkers
	queueSize := len(eb.jobChannel)

	// Log current status
	eb.logger.Infow("[BROKER] Worker health check",
		"active_workers", activeWorkers,
		"total_workers", eb.workerPoolSize,
		"available_capacity", availableCapacity,
		"queue_size", queueSize,
		"queue_utilization", float64(queueSize)/float64(cap(eb.jobChannel))*100,
	)

	// Check for potential issues
	if availableCapacity == 0 {
		eb.logger.Warnw("[BROKER] All workers are active - potential bottleneck",
			"active_workers", activeWorkers,
			"queue_size", queueSize,
		)
	}

	if availableCapacity < eb.workerPoolSize/4 {
		eb.logger.Warnw("[BROKER] Low worker availability detected",
			"active_workers", activeWorkers,
			"available_capacity", availableCapacity,
			"threshold", eb.workerPoolSize/4,
		)
	}

	if queueSize > eb.workerPoolSize*2 {
		eb.logger.Errorw("[BROKER] Large message queue detected - workers may be stuck",
			"queue_size", queueSize,
			"worker_pool_size", eb.workerPoolSize,
		)
	}

	// Check for stuck workers (no activity for more than 2 minutes)
	eb.checkForStuckWorkers()
}

// checkForStuckWorkers identifies workers that haven't shown activity recently
func (eb *EventBroker) checkForStuckWorkers() {
	stuckThreshold := 2 * time.Minute
	now := time.Now()

	eb.healthMu.RLock()
	stuckWorkers := []int{}
	for workerID, lastActivity := range eb.workerHealth {
		if now.Sub(lastActivity) > stuckThreshold {
			stuckWorkers = append(stuckWorkers, workerID)
		}
	}
	eb.healthMu.RUnlock()

	if len(stuckWorkers) > 0 {
		eb.logger.Errorw("[BROKER] Stuck workers detected",
			"stuck_worker_ids", stuckWorkers,
			"stuck_count", len(stuckWorkers),
			"threshold_minutes", stuckThreshold.Minutes(),
		)

		// Log detailed information about stuck workers
		for _, workerID := range stuckWorkers {
			eb.healthMu.RLock()
			lastActivity := eb.workerHealth[workerID]
			eb.healthMu.RUnlock()

			eb.logger.Errorw("[BROKER] Stuck worker details",
				"worker_id", workerID,
				"last_activity", lastActivity,
				"time_since_activity", now.Sub(lastActivity),
			)
		}
	}
}

// getActiveWorkerCount returns the current number of active workers
func (eb *EventBroker) getActiveWorkerCount() int {
	eb.activeWorkersMu.RLock()
	defer eb.activeWorkersMu.RUnlock()
	return eb.activeWorkers
}

// GetWorkerStats returns detailed statistics about worker health
func (eb *EventBroker) GetWorkerStats() map[string]any {
	activeWorkers := eb.getActiveWorkerCount()
	queueSize := len(eb.jobChannel)
	queueCapacity := cap(eb.jobChannel)

	stats := map[string]any{
		"active_workers":     activeWorkers,
		"total_workers":      eb.workerPoolSize,
		"available_capacity": eb.workerPoolSize - activeWorkers,
		"queue_size":         queueSize,
		"queue_capacity":     queueCapacity,
		"queue_utilization":  float64(queueSize) / float64(queueCapacity) * 100,
	}

	// Add worker health information
	eb.healthMu.RLock()
	workerHealth := make(map[int]time.Time)
	maps.Copy(workerHealth, eb.workerHealth)
	eb.healthMu.RUnlock()

	stats["worker_health"] = workerHealth
	stats["last_check"] = time.Now()

	return stats
}
