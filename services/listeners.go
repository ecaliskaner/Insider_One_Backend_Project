package services

import (
	"log/slog"
)

// StartListeners initializes background workers for domain-event observability.
func StartListeners(eb *EventBus) {
	topics := []string{
		EventWeekPlayed,
		EventMatchEdited,
		EventRollbackCompleted,
		EventStandingsRebuilt,
		EventPredictionCacheInvalidated,
	}

	for _, topic := range topics {
		events := eb.Subscribe(topic)
		go func(topic string, events <-chan interface{}) {
			for event := range events {
				if domainEvent, ok := event.(DomainEvent); ok {
					slog.Info("domain event", slog.String("topic", topic), slog.Any("fields", domainEvent.Fields))
				}
			}
		}(topic, events)
	}
}
