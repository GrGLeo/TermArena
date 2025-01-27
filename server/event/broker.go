package event

import (
	"sync"

	"go.uber.org/zap"
)

type EventBroker struct {
	eventQueue      *Queue
	subscribers     map[string][]func(Message) Message
	responseChannel map[string]chan Message
	logger          *zap.SugaredLogger
	mu              sync.Mutex
}

func NewEventBroker(logger *zap.SugaredLogger) *EventBroker {
	return &EventBroker{
		eventQueue:      NewQueue(),
		subscribers:     make(map[string][]func(Message) Message),
		responseChannel: make(map[string]chan Message),
		logger:          logger,
	}
}

func (eb *EventBroker) Publish(msg Message) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.eventQueue.Enqueue(msg)
}

func (eb *EventBroker) Subscribe(eventType string, callback func(Message) Message) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers[eventType] = append(eb.subscribers[eventType], callback)
}

func (eb *EventBroker) ResponseChannel(eventType string) chan Message {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	if _, ok := eb.responseChannel[eventType]; !ok {
		eb.responseChannel[eventType] = make(chan Message)
	}
	return eb.responseChannel[eventType]
}

func (eb *EventBroker) ProcessMessage() {
	for {
		eb.mu.Lock()
		msg := eb.eventQueue.Dequeue()
		if msg == nil {
			eb.mu.Unlock()
			continue
		}
		eventType := msg.Type()
		eb.logger.Infow("Processing message", "message", eventType)
		var respMsg Message
		if callbacks, ok := eb.subscribers[eventType]; ok {
			for _, callback := range callbacks {
				respMsg = callback(msg)
			}
		}
		eb.mu.Unlock()

		if channel, ok := eb.responseChannel[eventType]; ok {
			channel <- respMsg
		}
	}
}
