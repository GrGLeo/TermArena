package event

import (
	"sync"

	"go.uber.org/zap"
)

// EventBroker is a struct that manages the lifecycle of events, including publishing, subscribing, and processing.
// It uses a queue to hold events, a map to store subscribers for different event types, and channels to handle responses.
type EventBroker struct {
	eventQueue      *Queue
	subscribers     map[string][]func(Message) Message
	responseChannel map[string]chan Message
	logger          *zap.SugaredLogger
	mu              sync.Mutex
}

// NewEventBroker initializes and returns a new EventBroker instance.
// It sets up the event queue, subscriber map, response channel map, and logger.
func NewEventBroker(logger *zap.SugaredLogger) *EventBroker {
	return &EventBroker{
		eventQueue:      NewQueue(),
		subscribers:     make(map[string][]func(Message) Message),
		responseChannel: make(map[string]chan Message),
		logger:          logger,
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

// ResponseChannel returns a channel for receiving responses for a specific event type.
// If the channel does not exist, it creates a new one.
func (eb *EventBroker) ResponseChannel(eventType string) chan Message {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	if _, ok := eb.responseChannel[eventType]; !ok {
		eb.responseChannel[eventType] = make(chan Message)
	}
	return eb.responseChannel[eventType]
}

// ProcessMessage continuously processes messages from the event queue.
// It retrieves messages from the queue, invokes the corresponding subscriber callbacks, and sends responses to the appropriate channels.
func (eb *EventBroker) ProcessMessage() {
	for {
		msg := eb.eventQueue.Dequeue()
		eventType := msg.Type()
		eb.logger.Infow("Processing message", "message", eventType)
		var respMsg Message
		if callbacks, ok := eb.subscribers[eventType]; ok {
			for _, callback := range callbacks {
				respMsg = callback(msg)
        if respMsg != nil {
				  eb.logger.Infow("Response message", "message", respMsg.Type())
        }
			}
		}

		// Use the ResponseChannel method to ensure the channel is created
		channel := eb.ResponseChannel(eventType)
		channel <- respMsg
	}
}

// Helper function to list available channels for debugging
func (eb *EventBroker) listAvailableChannels() []string {
	channels := make([]string, 0, len(eb.responseChannel))
	for eventType := range eb.responseChannel {
		channels = append(channels, eventType)
	}
	return channels
}
