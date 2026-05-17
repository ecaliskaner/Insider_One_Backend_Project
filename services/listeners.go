package services

import (
	"log"
)

// StartListeners initializes background workers to process events
func StartListeners(eb *EventBus, s *LeagueServiceImpl) {
	weekFinishedCh := eb.Subscribe("week_finished")

	go func() {
		for event := range weekFinishedCh {
			if week, ok := event.(int); ok {
				log.Printf("Listener: Week %d officially finished. All state rebuilt deterministically.", week)
			}
		}
	}()
}
