package services

import (
	"sync"

	"github.com/ecaliskaner/Insider_One_Backend_Project/models"
)

// MatchFinishedEvent is the payload for the event bus
type MatchFinishedEvent struct {
	HomeTeam  models.Team
	AwayTeam  models.Team
	HomeGoals int
	AwayGoals int
	MatchID   int
}

// EventBus implements a simple internal Pub/Sub system
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan interface{}
}

// NewEventBus initializes a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan interface{}),
	}
}

// Subscribe adds a channel to listen for a specific topic
func (eb *EventBus) Subscribe(topic string) <-chan interface{} {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	ch := make(chan interface{}, 100)
	eb.subscribers[topic] = append(eb.subscribers[topic], ch)
	return ch
}

// Publish sends an event to all subscribers of a topic asynchronously
func (eb *EventBus) Publish(topic string, data interface{}) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if chans, found := eb.subscribers[topic]; found {
		for _, ch := range chans {
			// Non-blocking send
			select {
			case ch <- data:
			default:
				// If channel is full, drop or log (simplification for this task)
			}
		}
	}
}
