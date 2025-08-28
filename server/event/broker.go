package event

import (
	"sync"

	"go.uber.org/zap"
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
	}
}

// Publish adds a message to the event queue.
// This method is thread-safe and ensures that only one goroutine can access the queue at a time.
func (eb *EventBroker) Publish(msg Message) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
  eb.logger.Infow("Found message","message", msg.Type())
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
	eb.logger.Info("Starting event broker")
	for i := range eb.workerPoolSize {
		go eb.worker(i)
	}
	go eb.dispatch()
}

// dispatch is responsible for dequeuing messages and distributing them to workers.
func (eb *EventBroker) dispatch() {
	eb.logger.Info("Starting dispatcher")
	for {
		msg := eb.eventQueue.Dequeue()
		eb.jobChannel <- msg
	}
}

// worker processes messages from the job channel.
func (eb *EventBroker) worker(id int) {
	eb.logger.Infow("Starting worker", "id", id)
	for msg := range eb.jobChannel {
		eventType := msg.Type()
		eb.logger.Infow("Processing message", "worker_id", id, "message_type", eventType)
		var respMsg Message
		if callbacks, ok := eb.subscribers[eventType]; ok {
			for _, callback := range callbacks {
				respMsg = callback(msg)
				if respMsg != nil {
					eb.logger.Infow("Response message", "worker_id", id, "message_type", respMsg.Type())
				}
			}
		}
		if msg.ResponseChan() != nil {
			msg.ResponseChan() <- respMsg
		}
	}
}

